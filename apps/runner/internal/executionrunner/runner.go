package executionrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	ServerURL         string
	BearerToken       string
	ProjectID         string
	RepoBindingID     string
	RunnerID          string
	WorkspaceRef      string
	WorkspaceRoot     string
	CommitSHA         string
	BaselineID        string
	OverlayID         string
	SubmitReceipt     bool
	BuiltinDiagnostic bool
	ProjectProbe      bool
	ProjectTest       bool
	PollInterval      time.Duration
	LeaseTTLSeconds   int
	Once              bool
	HTTPClient        *http.Client
	LogWriter         io.Writer
}

type Runner struct {
	config Config
	client *apiClient
	logger *log.Logger
}

type StepResult string

const (
	StepNoWork           StepResult = "no_work"
	StepRunStarted       StepResult = "run_started"
	StepReceiptSubmitted StepResult = "receipt_submitted"
	StepLeaseExpired     StepResult = "lease_expired"
	StepInvalidLease     StepResult = "invalid_lease"
	StepAlreadyHandled   StepResult = "already_handled"
)

func Run(ctx context.Context, config Config) error {
	runner, err := NewRunner(config)
	if err != nil {
		return err
	}
	return runner.Run(ctx)
}

func ReportCapabilities(ctx context.Context, config Config) error {
	runner, err := NewRunner(config)
	if err != nil {
		return err
	}
	report, err := runner.client.submitRunnerCapabilityReport(ctx, runnerCapabilityReportRequest{
		RunnerID:                        runner.config.RunnerID,
		ProjectID:                       runner.config.ProjectID,
		RepoBindingID:                   runner.config.RepoBindingID,
		NetworkIsolationDeclared:        false,
		WorkspaceWriteIsolationDeclared: false,
		ProcessTreeControlDeclared:      false,
		StdoutStderrPolicyDeclared:      false,
		ArtifactPolicyDeclared:          false,
		TrustState:                      "self_declared_untrusted",
	})
	if err != nil {
		return err
	}
	runner.logger.Printf("submitted runner capability report report_id=%s runner_id=%s project_id=%s repo_binding_id=%s trust_state=%s", report.ID, report.RunnerID, report.ProjectID, report.RepoBindingID, report.TrustState)
	return nil
}

func NewRunner(config Config) (*Runner, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	client, err := newAPIClient(config.ServerURL, config.BearerToken, config.HTTPClient)
	if err != nil {
		return nil, err
	}
	if config.PollInterval == 0 {
		config.PollInterval = 10 * time.Second
	}
	if config.LeaseTTLSeconds == 0 {
		config.LeaseTTLSeconds = 900
	}
	if config.LogWriter == nil {
		config.LogWriter = io.Discard
	}
	return &Runner{
		config: config,
		client: client,
		logger: log.New(config.LogWriter, "", 0),
	}, nil
}

func (r *Runner) Run(ctx context.Context) error {
	for {
		result, err := r.Step(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		if r.config.Once {
			return nil
		}
		if result == StepNoWork {
			if err := sleepContext(ctx, r.config.PollInterval); err != nil {
				return err
			}
		}
	}
}

func (r *Runner) Step(ctx context.Context) (StepResult, error) {
	lease, ok, err := r.client.acquireLease(ctx, executionLeaseCreateRequest{
		ProjectID:     r.config.ProjectID,
		RepoBindingID: r.config.RepoBindingID,
		RunnerID:      r.config.RunnerID,
		TTLSeconds:    r.config.LeaseTTLSeconds,
	})
	if err != nil {
		return "", err
	}
	if !ok {
		r.logger.Printf("no execution work available")
		return StepNoWork, nil
	}
	r.logger.Printf("acquired execution lease execution_job_id=%s task_id=%s", lease.ExecutionJobID, lease.TaskID)
	if err := r.validateLease(lease); err != nil {
		return "", err
	}
	run, err := r.client.startRun(ctx, lease.ExecutionJobID, runStartRequest{
		LeaseID:    lease.ID,
		LeaseToken: lease.LeaseToken,
		RunnerID:   r.config.RunnerID,
	})
	if err != nil {
		switch apiErrorCode(err) {
		case "lease_expired":
			r.logger.Printf("execution lease expired before run start execution_job_id=%s task_id=%s; abandoning start", lease.ExecutionJobID, lease.TaskID)
			return StepLeaseExpired, nil
		case "invalid_lease":
			r.logger.Printf("execution lease rejected before run start execution_job_id=%s task_id=%s; abandoning start", lease.ExecutionJobID, lease.TaskID)
			return StepInvalidLease, nil
		case "already_started":
			r.logger.Printf("execution run already exists execution_job_id=%s task_id=%s; abandoning start", lease.ExecutionJobID, lease.TaskID)
			return StepAlreadyHandled, nil
		default:
			return "", err
		}
	}
	r.logger.Printf("started run run_id=%s execution_job_id=%s task_id=%s", run.ID, run.ExecutionJobID, run.TaskID)
	if r.config.ProjectProbe {
		plan, err := r.client.createCommandPlan(ctx, run.ID, executionCommandPlanRequest{
			ProjectID:     r.config.ProjectID,
			RepoBindingID: r.config.RepoBindingID,
			CommandKind:   "project_probe",
			Action:        "detect_declared_test_targets",
		})
		if err != nil {
			return "", err
		}
		if err := r.validateProjectProbePlan(plan, lease, run); err != nil {
			return "", err
		}
		startedAt := time.Now().UTC()
		metadata, err := detectDeclaredTestTargets(r.config.WorkspaceRoot, plan)
		if err != nil {
			return "", err
		}
		finishedAt := time.Now().UTC()
		receipt, err := r.client.submitReceipt(ctx, run.ID, executionReceiptRequest{
			ExecutionJobID:       run.ExecutionJobID,
			LeaseID:              lease.ID,
			LeaseToken:           lease.LeaseToken,
			RunnerID:             r.config.RunnerID,
			WorkspaceRef:         r.config.WorkspaceRef,
			CommitSHA:            r.config.CommitSHA,
			BaselineID:           r.config.BaselineID,
			OverlayID:            r.config.OverlayID,
			ExecutionMode:        "project_probe",
			CommandPlanID:        plan.ID,
			CommandKind:          "project_probe",
			Action:               "detect_declared_test_targets",
			ProcessStatus:        "metadata_only",
			ArtifactRefs:         []string{},
			ChangedPathsSummary:  []string{},
			RawSourceUploaded:    false,
			RunnerStartedAt:      &startedAt,
			RunnerFinishedAt:     &finishedAt,
			ProjectProbeMetadata: &metadata,
		})
		if err != nil {
			switch apiErrorCode(err) {
			case "lease_expired":
				r.logger.Printf("execution lease expired before project probe receipt execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepLeaseExpired, nil
			case "invalid_lease":
				r.logger.Printf("execution lease rejected before project probe receipt execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepInvalidLease, nil
			default:
				return "", err
			}
		}
		r.logger.Printf("submitted project probe execution receipt receipt_id=%s run_id=%s execution_job_id=%s command_plan_id=%s", receipt.ID, receipt.RunID, receipt.ExecutionJobID, receipt.CommandPlanID)
		return StepReceiptSubmitted, nil
	}
	if r.config.ProjectTest {
		plan, err := r.client.getCommandPlan(ctx, run.ID, "project_test", "run_declared_test_target")
		if err != nil {
			return "", err
		}
		if err := r.validateProjectTestPlan(plan, lease, run); err != nil {
			return "", err
		}
		startedAt := time.Now().UTC()
		result := rejectProjectTestExecution()
		finishedAt := time.Now().UTC()
		receipt, err := r.client.submitReceipt(ctx, run.ID, executionReceiptRequest{
			ExecutionJobID:      run.ExecutionJobID,
			LeaseID:             lease.ID,
			LeaseToken:          lease.LeaseToken,
			RunnerID:            r.config.RunnerID,
			WorkspaceRef:        r.config.WorkspaceRef,
			CommitSHA:           r.config.CommitSHA,
			BaselineID:          r.config.BaselineID,
			OverlayID:           r.config.OverlayID,
			ExecutionMode:       "project_test",
			CommandPlanID:       plan.ID,
			CommandKind:         "project_test",
			Action:              "run_declared_test_target",
			ProcessStatus:       result.ProcessStatus,
			ExitCode:            result.ExitCode,
			ArtifactRefs:        []string{},
			ChangedPathsSummary: []string{},
			RawSourceUploaded:   false,
			RunnerStartedAt:     &startedAt,
			RunnerFinishedAt:    &finishedAt,
			EnforcementReport:   &result.EnforcementReport,
		})
		if err != nil {
			switch apiErrorCode(err) {
			case "lease_expired":
				r.logger.Printf("execution lease expired before project test receipt execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepLeaseExpired, nil
			case "invalid_lease":
				r.logger.Printf("execution lease rejected before project test receipt execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepInvalidLease, nil
			default:
				return "", err
			}
		}
		r.logger.Printf("submitted project test execution receipt receipt_id=%s run_id=%s execution_job_id=%s command_plan_id=%s process_status=%s", receipt.ID, receipt.RunID, receipt.ExecutionJobID, receipt.CommandPlanID, result.ProcessStatus)
		return StepReceiptSubmitted, nil
	}
	if r.config.BuiltinDiagnostic {
		plan, err := r.client.createCommandPlan(ctx, run.ID, executionCommandPlanRequest{
			ProjectID:     r.config.ProjectID,
			RepoBindingID: r.config.RepoBindingID,
			CommandKind:   "builtin_diagnostic",
			Action:        "workspace_status",
		})
		if err != nil {
			return "", err
		}
		if err := r.validateBuiltinDiagnosticPlan(plan, lease, run); err != nil {
			return "", err
		}
		startedAt := time.Now().UTC()
		finishedAt := startedAt
		receipt, err := r.client.submitReceipt(ctx, run.ID, executionReceiptRequest{
			ExecutionJobID:      run.ExecutionJobID,
			LeaseID:             lease.ID,
			LeaseToken:          lease.LeaseToken,
			RunnerID:            r.config.RunnerID,
			WorkspaceRef:        r.config.WorkspaceRef,
			CommitSHA:           r.config.CommitSHA,
			BaselineID:          r.config.BaselineID,
			OverlayID:           r.config.OverlayID,
			ExecutionMode:       "builtin_diagnostic",
			CommandPlanID:       plan.ID,
			CommandKind:         "builtin_diagnostic",
			Action:              "workspace_status",
			ProcessStatus:       "metadata_only",
			ArtifactRefs:        []string{},
			ChangedPathsSummary: []string{},
			RawSourceUploaded:   false,
			RunnerStartedAt:     &startedAt,
			RunnerFinishedAt:    &finishedAt,
		})
		if err != nil {
			switch apiErrorCode(err) {
			case "lease_expired":
				r.logger.Printf("execution lease expired before builtin diagnostic receipt execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepLeaseExpired, nil
			case "invalid_lease":
				r.logger.Printf("execution lease rejected before builtin diagnostic receipt execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepInvalidLease, nil
			default:
				return "", err
			}
		}
		r.logger.Printf("submitted builtin diagnostic execution receipt receipt_id=%s run_id=%s execution_job_id=%s command_plan_id=%s", receipt.ID, receipt.RunID, receipt.ExecutionJobID, receipt.CommandPlanID)
		return StepReceiptSubmitted, nil
	}
	if r.config.SubmitReceipt {
		receipt, err := r.client.submitReceipt(ctx, run.ID, executionReceiptRequest{
			ExecutionJobID:      run.ExecutionJobID,
			LeaseID:             lease.ID,
			LeaseToken:          lease.LeaseToken,
			RunnerID:            r.config.RunnerID,
			WorkspaceRef:        r.config.WorkspaceRef,
			CommitSHA:           r.config.CommitSHA,
			BaselineID:          r.config.BaselineID,
			OverlayID:           r.config.OverlayID,
			ExecutionMode:       "no_command",
			ProcessStatus:       "not_executed",
			ArtifactRefs:        []string{},
			ChangedPathsSummary: []string{},
			RawSourceUploaded:   false,
		})
		if err != nil {
			switch apiErrorCode(err) {
			case "lease_expired":
				r.logger.Printf("execution lease expired before receipt submission execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepLeaseExpired, nil
			case "invalid_lease":
				r.logger.Printf("execution lease rejected before receipt submission execution_job_id=%s task_id=%s; abandoning receipt", lease.ExecutionJobID, lease.TaskID)
				return StepInvalidLease, nil
			default:
				return "", err
			}
		}
		r.logger.Printf("submitted no-command execution receipt receipt_id=%s run_id=%s execution_job_id=%s", receipt.ID, receipt.RunID, receipt.ExecutionJobID)
		return StepReceiptSubmitted, nil
	}
	return StepRunStarted, nil
}

func (r *Runner) validateLease(lease executionLease) error {
	if strings.TrimSpace(lease.RepoBindingID) == "" {
		return errors.New("execution lease repo_binding_id is required")
	}
	if lease.RepoBindingID != r.config.RepoBindingID {
		return fmt.Errorf("execution lease repo_binding_id %q does not match runner scope %q", lease.RepoBindingID, r.config.RepoBindingID)
	}
	if strings.TrimSpace(lease.ExecutionJob.ID) != "" && lease.ExecutionJob.ID != lease.ExecutionJobID {
		return fmt.Errorf("execution lease job id %q does not match nested job id %q", lease.ExecutionJobID, lease.ExecutionJob.ID)
	}
	if strings.TrimSpace(lease.ExecutionJob.TaskID) != "" && lease.ExecutionJob.TaskID != lease.TaskID {
		return fmt.Errorf("execution lease task id %q does not match nested job task id %q", lease.TaskID, lease.ExecutionJob.TaskID)
	}
	if strings.TrimSpace(lease.ExecutionJob.RepoBindingID) != "" && lease.ExecutionJob.RepoBindingID != r.config.RepoBindingID {
		return fmt.Errorf("execution lease nested repo_binding_id %q does not match runner scope %q", lease.ExecutionJob.RepoBindingID, r.config.RepoBindingID)
	}
	return nil
}

func (r *Runner) validateBuiltinDiagnosticPlan(plan executionCommandPlan, lease executionLease, run runStarted) error {
	if err := r.validateCommandPlanTrace(plan, lease, run); err != nil {
		return err
	}
	if plan.CommandKind != "builtin_diagnostic" {
		return fmt.Errorf("unsupported execution command kind %q", plan.CommandKind)
	}
	if plan.Action != "workspace_status" {
		return fmt.Errorf("unsupported execution command action %q", plan.Action)
	}
	if plan.ShellAllowed {
		return errors.New("execution command plan unexpectedly allows shell")
	}
	if len(plan.Argv) != 0 {
		return fmt.Errorf("execution command plan argv must be empty for builtin diagnostic, got %d arguments", len(plan.Argv))
	}
	if plan.WorkingDirectory != "." {
		return fmt.Errorf("execution command plan working directory %q is outside builtin diagnostic scope", plan.WorkingDirectory)
	}
	if len(plan.PathScope) != 1 || plan.PathScope[0] != "." {
		return fmt.Errorf("execution command plan path scope %#v is outside builtin diagnostic scope", plan.PathScope)
	}
	if plan.TimeoutSeconds != 30 || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 {
		return fmt.Errorf("execution command plan output/timeout policy = %d/%d/%d, want 30/0/0", plan.TimeoutSeconds, plan.MaxStdoutBytes, plan.MaxStderrBytes)
	}
	if len(plan.AllowedArtifactKinds) != 0 {
		return fmt.Errorf("execution command plan allowed artifacts = %#v, want none", plan.AllowedArtifactKinds)
	}
	if plan.RawSourceUploadAllowed {
		return errors.New("execution command plan unexpectedly allows raw source upload")
	}
	if plan.State != "planned" {
		return fmt.Errorf("execution command plan state %q is not planned", plan.State)
	}
	return nil
}

func (r *Runner) validateProjectProbePlan(plan executionCommandPlan, lease executionLease, run runStarted) error {
	if err := r.validateCommandPlanTrace(plan, lease, run); err != nil {
		return err
	}
	if plan.CommandKind != "project_probe" {
		return fmt.Errorf("unsupported execution command kind %q", plan.CommandKind)
	}
	if plan.Action != "detect_declared_test_targets" {
		return fmt.Errorf("unsupported execution command action %q", plan.Action)
	}
	if plan.ShellAllowed {
		return errors.New("execution command plan unexpectedly allows shell")
	}
	if len(plan.Argv) != 0 {
		return fmt.Errorf("execution command plan argv must be empty for project probe, got %d arguments", len(plan.Argv))
	}
	if plan.WorkingDirectory != "." {
		return fmt.Errorf("execution command plan working directory %q is outside project probe scope", plan.WorkingDirectory)
	}
	if len(plan.PathScope) != 1 || plan.PathScope[0] != "." {
		return fmt.Errorf("execution command plan path scope %#v is outside project probe scope", plan.PathScope)
	}
	if plan.TimeoutSeconds != 30 || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 {
		return fmt.Errorf("execution command plan output/timeout policy = %d/%d/%d, want 30/0/0", plan.TimeoutSeconds, plan.MaxStdoutBytes, plan.MaxStderrBytes)
	}
	if len(plan.AllowedArtifactKinds) != 0 {
		return fmt.Errorf("execution command plan allowed artifacts = %#v, want none", plan.AllowedArtifactKinds)
	}
	if plan.RawSourceUploadAllowed {
		return errors.New("execution command plan unexpectedly allows raw source upload")
	}
	if plan.State != "planned" {
		return fmt.Errorf("execution command plan state %q is not planned", plan.State)
	}
	return nil
}

func (r *Runner) validateCommandPlanTrace(plan executionCommandPlan, lease executionLease, run runStarted) error {
	if plan.ProjectID != r.config.ProjectID {
		return fmt.Errorf("execution command plan project_id %q does not match runner scope %q", plan.ProjectID, r.config.ProjectID)
	}
	if plan.RepoBindingID != r.config.RepoBindingID {
		return fmt.Errorf("execution command plan repo_binding_id %q does not match runner scope %q", plan.RepoBindingID, r.config.RepoBindingID)
	}
	if plan.ExecutionJobID != lease.ExecutionJobID || plan.ExecutionJobID != run.ExecutionJobID {
		return fmt.Errorf("execution command plan job id %q does not match leased/run job %q/%q", plan.ExecutionJobID, lease.ExecutionJobID, run.ExecutionJobID)
	}
	if plan.RunID != run.ID {
		return fmt.Errorf("execution command plan run id %q does not match run %q", plan.RunID, run.ID)
	}
	if plan.TaskID != lease.TaskID || plan.TaskID != run.TaskID {
		return fmt.Errorf("execution command plan task id %q does not match leased/run task %q/%q", plan.TaskID, lease.TaskID, run.TaskID)
	}
	if plan.CheckoutReceiptID != lease.CheckoutReceiptID || plan.CheckoutReceiptID != run.CheckoutReceiptID {
		return fmt.Errorf("execution command plan checkout receipt id %q does not match leased/run receipt %q/%q", plan.CheckoutReceiptID, lease.CheckoutReceiptID, run.CheckoutReceiptID)
	}
	return nil
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.ServerURL) == "" {
		return errors.New("server url is required")
	}
	if strings.TrimSpace(config.BearerToken) == "" {
		return errors.New("runner bearer token is required")
	}
	if strings.TrimSpace(config.ProjectID) == "" {
		return errors.New("project id is required")
	}
	if strings.TrimSpace(config.RepoBindingID) == "" {
		return errors.New("repo binding id is required")
	}
	if strings.TrimSpace(config.RunnerID) == "" {
		return errors.New("runner id is required")
	}
	activeReceiptModes := 0
	for _, enabled := range []bool{config.SubmitReceipt, config.BuiltinDiagnostic, config.ProjectProbe, config.ProjectTest} {
		if enabled {
			activeReceiptModes++
		}
	}
	if activeReceiptModes > 1 {
		return errors.New("execution receipt modes are mutually exclusive")
	}
	if config.SubmitReceipt || config.BuiltinDiagnostic || config.ProjectProbe || config.ProjectTest {
		if strings.TrimSpace(config.WorkspaceRef) == "" {
			return errors.New("workspace ref is required for execution receipt mode")
		}
		if strings.TrimSpace(config.CommitSHA) == "" {
			return errors.New("commit sha is required for execution receipt mode")
		}
	}
	if config.ProjectProbe && strings.TrimSpace(config.WorkspaceRoot) == "" {
		return errors.New("workspace root is required for project probe mode")
	}
	if config.ProjectTest && strings.TrimSpace(config.WorkspaceRoot) == "" {
		return errors.New("workspace root is required for project test mode")
	}
	if config.PollInterval < 0 {
		return errors.New("poll interval must be non-negative")
	}
	if config.LeaseTTLSeconds < 0 {
		return errors.New("lease ttl seconds must be non-negative")
	}
	return nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil
	case <-timer.C:
		return nil
	}
}
