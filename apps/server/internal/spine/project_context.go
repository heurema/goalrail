package spine

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UserID string

type InstallationID string

type OrganizationID string

type OrganizationMembershipID string

type ProjectID string

type EntityState string

const EntityStateActive EntityState = "active"

type InstallationMode string

const (
	InstallationModeSelfHosted InstallationMode = "self_hosted"
	InstallationModeSaaS       InstallationMode = "saas"
)

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

type Installation struct {
	ID            InstallationID   `json:"id"`
	Mode          InstallationMode `json:"mode"`
	PublicBaseURL string           `json:"public_base_url"`
	State         EntityState      `json:"state"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type Organization struct {
	ID             OrganizationID `json:"id"`
	InstallationID InstallationID `json:"installation_id"`
	Slug           string         `json:"slug"`
	DisplayName    string         `json:"display_name"`
	State          EntityState    `json:"state"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
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
	ID              ProjectID      `json:"id"`
	OrganizationID  OrganizationID `json:"organization_id"`
	CreatedByUserID UserID         `json:"created_by_user_id"`
	Slug            string         `json:"slug"`
	DisplayName     string         `json:"display_name"`
	State           EntityState    `json:"state"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type RepoBinding struct {
	ID                   RepoBindingID         `json:"id"`
	OrganizationID       OrganizationID        `json:"organization_id"`
	ProjectID            ProjectID             `json:"project_id"`
	CreatedByUserID      UserID                `json:"created_by_user_id"`
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

func NewUserID() (UserID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return UserID(id), nil
}

func NewInstallationID() (InstallationID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return InstallationID(id), nil
}

func NewOrganizationID() (OrganizationID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return OrganizationID(id), nil
}

func NewOrganizationMembershipID() (OrganizationMembershipID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return OrganizationMembershipID(id), nil
}

func NewProjectID() (ProjectID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return ProjectID(id), nil
}

func NewRepoBindingID() (RepoBindingID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return RepoBindingID(id), nil
}

func newUUIDv7() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("new uuidv7: %w", err)
	}
	return id.String(), nil
}
