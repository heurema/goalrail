package checkoutrunner

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
	ServerURL       string
	BearerToken     string
	ProjectID       string
	RepoBindingID   string
	CheckoutJobID   string
	RunnerID        string
	WorkspaceRef    string
	CommitSHA       string
	BaselineID      string
	OverlayID       string
	Dirty           bool
	Partial         bool
	PartialReasons  []string
	PollInterval    time.Duration
	LeaseTTLSeconds int
	Once            bool
	HTTPClient      *http.Client
	LogWriter       io.Writer
}

type Runner struct {
	config Config
	client *apiClient
	logger *log.Logger
}

type StepResult string

const (
	StepNoWork         StepResult = "no_work"
	StepReceipt        StepResult = "receipt_submitted"
	StepLeaseExpired   StepResult = "lease_expired"
	StepInvalidLease   StepResult = "invalid_lease"
	StepAlreadyHandled StepResult = "already_handled"
)

func Run(ctx context.Context, config Config) error {
	runner, err := NewRunner(config)
	if err != nil {
		return err
	}
	return runner.Run(ctx)
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
	lease, ok, err := r.client.acquireLease(ctx, checkoutLeaseCreateRequest{
		ProjectID:     r.config.ProjectID,
		RepoBindingID: r.config.RepoBindingID,
		CheckoutJobID: strings.TrimSpace(r.config.CheckoutJobID),
		RunnerID:      r.config.RunnerID,
		TTLSeconds:    r.config.LeaseTTLSeconds,
	})
	if err != nil {
		return "", err
	}
	if !ok {
		r.logger.Printf("no checkout work available")
		return StepNoWork, nil
	}
	r.logger.Printf("acquired checkout lease job_id=%s task_id=%s", lease.JobID, lease.TaskID)

	receiptRequest, err := r.receiptRequestForLease(lease)
	if err != nil {
		return "", err
	}
	receipt, err := r.client.submitReceipt(ctx, lease.JobID, receiptRequest)
	if err != nil {
		switch apiErrorCode(err) {
		case "lease_expired":
			r.logger.Printf("checkout lease expired before receipt submission job_id=%s task_id=%s; abandoning local receipt", lease.JobID, lease.TaskID)
			return StepLeaseExpired, nil
		case "invalid_lease":
			r.logger.Printf("checkout lease rejected before receipt submission job_id=%s task_id=%s; abandoning local receipt", lease.JobID, lease.TaskID)
			return StepInvalidLease, nil
		case "already_receipted":
			r.logger.Printf("checkout receipt already exists job_id=%s task_id=%s; abandoning local receipt", lease.JobID, lease.TaskID)
			return StepAlreadyHandled, nil
		default:
			return "", err
		}
	}
	r.logger.Printf("submitted checkout receipt receipt_id=%s job_id=%s task_id=%s", receipt.ID, receipt.JobID, receipt.TaskID)
	return StepReceipt, nil
}

func (r *Runner) receiptRequestForLease(lease checkoutLease) (checkoutReceiptSubmitRequest, error) {
	if err := r.validateLeaseInstruction(lease); err != nil {
		return checkoutReceiptSubmitRequest{}, err
	}
	return checkoutReceiptSubmitRequest{
		LeaseToken:        lease.LeaseToken,
		RunnerID:          r.config.RunnerID,
		WorkspaceRef:      workspaceRefForLease(r.config.WorkspaceRef, lease),
		CommitSHA:         r.config.CommitSHA,
		BaselineID:        r.config.BaselineID,
		OverlayID:         r.config.OverlayID,
		Dirty:             r.config.Dirty,
		Partial:           r.config.Partial,
		PartialReasons:    cloneNonBlankStrings(r.config.PartialReasons),
		RawSourceUploaded: false,
	}, nil
}

func (r *Runner) validateLeaseInstruction(lease checkoutLease) error {
	instruction := lease.Instruction
	if strings.TrimSpace(instruction.JobID) != "" && instruction.JobID != lease.JobID {
		return fmt.Errorf("checkout lease instruction job_id %q does not match lease job_id %q", instruction.JobID, lease.JobID)
	}
	if strings.TrimSpace(instruction.TaskID) != "" && instruction.TaskID != lease.TaskID {
		return fmt.Errorf("checkout lease instruction task_id %q does not match lease task_id %q", instruction.TaskID, lease.TaskID)
	}
	if strings.TrimSpace(instruction.RepoBindingID) == "" {
		return errors.New("checkout lease instruction repo_binding_id is required")
	}
	if instruction.RepoBindingID != r.config.RepoBindingID {
		return fmt.Errorf("checkout lease repo_binding_id %q does not match runner scope %q", instruction.RepoBindingID, r.config.RepoBindingID)
	}
	if instruction.RawSourceUploaded {
		return errors.New("checkout lease instruction unexpectedly claims raw source upload")
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
	if strings.TrimSpace(config.WorkspaceRef) == "" {
		return errors.New("workspace ref is required")
	}
	if strings.TrimSpace(config.CommitSHA) == "" {
		return errors.New("commit sha is required")
	}
	if config.PollInterval < 0 {
		return errors.New("poll interval must be non-negative")
	}
	if config.LeaseTTLSeconds < 0 {
		return errors.New("lease ttl seconds must be non-negative")
	}
	return nil
}

func workspaceRefForLease(base string, lease checkoutLease) string {
	return fmt.Sprintf("%s#checkout_job=%s;task=%s;repo_binding=%s", strings.TrimSpace(base), lease.JobID, lease.TaskID, lease.Instruction.RepoBindingID)
}

func cloneNonBlankStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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
