package spine

type WorkStartOutput struct {
	Mode                 string `json:"mode"`
	ServerURL            string `json:"server_url"`
	OrganizationID       string `json:"organization_id"`
	ProjectID            string `json:"project_id"`
	RepoBindingID        string `json:"repo_binding_id"`
	IntakeID             string `json:"intake_id"`
	IntakeState          string `json:"intake_state"`
	GoalID               string `json:"goal_id"`
	GoalState            string `json:"goal_state"`
	Title                string `json:"title"`
	LocalConfigPath      string `json:"local_config_path"`
	Message              string `json:"message"`
	NextSuggestedCommand string `json:"next_suggested_command"`
}
