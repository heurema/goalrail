package spine

const RepoBindingStatusPendingServerKeyProvisioning = "pending_server_key_provisioning"

type RepoBindingDraft struct {
	RepoURL     string `json:"repo_url"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	NextCommand string `json:"next_suggested_command"`
}
