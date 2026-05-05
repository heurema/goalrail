package spine

const RepoBindingStatusPendingServerKeyProvisioning = "pending_server_key_provisioning"

type RepoBindingDraft struct {
	RepoURL               string   `json:"repo_url"`
	Status                string   `json:"status"`
	Message               string   `json:"message"`
	NextCommand           string   `json:"next_suggested_command"`
	GitRoot               string   `json:"git_root"`
	RemoteName            string   `json:"remote_name"`
	Provider              string   `json:"provider"`
	ProviderHost          string   `json:"provider_host"`
	RepositoryFullName    string   `json:"repository_full_name"`
	ProviderDefaultBranch string   `json:"provider_default_branch"`
	WorkflowBaseBranch    string   `json:"workflow_base_branch"`
	HeadSHA               string   `json:"head_sha"`
	Warnings              []string `json:"warnings"`
}

type RepoBindingInitRequest struct {
	Provider              string `json:"provider"`
	RepositoryFullName    string `json:"repository_full_name"`
	RepositoryURL         string `json:"repository_url"`
	ProviderDefaultBranch string `json:"provider_default_branch"`
	WorkflowBaseBranch    string `json:"workflow_base_branch"`
	LocalRemoteName       string `json:"local_remote_name"`
	LocalHeadSHA          string `json:"local_head_sha"`
}

type RepositoryContextInitRequest struct {
	Provider                    string `json:"provider"`
	RepositoryFullName          string `json:"repository_full_name"`
	RepositoryURL               string `json:"repository_url"`
	ProviderDefaultBranch       string `json:"provider_default_branch"`
	WorkflowBaseBranch          string `json:"workflow_base_branch"`
	LocalRemoteName             string `json:"local_remote_name"`
	LocalHeadSHA                string `json:"local_head_sha"`
	SuggestedProjectSlug        string `json:"suggested_project_slug"`
	SuggestedProjectDisplayName string `json:"suggested_project_display_name"`
}

type RepoBindingInitResponse struct {
	RepoBindingID         string `json:"repo_binding_id"`
	ProjectID             string `json:"project_id"`
	OrganizationID        string `json:"organization_id"`
	Provider              string `json:"provider"`
	RepositoryFullName    string `json:"repository_full_name"`
	RepositoryURL         string `json:"repository_url"`
	ProviderDefaultBranch string `json:"provider_default_branch"`
	WorkflowBaseBranch    string `json:"workflow_base_branch"`
	State                 string `json:"state"`
	Created               bool   `json:"created"`
	Message               string `json:"message"`
}

type RepositoryContextInitResponse struct {
	OrganizationID        string `json:"organization_id"`
	ProjectID             string `json:"project_id"`
	ProjectSlug           string `json:"project_slug"`
	ProjectDisplayName    string `json:"project_display_name"`
	ProjectCreated        bool   `json:"project_created"`
	RepoBindingID         string `json:"repo_binding_id"`
	RepoBindingCreated    bool   `json:"repo_binding_created"`
	Provider              string `json:"provider"`
	RepositoryFullName    string `json:"repository_full_name"`
	RepositoryURL         string `json:"repository_url"`
	ProviderDefaultBranch string `json:"provider_default_branch"`
	WorkflowBaseBranch    string `json:"workflow_base_branch"`
	State                 string `json:"state"`
	Message               string `json:"message"`
}

type RepoBindingInitOutput struct {
	Mode                  string `json:"mode"`
	ServerURL             string `json:"server_url"`
	ProjectID             string `json:"project_id"`
	RepoBindingID         string `json:"repo_binding_id"`
	OrganizationID        string `json:"organization_id"`
	Provider              string `json:"provider"`
	RepositoryFullName    string `json:"repository_full_name"`
	RepositoryURL         string `json:"repository_url"`
	ProviderDefaultBranch string `json:"provider_default_branch"`
	WorkflowBaseBranch    string `json:"workflow_base_branch"`
	State                 string `json:"state"`
	Created               bool   `json:"created"`
	Message               string `json:"message"`
	NextCommand           string `json:"next_suggested_command"`
	LocalConfigPath       string `json:"local_config_path"`
	LocalConfigStatus     string `json:"local_config_status"`
}

type RepositoryContextInitOutput struct {
	Mode                  string `json:"mode"`
	ServerURL             string `json:"server_url"`
	OrganizationID        string `json:"organization_id"`
	ProjectID             string `json:"project_id"`
	ProjectSlug           string `json:"project_slug"`
	ProjectDisplayName    string `json:"project_display_name"`
	ProjectCreated        bool   `json:"project_created"`
	RepoBindingID         string `json:"repo_binding_id"`
	RepoBindingCreated    bool   `json:"repo_binding_created"`
	Provider              string `json:"provider"`
	RepositoryFullName    string `json:"repository_full_name"`
	RepositoryURL         string `json:"repository_url"`
	ProviderDefaultBranch string `json:"provider_default_branch"`
	WorkflowBaseBranch    string `json:"workflow_base_branch"`
	State                 string `json:"state"`
	Message               string `json:"message"`
	LocalConfigPath       string `json:"local_config_path"`
	LocalConfigStatus     string `json:"local_config_status"`
	ContextSnapshotID     string `json:"context_snapshot_id"`
	ContextSnapshotStatus string `json:"context_snapshot_status"`
	ContextFingerprint    string `json:"context_fingerprint"`
	NextCommand           string `json:"next_suggested_command"`
}

type RepositoryContextSnapshotRequest struct {
	Source                  string                              `json:"source"`
	SchemaVersion           int                                 `json:"schema_version"`
	Repository              RepositoryContextSnapshotRepository `json:"repository"`
	DetectedPaths           []string                            `json:"detected_paths"`
	DetectedToolchains      []string                            `json:"detected_toolchains"`
	DetectedPackageManagers []string                            `json:"detected_package_managers"`
	WorkspaceCandidates     []string                            `json:"workspace_candidates"`
}

type RepositoryContextSnapshotRepository struct {
	Provider              string `json:"provider"`
	FullName              string `json:"full_name"`
	URL                   string `json:"url"`
	ProviderDefaultBranch string `json:"provider_default_branch"`
	WorkflowBaseBranch    string `json:"workflow_base_branch"`
	RemoteName            string `json:"remote_name"`
	HeadSHA               string `json:"head_sha"`
}

type RepositoryContextSnapshotResponse struct {
	ContextSnapshotID string `json:"context_snapshot_id"`
	OrganizationID    string `json:"organization_id"`
	ProjectID         string `json:"project_id"`
	RepoBindingID     string `json:"repo_binding_id"`
	Source            string `json:"source"`
	SchemaVersion     int    `json:"schema_version"`
	Fingerprint       string `json:"fingerprint"`
	Created           bool   `json:"created"`
	Message           string `json:"message"`
}
