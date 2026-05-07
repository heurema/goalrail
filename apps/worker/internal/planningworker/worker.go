package planningworker

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	ServerURL       string
	WorkerID        string
	PollInterval    time.Duration
	LeaseTTLSeconds int
	Once            bool
	HTTPClient      *http.Client
	LogWriter       io.Writer
	Planner         proposalPlanner
}

type Runner struct {
	config Config
	client *apiClient
	logger *log.Logger
}

type StepResult string

const (
	StepNoWork         StepResult = "no_work"
	StepProposal       StepResult = "proposal_submitted"
	StepLeaseExpired   StepResult = "lease_expired"
	StepInvalidLease   StepResult = "invalid_lease"
	StepUnsupported    StepResult = "unsupported_planner_input"
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
	client, err := newAPIClient(config.ServerURL, config.HTTPClient)
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
	if config.Planner == nil {
		config.Planner = minimalPlanner{}
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
	lease, ok, err := r.client.acquireLease(ctx, leaseCreateRequest{
		LeasedBy: actorRef{
			Kind: "worker",
			ID:   r.config.WorkerID,
		},
		TTLSeconds: r.config.LeaseTTLSeconds,
	})
	if err != nil {
		return "", err
	}
	if !ok {
		r.logger.Printf("no planning work available")
		return StepNoWork, nil
	}
	r.logger.Printf("acquired planning lease lease_id=%s plan_id=%s", lease.ID, lease.PlanID)

	plan, err := r.client.getPlan(ctx, lease.PlanID)
	if err != nil {
		return "", err
	}
	proposal, err := r.config.Planner.BuildProposal(r.config.WorkerID, lease, plan)
	if err != nil {
		if errors.Is(err, errUnsupportedPlannerInput) {
			r.logger.Printf("unsupported planner input plan_id=%s: %v", lease.PlanID, err)
			return StepUnsupported, nil
		}
		return "", err
	}

	submitted, err := r.client.submitProposal(ctx, plan.ID, proposal)
	if err != nil {
		switch apiErrorCode(err) {
		case "lease_expired":
			r.logger.Printf("planning lease expired before proposal submission lease_id=%s plan_id=%s; abandoning local proposal", lease.ID, lease.PlanID)
			return StepLeaseExpired, nil
		case "invalid_lease":
			r.logger.Printf("planning lease rejected before proposal submission lease_id=%s plan_id=%s; abandoning local proposal", lease.ID, lease.PlanID)
			return StepInvalidLease, nil
		case "already_proposed":
			r.logger.Printf("planning proposal already exists plan_id=%s; abandoning local proposal", lease.PlanID)
			return StepAlreadyHandled, nil
		default:
			return "", err
		}
	}
	r.logger.Printf("submitted planning proposal proposal_id=%s plan_id=%s", submitted.ID, submitted.PlanID)
	return StepProposal, nil
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.ServerURL) == "" {
		return errors.New("server url is required")
	}
	if strings.TrimSpace(config.WorkerID) == "" {
		return errors.New("worker id is required")
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
