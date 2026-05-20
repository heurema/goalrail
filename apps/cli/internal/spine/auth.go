package spine

type AuthSessionMetadata struct {
	ServerURL             string `json:"server_url,omitempty"`
	UsedStoredAccessToken bool   `json:"used_stored_access_token"`
	RefreshAttempted      bool   `json:"refresh_attempted"`
	AccessTokenRefreshed  bool   `json:"access_token_refreshed"`
	Reason                string `json:"reason,omitempty"`
}
