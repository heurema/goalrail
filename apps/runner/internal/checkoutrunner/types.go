package checkoutrunner

import "time"

type checkoutLeaseCreateRequest struct {
	RunnerID   string `json:"runner_id"`
	TTLSeconds int    `json:"ttl_seconds,omitempty"`
}

type checkoutLease struct {
	JobID          string              `json:"job_id"`
	TaskID         string              `json:"task_id"`
	State          string              `json:"state"`
	RunnerID       string              `json:"runner_id"`
	LeaseToken     string              `json:"lease_token"`
	LeaseExpiresAt time.Time           `json:"lease_expires_at"`
	Instruction    checkoutInstruction `json:"instruction"`
}

type checkoutInstruction struct {
	JobID              string    `json:"job_id"`
	TaskID             string    `json:"task_id"`
	RepoBindingID      string    `json:"repo_binding_id"`
	AccessMode         string    `json:"access_mode"`
	Provider           string    `json:"provider"`
	RepositoryFullName string    `json:"repository_full_name"`
	RepositoryURL      string    `json:"repository_url"`
	WorkflowBaseBranch string    `json:"workflow_base_branch"`
	PathScope          string    `json:"path_scope"`
	SourceRef          sourceRef `json:"source_ref"`
	RawSourceUploaded  bool      `json:"raw_source_uploaded"`
}

type sourceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type checkoutReceiptSubmitRequest struct {
	LeaseToken        string   `json:"lease_token"`
	RunnerID          string   `json:"runner_id"`
	WorkspaceRef      string   `json:"workspace_ref"`
	CommitSHA         string   `json:"commit_sha"`
	BaselineID        string   `json:"baseline_id,omitempty"`
	OverlayID         string   `json:"overlay_id,omitempty"`
	Dirty             bool     `json:"dirty"`
	Partial           bool     `json:"partial"`
	PartialReasons    []string `json:"partial_reasons,omitempty"`
	RawSourceUploaded bool     `json:"raw_source_uploaded"`
}

type checkoutReceipt struct {
	ID                string `json:"id"`
	JobID             string `json:"job_id"`
	TaskID            string `json:"task_id"`
	RepoBindingID     string `json:"repo_binding_id"`
	RunnerID          string `json:"runner_id"`
	WorkspaceRef      string `json:"workspace_ref"`
	CommitSHA         string `json:"commit_sha"`
	RawSourceUploaded bool   `json:"raw_source_uploaded"`
}
