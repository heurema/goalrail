package spine

import "time"

type UserID string

type OrganizationID string

type OrganizationMembershipID string

type ProjectID string

type EntityState string

const EntityStateActive EntityState = "active"

type OrganizationMembershipRole string

const (
	OrganizationMembershipRoleOwner  OrganizationMembershipRole = "owner"
	OrganizationMembershipRoleAdmin  OrganizationMembershipRole = "admin"
	OrganizationMembershipRoleMember OrganizationMembershipRole = "member"
	OrganizationMembershipRoleViewer OrganizationMembershipRole = "viewer"
)

type RepoBindingAccessMode string

const (
	RepoBindingAccessModeProviderTokenCheckout    RepoBindingAccessMode = "provider_token_checkout"
	RepoBindingAccessModeCustomerRunnerCheckout   RepoBindingAccessMode = "customer_runner_checkout"
	RepoBindingAccessModeCustomerMountedWorkspace RepoBindingAccessMode = "customer_mounted_workspace"
	RepoBindingAccessModeMetadataOnly             RepoBindingAccessMode = "metadata_only"
)

type User struct {
	ID          UserID      `json:"id"`
	DisplayName string      `json:"display_name"`
	Email       string      `json:"email,omitempty"`
	State       EntityState `json:"state"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type Organization struct {
	ID          OrganizationID `json:"id"`
	Slug        string         `json:"slug"`
	DisplayName string         `json:"display_name"`
	State       EntityState    `json:"state"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type OrganizationMembership struct {
	ID             OrganizationMembershipID   `json:"id"`
	OrganizationID OrganizationID             `json:"organization_id"`
	UserID         UserID                     `json:"user_id"`
	Role           OrganizationMembershipRole `json:"role"`
	State          EntityState                `json:"state"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

type Project struct {
	ID             ProjectID      `json:"id"`
	OrganizationID OrganizationID `json:"organization_id"`
	Slug           string         `json:"slug"`
	DisplayName    string         `json:"display_name"`
	State          EntityState    `json:"state"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type RepoBinding struct {
	ID                   RepoBindingID         `json:"id"`
	OrganizationID       OrganizationID        `json:"organization_id"`
	ProjectID            ProjectID             `json:"project_id"`
	VcsConnectionID      string                `json:"vcs_connection_id,omitempty"`
	Provider             string                `json:"provider"`
	RepositoryExternalID string                `json:"repository_external_id,omitempty"`
	RepositoryFullName   string                `json:"repository_full_name"`
	RepositoryURL        string                `json:"repository_url"`
	DefaultBranch        string                `json:"default_branch"`
	PathScope            string                `json:"path_scope"`
	AccessMode           RepoBindingAccessMode `json:"access_mode"`
	State                EntityState           `json:"state"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

type ResolvedRepoBindingContext struct {
	OrganizationID OrganizationID `json:"organization_id"`
	ProjectID      ProjectID      `json:"project_id"`
	RepoBindingID  RepoBindingID  `json:"repo_binding_id"`
}
