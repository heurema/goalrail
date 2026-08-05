package setup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/releasebundle"
)

const (
	PlanVerificationSchemaV1   = "goalrail.setup-plan-verification/v1"
	maxBundleContractBytes     = 1 << 20
	maxVerifiedBundleFiles     = 16 << 10
	maxVerifiedBundleEntries   = 64 << 10
	maxVerifiedBundleFileBytes = int64(512 << 20)
	maxVerifiedBundleBytes     = int64(2 << 30)
)

var ErrBundleInvalid = errors.New("temporary setup bundle is invalid")

// VerifyPlanOptions identifies only read-only inputs. BundleRoot is an
// extracted temporary setup bundle; no path here is a durable install target.
type VerifyPlanOptions struct {
	RepositoryRoot      string
	Home                string
	BundleRoot          string
	PlanPath            string
	AuthorizationPath   string
	ReleaseMetadataPath string
	ReleaseChannel      string
	Scaffold            ambient.Scaffold
}

// PlanVerification is the bounded machine-facing result. Absolute local paths
// remain inputs and are deliberately omitted from the report.
type PlanVerification struct {
	Schema              string              `json:"schema"`
	Verified            bool                `json:"verified"`
	ProjectID           domain.ProjectID    `json:"project_id"`
	PlanDigest          domain.SHA256Digest `json:"plan_digest"`
	AuthorizationDigest domain.SHA256Digest `json:"authorization_digest"`
	ManifestDigest      domain.SHA256Digest `json:"manifest_digest"`
	ReleaseVersion      string              `json:"release_version"`
	Platform            string              `json:"platform"`
}

// VerifiedPlan is intentionally non-constructible with useful state outside
// this package. The apply milestone can accept this proof without trusting a
// caller-built report.
type VerifiedPlan struct {
	attempt      AuthorizedAttempt
	bundleRoot   string
	manifest     releasebundle.SetupBundleManifest
	manifestRaw  []byte
	metadataRaw  []byte
	verification PlanVerification
	options      VerifyPlanOptions
}

// Report returns the bounded result of verification.
func (verified VerifiedPlan) Report() PlanVerification {
	return verified.verification
}

// VerifyPlan proves that an extracted temporary bundle matches its canonical
// manifest and that re-planning from those exact bytes is semantically equal
// to the explicitly authorized complete plan. It performs no writes.
func VerifyPlan(ctx context.Context, options VerifyPlanOptions) (VerifiedPlan, error) {
	var err error
	options, err = normalizeVerifyPlanOptions(options)
	if err != nil {
		return VerifiedPlan{}, err
	}
	bundleRoot, manifestRaw, manifest, err := loadTemporaryBundleManifest(options.BundleRoot)
	if err != nil {
		return VerifiedPlan{}, err
	}
	if manifest.Platform.OS != runtime.GOOS || manifest.Platform.Arch != runtime.GOARCH {
		return VerifiedPlan{}, fmt.Errorf("%w: bundle platform %s does not match current %s_%s", ErrBundleInvalid, manifest.Platform.Key(), runtime.GOOS, runtime.GOARCH)
	}

	plan, planArtifact, err := readCanonicalPlan(options.PlanPath)
	if err != nil {
		return VerifiedPlan{}, err
	}
	authorization, authorizationArtifact, err := readCanonicalAuthorization(options.AuthorizationPath)
	if err != nil {
		return VerifiedPlan{}, err
	}
	metadataRaw, err := readCanonicalReleaseMetadata(options.ReleaseMetadataPath)
	if err != nil {
		return VerifiedPlan{}, err
	}

	planOptions := PlanOptions{
		RepositoryRoot: options.RepositoryRoot,
		Home:           options.Home,
		OS:             manifest.Platform.OS,
		Arch:           manifest.Platform.Arch,
		Scaffold:       options.Scaffold,
		ReleaseChannel: options.ReleaseChannel,
		Evidence: &ReleaseEvidence{
			MetadataRaw: metadataRaw,
			ManifestRaw: manifestRaw,
		},
	}
	planOptions, err = normalizeOptions(planOptions)
	if err != nil {
		return VerifiedPlan{}, err
	}
	binding, err := BindAuthorization(
		Result{Plan: plan, Artifact: planArtifact},
		planOptions,
		Consent{Decision: ConsentGrant, Scope: ConsentExactPlan, Authorization: &authorization},
	)
	if err != nil {
		return VerifiedPlan{}, err
	}
	attempt, err := binding.BeforeFirstMutation(ctx)
	if err != nil {
		return VerifiedPlan{}, err
	}
	if err := verifyTemporaryBundleFiles(bundleRoot, manifestRaw, manifest); err != nil {
		return VerifiedPlan{}, err
	}

	verification := PlanVerification{
		Schema:              PlanVerificationSchemaV1,
		Verified:            true,
		ProjectID:           plan.ProjectID,
		PlanDigest:          attempt.PlanArtifact().Digest(),
		AuthorizationDigest: authorizationArtifact.Digest(),
		ManifestDigest:      domain.DigestCanonicalJSON(manifestRaw),
		ReleaseVersion:      manifest.ReleaseVersion,
		Platform:            plan.Platform,
	}
	return VerifiedPlan{
		attempt:      attempt,
		bundleRoot:   bundleRoot,
		manifest:     manifest,
		manifestRaw:  append([]byte(nil), manifestRaw...),
		metadataRaw:  append([]byte(nil), metadataRaw...),
		verification: verification,
		options: VerifyPlanOptions{
			RepositoryRoot:      planOptions.RepositoryRoot,
			Home:                planOptions.Home,
			BundleRoot:          bundleRoot,
			PlanPath:            options.PlanPath,
			AuthorizationPath:   options.AuthorizationPath,
			ReleaseMetadataPath: options.ReleaseMetadataPath,
			ReleaseChannel:      planOptions.ReleaseChannel,
			Scaffold:            planOptions.Scaffold,
		},
	}, nil
}

func normalizeVerifyPlanOptions(options VerifyPlanOptions) (VerifyPlanOptions, error) {
	paths := []struct {
		label string
		value *string
	}{
		{label: "bundle root", value: &options.BundleRoot},
		{label: "plan", value: &options.PlanPath},
		{label: "authorization", value: &options.AuthorizationPath},
		{label: "release metadata", value: &options.ReleaseMetadataPath},
	}
	for _, input := range paths {
		if strings.TrimSpace(*input.value) == "" {
			return VerifyPlanOptions{}, fmt.Errorf("%s path is required", input.label)
		}
		absolute, err := filepath.Abs(*input.value)
		if err != nil {
			return VerifyPlanOptions{}, fmt.Errorf("resolve %s path: %w", input.label, err)
		}
		*input.value = filepath.Clean(absolute)
	}
	return options, nil
}

func loadTemporaryBundleManifest(root string) (string, []byte, releasebundle.SetupBundleManifest, error) {
	if strings.TrimSpace(root) == "" {
		return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: bundle root is required", ErrBundleInvalid)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: resolve bundle root: %v", ErrBundleInvalid, err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: resolve bundle root links: %v", ErrBundleInvalid, err)
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: bundle root is not a real directory", ErrBundleInvalid)
	}

	manifestPath := filepath.Join(resolved, releasebundle.BundleManifestPath)
	manifestRaw, err := readBundleManifest(manifestPath)
	if err != nil {
		return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: %v", ErrBundleInvalid, err)
	}
	manifest, err := releasebundle.DecodeSetupBundleManifest(bytes.NewReader(manifestRaw))
	if err != nil {
		return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: decode manifest: %v", ErrBundleInvalid, err)
	}
	if len(manifest.Files) > maxVerifiedBundleFiles {
		return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: manifest file count exceeds %d", ErrBundleInvalid, maxVerifiedBundleFiles)
	}
	var totalBytes int64
	for _, record := range manifest.Files {
		if record.SizeBytes < 0 || record.SizeBytes > maxVerifiedBundleFileBytes {
			return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: manifest file %s exceeds size limit", ErrBundleInvalid, record.Path)
		}
		if totalBytes > maxVerifiedBundleBytes-record.SizeBytes {
			return "", nil, releasebundle.SetupBundleManifest{}, fmt.Errorf("%w: manifest total size exceeds limit", ErrBundleInvalid)
		}
		totalBytes += record.SizeBytes
	}
	return filepath.Clean(resolved), manifestRaw, manifest, nil
}

func verifyTemporaryBundleFiles(root string, manifestRaw []byte, manifest releasebundle.SetupBundleManifest) error {
	currentManifest, err := readBundleManifest(filepath.Join(root, releasebundle.BundleManifestPath))
	if err != nil || !bytes.Equal(currentManifest, manifestRaw) {
		return fmt.Errorf("%w: embedded manifest changed during verification", ErrBundleInvalid)
	}

	expected := make(map[string]releasebundle.ManifestFile, len(manifest.Files))
	allowedDirectories := map[string]struct{}{".": {}}
	for _, record := range manifest.Files {
		expected[record.Path] = record
		for directory := filepath.ToSlash(filepath.Dir(record.Path)); directory != "."; directory = filepath.ToSlash(filepath.Dir(directory)) {
			allowedDirectories[directory] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(expected))
	entries := 0
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		entries++
		if entries > maxVerifiedBundleEntries {
			return fmt.Errorf("bundle entry count exceeds %d", maxVerifiedBundleEntries)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("bundle entry %s is a symbolic link", relative)
		}
		if entry.IsDir() {
			if _, ok := allowedDirectories[relative]; !ok {
				return fmt.Errorf("bundle contains unmanifested directory %s", relative)
			}
			return nil
		}
		if relative == releasebundle.BundleManifestPath {
			return nil
		}
		record, ok := expected[relative]
		if !ok {
			return fmt.Errorf("bundle contains unmanifested file %s", relative)
		}
		if err := verifyManifestFile(path, record); err != nil {
			return err
		}
		seen[relative] = struct{}{}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBundleInvalid, err)
	}
	if len(seen) != len(expected) {
		return fmt.Errorf("%w: bundle file set is incomplete", ErrBundleInvalid)
	}
	return nil
}

func verifyManifestFile(path string, record releasebundle.ManifestFile) error {
	if record.SizeBytes < 0 || record.SizeBytes > maxVerifiedBundleFileBytes {
		return fmt.Errorf("bundle entry %s exceeds size limit", record.Path)
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", record.Path, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("bundle entry %s is not regular", record.Path)
	}
	mode, err := strconv.ParseUint(record.Mode, 8, 32)
	if err != nil || info.Mode().Perm() != fs.FileMode(mode) || info.Size() != record.SizeBytes {
		return fmt.Errorf("bundle metadata disagrees for %s", record.Path)
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, record.SizeBytes+1))
	if err != nil || written != record.SizeBytes {
		return fmt.Errorf("read bundle entry %s", record.Path)
	}
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if digest != record.SHA256 {
		return fmt.Errorf("bundle digest disagrees for %s", record.Path)
	}
	return nil
}

func readBundleManifest(path string) ([]byte, error) {
	raw, info, err := boundedio.ReadRegularFileWithInfo(path, "setup bundle manifest", maxBundleContractBytes)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm() != 0o644 {
		return nil, fmt.Errorf("setup bundle manifest mode must be 0644")
	}
	return raw, nil
}

func readCanonicalPlan(path string) (domain.SetupPlan, domain.CanonicalArtifact, error) {
	raw, err := boundedio.ReadRegularFile(path, "authorized setup plan", domain.MaxSetupPlanBytes)
	if err != nil {
		return domain.SetupPlan{}, domain.CanonicalArtifact{}, err
	}
	plan, err := domain.DecodeSetupPlan(bytes.NewReader(raw))
	if err != nil {
		return domain.SetupPlan{}, domain.CanonicalArtifact{}, fmt.Errorf("decode authorized setup plan: %w", err)
	}
	artifact, err := domain.FreezeSetupPlan(plan)
	if err != nil {
		return domain.SetupPlan{}, domain.CanonicalArtifact{}, err
	}
	if !bytes.Equal(raw, artifact.CanonicalJSON()) {
		return domain.SetupPlan{}, domain.CanonicalArtifact{}, fmt.Errorf("authorized setup plan is not canonical JSON")
	}
	return plan, artifact, nil
}

func readCanonicalAuthorization(path string) (domain.PlanAuthorizationReference, domain.CanonicalArtifact, error) {
	raw, err := boundedio.ReadRegularFile(path, "setup plan authorization", domain.MaxPlanAuthorizationBytes)
	if err != nil {
		return domain.PlanAuthorizationReference{}, domain.CanonicalArtifact{}, err
	}
	authorization, err := domain.DecodePlanAuthorizationReference(bytes.NewReader(raw))
	if err != nil {
		return domain.PlanAuthorizationReference{}, domain.CanonicalArtifact{}, fmt.Errorf("decode setup plan authorization: %w", err)
	}
	artifact, err := domain.FreezePlanAuthorizationReference(authorization)
	if err != nil {
		return domain.PlanAuthorizationReference{}, domain.CanonicalArtifact{}, err
	}
	if !bytes.Equal(raw, artifact.CanonicalJSON()) {
		return domain.PlanAuthorizationReference{}, domain.CanonicalArtifact{}, fmt.Errorf("setup plan authorization is not canonical JSON")
	}
	return authorization, artifact, nil
}

func readCanonicalReleaseMetadata(path string) ([]byte, error) {
	raw, err := boundedio.ReadRegularFile(path, "release metadata", maxBundleContractBytes)
	if err != nil {
		return nil, err
	}
	if _, err := releasebundle.DecodeReleaseMetadata(bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("decode release metadata: %w", err)
	}
	return raw, nil
}
