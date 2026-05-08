package executionrunner

import "time"

type executionLeaseCreateRequest struct {
	ProjectID     string `json:"project_id"`
	RepoBindingID string `json:"repo_binding_id"`
	RunnerID      string `json:"runner_id"`
	TTLSeconds    int    `json:"ttl_seconds,omitempty"`
}

type executionLease struct {
	ID                string       `json:"id"`
	ExecutionJobID    string       `json:"execution_job_id"`
	TaskID            string       `json:"task_id"`
	CheckoutReceiptID string       `json:"checkout_receipt_id"`
	RepoBindingID     string       `json:"repo_binding_id"`
	RunnerID          string       `json:"runner_id"`
	State             string       `json:"state"`
	LeaseToken        string       `json:"lease_token"`
	ExpiresAt         time.Time    `json:"expires_at"`
	ExecutionJob      executionJob `json:"execution_job"`
}

type executionJob struct {
	ID                string `json:"id"`
	TaskID            string `json:"task_id"`
	RepoBindingID     string `json:"repo_binding_id"`
	CheckoutReceiptID string `json:"checkout_receipt_id"`
	State             string `json:"state"`
}

type runStartRequest struct {
	LeaseID    string `json:"lease_id"`
	LeaseToken string `json:"lease_token"`
	RunnerID   string `json:"runner_id"`
}

type runStarted struct {
	ID               string `json:"id"`
	ExecutionJobID   string `json:"execution_job_id"`
	ExecutionLeaseID string `json:"execution_lease_id"`
	TaskID           string `json:"task_id"`
	RunnerID         string `json:"runner_id"`
	State            string `json:"state"`
}
