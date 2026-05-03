package spine

import "time"

type UserSessionID string

type UserSessionState string

const (
	UserSessionStateActive  UserSessionState = "active"
	UserSessionStateRevoked UserSessionState = "revoked"
	UserSessionStateExpired UserSessionState = "expired"
)

type UserPasswordCredential struct {
	UserID             UserID     `json:"user_id"`
	PasswordHash       string     `json:"-"`
	MustChangePassword bool       `json:"must_change_password"`
	PasswordChangedAt  *time.Time `json:"password_changed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type UserSession struct {
	ID               UserSessionID    `json:"id"`
	UserID           UserID           `json:"user_id"`
	RefreshTokenHash string           `json:"-"`
	State            UserSessionState `json:"state"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	ExpiresAt        time.Time        `json:"expires_at"`
	RevokedAt        *time.Time       `json:"revoked_at,omitempty"`
	LastUsedAt       *time.Time       `json:"last_used_at,omitempty"`
}

type CLIAuthCode struct {
	CodeHash            string     `json:"-"`
	UserID              UserID     `json:"user_id"`
	RedirectURI         string     `json:"redirect_uri"`
	State               string     `json:"state"`
	CodeChallenge       string     `json:"code_challenge"`
	CodeChallengeMethod string     `json:"code_challenge_method"`
	CreatedAt           time.Time  `json:"created_at"`
	ExpiresAt           time.Time  `json:"expires_at"`
	ConsumedAt          *time.Time `json:"consumed_at,omitempty"`
}

func NewUserSessionID() (UserSessionID, error) {
	id, err := newUUIDv7()
	if err != nil {
		return "", err
	}
	return UserSessionID(id), nil
}
