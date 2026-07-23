package localrun

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const StateHomeEnvironment = "GOALRAIL_STATE_HOME"

var (
	ErrStateAlreadyExists = errors.New("local-run state already exists")
	ErrStateNotFound      = errors.New("local-run state not found")
)

type Store struct {
	root string
}

func ResolveStateRoot(explicit string) (string, error) {
	root := strings.TrimSpace(explicit)
	if root == "" {
		root = strings.TrimSpace(os.Getenv(StateHomeEnvironment))
	}
	if root == "" {
		if xdg := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); xdg != "" {
			root = filepath.Join(xdg, "goalrail")
		}
	}
	if root == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user state directory: %w", err)
		}
		root = filepath.Join(userHome, ".local", "state", "goalrail")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve state root: %w", err)
	}
	absolute = filepath.Clean(absolute)
	if absolute == string(filepath.Separator) {
		return "", fmt.Errorf("filesystem root cannot be the Goalrail state directory")
	}
	return absolute, nil
}

func NewStore(root string) (*Store, error) {
	resolved, err := ResolveStateRoot(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create state root: %w", err)
	}
	if err := os.Chmod(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("protect state root: %w", err)
	}
	return &Store{root: resolved}, nil
}

func (store *Store) Root() string {
	return store.root
}

func (store *Store) WriteJSONOnce(relative string, value any, allowIdentical bool) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode local-run state: %w", err)
	}
	return store.WriteBytesOnce(relative, encoded, allowIdentical)
}

func (store *Store) WriteBytesOnce(relative string, content []byte, allowIdentical bool) error {
	target, err := store.path(relative)
	if err != nil {
		return err
	}
	directory := filepath.Dir(target)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("protect state directory: %w", err)
	}

	temp, err := os.CreateTemp(directory, ".goalrail-state-*")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("protect temporary state: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary state: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync temporary state: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary state: %w", err)
	}

	if err := os.Link(tempPath, target); err != nil {
		if errors.Is(err, os.ErrExist) {
			if allowIdentical {
				existing, readErr := os.ReadFile(target)
				if readErr != nil {
					return fmt.Errorf("read existing state: %w", readErr)
				}
				if bytes.Equal(existing, content) {
					return nil
				}
			}
			return fmt.Errorf("%w: %s", ErrStateAlreadyExists, relative)
		}
		return fmt.Errorf("publish state: %w", err)
	}
	return nil
}

func (store *Store) ReadBytes(relative string) ([]byte, error) {
	target, err := store.path(relative)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrStateNotFound, relative)
	}
	if err != nil {
		return nil, fmt.Errorf("read local-run state: %w", err)
	}
	return content, nil
}

func (store *Store) ReadJSON(relative string, target any) error {
	content, err := store.ReadBytes(relative)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode local-run state: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("decode local-run state: multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode local-run state: %w", err)
	}
	return nil
}

func (store *Store) Exists(relative string) (bool, error) {
	target, err := store.path(relative)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(target)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("inspect local-run state: %w", err)
}

func (store *Store) path(relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.ContainsRune(relative, '\x00') {
		return "", fmt.Errorf("invalid state path %q", relative)
	}
	cleaned := filepath.Clean(relative)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("state path escapes the state root")
	}
	return filepath.Join(store.root, cleaned), nil
}
