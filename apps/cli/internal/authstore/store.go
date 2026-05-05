package authstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrSessionNotFound = errors.New("auth session not found")

type Session struct {
	ServerURL            string    `json:"server_url"`
	AccessToken          string    `json:"access_token"`
	RefreshToken         string    `json:"refresh_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
	TokenType            string    `json:"token_type"`
}

type FileStore struct {
	path string
}

func NewFileStore(path string) FileStore {
	return FileStore{path: path}
}

func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, "goalrail", "auth.json"), nil
}

func (s FileStore) Load() (Session, error) {
	file, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, fmt.Errorf("open auth file: %w", err)
	}
	defer file.Close()

	var session Session
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&session); err != nil {
		return Session{}, fmt.Errorf("decode auth file: %w", err)
	}
	if session.ServerURL == "" || session.AccessToken == "" || session.TokenType == "" || session.AccessTokenExpiresAt.IsZero() {
		return Session{}, fmt.Errorf("auth session is incomplete")
	}
	return session, nil
}

func (s FileStore) Save(session Session) error {
	if session.ServerURL == "" || session.AccessToken == "" || session.RefreshToken == "" || session.TokenType == "" || session.AccessTokenExpiresAt.IsZero() {
		return fmt.Errorf("auth session is incomplete")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create auth config directory: %w", err)
	}

	file, err := os.OpenFile(s.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open auth file: %w", err)
	}
	encodeErr := json.NewEncoder(file).Encode(session)
	closeErr := file.Close()
	chmodErr := os.Chmod(s.path, 0o600)
	if encodeErr != nil {
		return fmt.Errorf("write auth file: %w", encodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close auth file: %w", closeErr)
	}
	if chmodErr != nil {
		return fmt.Errorf("restrict auth file permissions: %w", chmodErr)
	}
	return nil
}
