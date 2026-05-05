package spine

import "time"

type VcsConnectionID string

type VcsConnectionState string

const VcsConnectionStatePendingSetup VcsConnectionState = "pending_setup"

type VcsConnection struct {
	ID                  VcsConnectionID    `json:"id"`
	InstallationID      InstallationID     `json:"installation_id"`
	OrganizationID      OrganizationID     `json:"organization_id"`
	CreatedByUserID     UserID             `json:"created_by_user_id"`
	ProviderKind        string             `json:"provider_kind"`
	ProviderInstanceURL string             `json:"provider_instance_url"`
	State               VcsConnectionState `json:"state"`
	SetupExpiresAt      time.Time          `json:"setup_expires_at"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

type VcsConnectionCreateRequest struct {
	ProviderKind        string `json:"provider_kind"`
	ProviderInstanceURL string `json:"provider_instance_url"`
}

func NewVcsConnectionID() (VcsConnectionID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return VcsConnectionID(id), nil
}
