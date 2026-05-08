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
	ServerURL       string
	BearerToken     string
	ProjectID       string
	RepoBindingID   string
	RunnerID        string
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
	StepRunStarted     StepResult = "run_started"
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
