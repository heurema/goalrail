package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrJWTSecretMissing = errors.New("auth JWT secret is not configured")
	ErrInvalidToken     = errors.New("access token is invalid")
	ErrExpiredToken     = errors.New("access token is expired")
)

type AccessTokenClaims struct {
	UserID    spine.UserID
	SessionID spine.UserSessionID
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type AccessTokenManager struct {
	secret []byte
}

func NewAccessTokenManager(secret string) AccessTokenManager {
	return AccessTokenManager{secret: []byte(strings.TrimSpace(secret))}
}

func (m AccessTokenManager) Sign(claims AccessTokenClaims) (string, error) {
	if len(m.secret) == 0 {
		return "", ErrJWTSecretMissing
	}
	if strings.TrimSpace(string(claims.UserID)) == "" || strings.TrimSpace(string(claims.SessionID)) == "" {
		return "", fmt.Errorf("%w: missing subject or session", ErrInvalidToken)
	}
	header := jwtHeader{Algorithm: "HS256", Type: "JWT"}
	payload := jwtPayload{
		Subject:   string(claims.UserID),
		SessionID: string(claims.SessionID),
		IssuedAt:  claims.IssuedAt.UTC().Unix(),
		ExpiresAt: claims.ExpiresAt.UTC().Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal JWT header: %w", err)
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal JWT payload: %w", err)
	}

	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := m.sign(unsigned)
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (m AccessTokenManager) Validate(token string, now time.Time) (AccessTokenClaims, error) {
	if len(m.secret) == 0 {
		return AccessTokenClaims{}, ErrJWTSecretMissing
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return AccessTokenClaims{}, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return AccessTokenClaims{}, ErrInvalidToken
	}
	if !hmac.Equal(signature, m.sign(unsigned)) {
		return AccessTokenClaims{}, ErrInvalidToken
	}

	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return AccessTokenClaims{}, err
	}
	if header.Algorithm != "HS256" || header.Type != "JWT" {
		return AccessTokenClaims{}, ErrInvalidToken
	}

	var payload jwtPayload
	if err := decodeJWTPart(parts[1], &payload); err != nil {
		return AccessTokenClaims{}, err
	}
	if strings.TrimSpace(payload.Subject) == "" || strings.TrimSpace(payload.SessionID) == "" || payload.ExpiresAt <= 0 {
		return AccessTokenClaims{}, ErrInvalidToken
	}
	if !now.UTC().Before(time.Unix(payload.ExpiresAt, 0).UTC()) {
		return AccessTokenClaims{}, ErrExpiredToken
	}

	return AccessTokenClaims{
		UserID:    spine.UserID(payload.Subject),
		SessionID: spine.UserSessionID(payload.SessionID),
		IssuedAt:  time.Unix(payload.IssuedAt, 0).UTC(),
		ExpiresAt: time.Unix(payload.ExpiresAt, 0).UTC(),
	}, nil
}

func (m AccessTokenManager) sign(unsigned string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(unsigned))
	return mac.Sum(nil)
}

func decodeJWTPart(part string, target any) error {
	raw, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return ErrInvalidToken
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidToken
	}
	return nil
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type jwtPayload struct {
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}
