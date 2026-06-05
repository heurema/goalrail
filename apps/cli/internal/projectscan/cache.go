package projectscan

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	latestBaselineFile = "latest-baseline.json"
	overlayCurrentFile = "overlay-current.json"
	statusReceiptFile  = "status-porcelain-v2.txt"
)

type Cache struct {
	Root string
}

func NewCache(root string) Cache {
	return Cache{Root: root}
}

func DefaultCacheRoot() (string, error) {
	root, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "goalrail", "project-scans"), nil
}

func (c Cache) root() (string, error) {
	if strings.TrimSpace(c.Root) != "" {
		return c.Root, nil
	}
	return DefaultCacheRoot()
}

func (c Cache) Directory(repoBindingID string, canonicalRepoRoot string) (string, error) {
	root, err := c.root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, cacheKey(repoBindingID, canonicalRepoRoot)), nil
}

func (c Cache) LoadLatestBaseline(repoBindingID string, canonicalRepoRoot string) (RepositoryBaselineProfile, bool, error) {
	dir, err := c.Directory(repoBindingID, canonicalRepoRoot)
	if err != nil {
		return RepositoryBaselineProfile{}, false, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, latestBaselineFile))
	if errors.Is(err, os.ErrNotExist) {
		return RepositoryBaselineProfile{}, false, nil
	}
	if err != nil {
		return RepositoryBaselineProfile{}, false, err
	}
	var baseline RepositoryBaselineProfile
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return RepositoryBaselineProfile{}, false, nil
	}
	if strings.TrimSpace(baseline.RepositoryBaselineProfileID) == "" {
		return RepositoryBaselineProfile{}, false, nil
	}
	return baseline, true, nil
}

func (c Cache) LoadCurrentOverlay(repoBindingID string, canonicalRepoRoot string) (WorkspaceOverlay, bool, error) {
	dir, err := c.Directory(repoBindingID, canonicalRepoRoot)
	if err != nil {
		return WorkspaceOverlay{}, false, err
	}
	raw, err := os.ReadFile(filepath.Join(dir, overlayCurrentFile))
	if errors.Is(err, os.ErrNotExist) {
		return WorkspaceOverlay{}, false, nil
	}
	if err != nil {
		return WorkspaceOverlay{}, false, err
	}
	var overlay WorkspaceOverlay
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return WorkspaceOverlay{}, false, nil
	}
	if strings.TrimSpace(overlay.WorkspaceOverlayID) == "" {
		return WorkspaceOverlay{}, false, nil
	}
	return overlay, true, nil
}

func (c Cache) WriteBaseline(profile RepositoryBaselineProfile) error {
	dir, err := c.Directory(profile.RepoBindingID, profile.CanonicalRepoRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	shortHead := profile.HeadSHA
	if len(shortHead) > 12 {
		shortHead = shortHead[:12]
	}
	name := fmt.Sprintf("baseline-%s-v%d.json", shortHead, profile.SchemaVersion)
	if err := writeJSONAtomic(filepath.Join(dir, name), profile); err != nil {
		return err
	}
	return writeJSONAtomic(filepath.Join(dir, latestBaselineFile), profile)
}

func (c Cache) WriteOverlay(overlay WorkspaceOverlay, rawStatus string) error {
	dir, err := c.Directory(overlay.RepoBindingID, overlay.CanonicalRepoRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if rawStatus != "" {
		if err := writeFileAtomic(filepath.Join(dir, statusReceiptFile), []byte(rawStatus)); err != nil {
			return err
		}
	}
	return writeJSONAtomic(filepath.Join(dir, overlayCurrentFile), overlay)
}

func writeJSONAtomic(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return writeFileAtomic(path, raw)
}

func writeFileAtomic(path string, raw []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() {
		_ = os.Remove(tempName)
	}()
	if _, err := temp.Write(raw); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}
