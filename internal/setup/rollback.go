package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/releasebundle"
)

const (
	RecoverySetSchemaV1    = "goalrail.setup-recovery/v1"
	RollbackResultSchemaV1 = "goalrail.setup-rollback-result/v1"
	maxRecoverySetBytes    = int64(4 << 20)
)

var (
	ErrRecoveryInvalid      = errors.New("setup recovery set is invalid")
	ErrRollbackStateChanged = errors.New("setup rollback target state changed")
)

type recoveryMutationState string

const (
	recoveryPending    recoveryMutationState = "pending"
	recoveryPrepared   recoveryMutationState = "prepared"
	recoveryApplied    recoveryMutationState = "applied"
	recoveryRolledBack recoveryMutationState = "rolled_back"
)

// RecoverySet is deliberately machine-local. BeforeBytes may contain the
// bounded prior scaffold configuration, so the artifact is always written as
// a private 0600 file and is never embedded in command results or receipts.
type RecoverySet struct {
	Schema              string              `json:"schema"`
	ProjectID           domain.ProjectID    `json:"project_id"`
	PlanDigest          domain.SHA256Digest `json:"plan_digest"`
	AuthorizationDigest domain.SHA256Digest `json:"authorization_digest"`
	Home                string              `json:"home"`
	RepositoryRoot      string              `json:"repository_root"`
	Scaffold            ambient.Scaffold    `json:"scaffold"`
	ManifestJSON        []byte              `json:"manifest_json"`
	Mutations           []RecoveryMutation  `json:"mutations"`
}

type RecoveryMutation struct {
	ID           string                   `json:"id"`
	Kind         domain.SetupMutationKind `json:"kind"`
	Scope        domain.SetupScope        `json:"scope"`
	Target       string                   `json:"target"`
	State        recoveryMutationState    `json:"state"`
	BeforeExists bool                     `json:"before_exists"`
	BeforeDigest *domain.SHA256Digest     `json:"before_digest,omitempty"`
	BeforeMode   uint32                   `json:"before_mode,omitempty"`
	BeforeBytes  []byte                   `json:"before_bytes,omitempty"`
	AfterDigest  domain.SHA256Digest      `json:"after_digest"`
}

type recoveryStore struct {
	path   string
	digest domain.SHA256Digest
	set    RecoverySet
}

type RollbackOptions struct {
	Home           string
	RepositoryRoot string
	RecoveryPath   string
}

type RollbackResult struct {
	Schema                string              `json:"schema"`
	RolledBack            bool                `json:"rolled_back"`
	Changed               bool                `json:"changed"`
	ProjectID             domain.ProjectID    `json:"project_id"`
	PlanDigest            domain.SHA256Digest `json:"plan_digest"`
	AuthorizationDigest   domain.SHA256Digest `json:"authorization_digest"`
	RecoveryPath          string              `json:"recovery_path"`
	RecoveryDigest        domain.SHA256Digest `json:"recovery_digest"`
	RolledBackMutationIDs []string            `json:"rolled_back_mutation_ids"`
	PendingMutationIDs    []string            `json:"pending_mutation_ids"`
	FailedMutationID      string              `json:"failed_mutation_id,omitempty"`
}

func createRecoveryStore(plan domain.SetupPlan, verified VerifiedPlan, contract applyContract) (*recoveryStore, error) {
	set := RecoverySet{
		Schema:              RecoverySetSchemaV1,
		ProjectID:           plan.ProjectID,
		PlanDigest:          verified.attempt.PlanArtifact().Digest(),
		AuthorizationDigest: verified.attempt.AuthorizationArtifact().Digest(),
		Home:                contract.home,
		RepositoryRoot:      contract.repository,
		Scaffold:            verified.options.Scaffold,
		ManifestJSON:        append([]byte(nil), verified.manifestRaw...),
		Mutations:           make([]RecoveryMutation, 0, len(contract.mutations)),
	}
	for _, mutationID := range orderedMutationIDs(contract) {
		mutation := contract.mutations[mutationID]
		set.Mutations = append(set.Mutations, RecoveryMutation{
			ID: mutation.ID, Kind: mutation.Kind, Scope: mutation.Scope,
			Target: absoluteMutationTarget(contract, mutation), State: recoveryPending,
			AfterDigest: mutation.DesiredDigest,
		})
	}
	path, err := recoverySetPath(contract.home, set.PlanDigest, set.AuthorizationDigest)
	if err != nil {
		return nil, err
	}
	store := &recoveryStore{path: path, set: set}
	if err := store.create(); err != nil {
		return nil, err
	}
	return store, nil
}

func absoluteMutationTarget(contract applyContract, mutation domain.SetupMutation) string {
	if mutation.Scope == domain.SetupScopeRepository {
		return filepath.Join(contract.repository, filepath.FromSlash(mutation.Path))
	}
	return mutation.Path
}

func (store *recoveryStore) prepare(contract applyContract, verified VerifiedPlan, mutationID string, mutation domain.SetupMutation) error {
	entry, err := store.mutation(mutationID)
	if err != nil {
		return err
	}
	if entry.State != recoveryPending {
		return fmt.Errorf("%w: mutation %s is not pending", ErrRecoveryInvalid, mutationID)
	}

	switch mutationID {
	case "install-bundle":
		if _, exists, err := inspectSafePath(contract.home, contract.versionRoot); err != nil {
			return err
		} else if exists {
			return fmt.Errorf("%w: bundle destination now exists", ErrApplyStateChanged)
		}
	case "select-executable":
		digest, mode, exists, err := inspectRecoveryFile(contract.stableExecutable, maxVerifiedBundleFileBytes)
		if err != nil {
			return err
		}
		if !equalDigestPointers(recoveryDigestPointer(digest, exists), mutation.ExpectedBeforeDigest) {
			return fmt.Errorf("%w: executable state differs before recovery capture", ErrApplyStateChanged)
		}
		entry.BeforeExists = exists
		entry.BeforeDigest = recoveryDigestPointer(digest, exists)
		entry.BeforeMode = uint32(mode.Perm())
		if exists && digest != mutation.DesiredDigest {
			return fmt.Errorf("%w: executable replacement would require retaining unplanned prior bytes", ErrApplyInvalid)
		}
	case "install-hook":
		connection, err := ambient.PlanRegistration(contract.target, contract.stableExecutable)
		if err != nil {
			return err
		}
		preview, err := ambient.PreviewConnection(connection)
		if err != nil {
			return err
		}
		if len(preview.Before) > maxAppliedConfig || len(preview.After) > maxAppliedConfig {
			return fmt.Errorf("scaffold recovery exceeds %d bytes", maxAppliedConfig)
		}
		beforeDigest := digestPointerOrNil(preview.Before, preview.BeforeExists)
		if !equalDigestPointers(beforeDigest, mutation.ExpectedBeforeDigest) || digestBytes(preview.After) != mutation.DesiredDigest {
			return fmt.Errorf("%w: scaffold state differs before recovery capture", ErrApplyStateChanged)
		}
		allowedRoot := contract.home
		if contract.target.Scope == ambient.ScopeRepository {
			allowedRoot = contract.repository
		}
		info, exists, err := inspectSafePath(allowedRoot, contract.target.Path)
		if err != nil {
			return err
		}
		entry.BeforeExists = preview.BeforeExists
		entry.BeforeDigest = beforeDigest
		entry.BeforeBytes = append([]byte(nil), preview.Before...)
		if exists {
			entry.BeforeMode = uint32(info.Mode().Perm())
		}
	default:
		return fmt.Errorf("%w: unknown recovery mutation %s", ErrRecoveryInvalid, mutationID)
	}

	entry.State = recoveryPrepared
	return store.persist()
}

func (store *recoveryStore) markApplied(mutationID string) error {
	entry, err := store.mutation(mutationID)
	if err != nil {
		return err
	}
	if entry.State != recoveryPrepared {
		return fmt.Errorf("%w: mutation %s is not prepared", ErrRecoveryInvalid, mutationID)
	}
	entry.State = recoveryApplied
	return store.persist()
}

func (store *recoveryStore) mutation(id string) (*RecoveryMutation, error) {
	for index := range store.set.Mutations {
		if store.set.Mutations[index].ID == id {
			return &store.set.Mutations[index], nil
		}
	}
	return nil, fmt.Errorf("%w: mutation %s is absent", ErrRecoveryInvalid, id)
}

func (store *recoveryStore) create() error {
	raw, err := encodeRecoverySet(store.set)
	if err != nil {
		return err
	}
	created, err := ensureDirectoryTree(store.set.Home, filepath.Dir(store.path))
	if err != nil {
		return err
	}
	if err := writeExclusiveRegularFile(store.path, raw, 0o600); err != nil {
		removeEmptyDirectories(created)
		return fmt.Errorf("%w: reserve recovery path: %v", ErrRecoveryInvalid, err)
	}
	store.digest = digestBytes(raw)
	return nil
}

func (store *recoveryStore) persist() error {
	raw, err := encodeRecoverySet(store.set)
	if err != nil {
		return err
	}
	currentDigest, mode, err := digestRegularFile(store.path, maxRecoverySetBytes)
	if err != nil || currentDigest != store.digest || mode.Perm() != 0o600 {
		return fmt.Errorf("%w: recovery artifact changed before update", ErrRecoveryInvalid)
	}
	temporary, err := os.CreateTemp(filepath.Dir(store.path), ".goalrail-recovery-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	currentDigest, mode, err = digestRegularFile(store.path, maxRecoverySetBytes)
	if err != nil || currentDigest != store.digest || mode.Perm() != 0o600 {
		return fmt.Errorf("%w: recovery artifact changed before replacement", ErrRecoveryInvalid)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return err
	}
	store.digest = digestBytes(raw)
	return nil
}

// Rollback restores completed setup mutations in reverse order. It refuses a
// target whose current bytes no longer equal either the captured before state
// or the exact applied state, so unrelated later changes are never overwritten.
func Rollback(ctx context.Context, options RollbackOptions) (RollbackResult, error) {
	options, err := normalizeRollbackOptions(options)
	if err != nil {
		return RollbackResult{}, err
	}
	store, err := openRecoveryStore(options)
	if err != nil {
		return RollbackResult{}, err
	}
	result := RollbackResult{
		Schema: RollbackResultSchemaV1, ProjectID: store.set.ProjectID,
		PlanDigest: store.set.PlanDigest, AuthorizationDigest: store.set.AuthorizationDigest,
		RecoveryPath: store.path, RecoveryDigest: store.digest,
		RolledBackMutationIDs: []string{}, PendingMutationIDs: pendingRecoveryMutationIDs(store.set),
	}
	manifest, err := releasebundle.DecodeSetupBundleManifest(bytes.NewReader(store.set.ManifestJSON))
	if err != nil {
		return result, fmt.Errorf("%w: decode retained bundle manifest: %v", ErrRecoveryInvalid, err)
	}

	for index := len(store.set.Mutations) - 1; index >= 0; index-- {
		entry := &store.set.Mutations[index]
		if entry.State == recoveryPending || entry.State == recoveryRolledBack {
			continue
		}
		if err := ctx.Err(); err != nil {
			result.FailedMutationID = entry.ID
			return result, err
		}
		changed, err := rollbackRecoveryMutation(*entry, manifest, store.set.ManifestJSON)
		if err != nil {
			result.FailedMutationID = entry.ID
			result.RecoveryDigest = store.digest
			result.PendingMutationIDs = pendingRecoveryMutationIDs(store.set)
			return result, fmt.Errorf("rollback mutation %s: %w", entry.ID, err)
		}
		entry.State = recoveryRolledBack
		if err := store.persist(); err != nil {
			result.FailedMutationID = entry.ID
			return result, fmt.Errorf("persist rollback state for %s: %w", entry.ID, err)
		}
		result.RecoveryDigest = store.digest
		result.Changed = result.Changed || changed
		result.RolledBackMutationIDs = append(result.RolledBackMutationIDs, entry.ID)
		result.PendingMutationIDs = pendingRecoveryMutationIDs(store.set)
	}
	result.RolledBack = len(result.PendingMutationIDs) == 0
	return result, nil
}

func rollbackRecoveryMutation(entry RecoveryMutation, manifest releasebundle.SetupBundleManifest, manifestRaw []byte) (bool, error) {
	switch entry.ID {
	case "install-bundle":
		if entry.BeforeExists {
			return false, fmt.Errorf("%w: bundle recovery unexpectedly has prior state", ErrRecoveryInvalid)
		}
		if _, err := os.Lstat(entry.Target); errors.Is(err, fs.ErrNotExist) {
			return false, nil
		} else if err != nil {
			return false, err
		}
		if err := verifyInstalledBundle(entry.Target, manifest, manifestRaw); err != nil {
			return false, fmt.Errorf("%w: installed bundle is no longer exact: %v", ErrRollbackStateChanged, err)
		}
		if err := removeVerifiedBundle(entry.Target, manifest); err != nil {
			return false, err
		}
		return true, nil
	case "select-executable":
		return rollbackRegularFile(entry, maxVerifiedBundleFileBytes)
	case "install-hook":
		return rollbackRegularFile(entry, maxAppliedConfig)
	default:
		return false, fmt.Errorf("%w: unknown mutation %s", ErrRecoveryInvalid, entry.ID)
	}
}

func rollbackRegularFile(entry RecoveryMutation, limit int64) (bool, error) {
	currentDigest, currentMode, exists, err := inspectRecoveryFile(entry.Target, limit)
	if err != nil {
		return false, err
	}
	if matchesBeforeState(entry, currentDigest, currentMode, exists) {
		return false, nil
	}
	if !exists || currentDigest != entry.AfterDigest {
		return false, fmt.Errorf("%w: %s differs from the exact applied state", ErrRollbackStateChanged, entry.Target)
	}
	if !entry.BeforeExists {
		if err := checkRecoveryAppliedState(entry.Target, entry.AfterDigest, limit); err != nil {
			return false, err
		}
		if err := os.Remove(entry.Target); err != nil {
			return false, err
		}
		return true, nil
	}
	if entry.BeforeDigest == nil {
		return false, fmt.Errorf("%w: missing prior digest for %s", ErrRecoveryInvalid, entry.ID)
	}
	if len(entry.BeforeBytes) == 0 && *entry.BeforeDigest == entry.AfterDigest {
		if err := checkRecoveryAppliedState(entry.Target, entry.AfterDigest, limit); err != nil {
			return false, err
		}
		if err := os.Chmod(entry.Target, fs.FileMode(entry.BeforeMode)); err != nil {
			return false, err
		}
		return true, nil
	}
	if digestBytes(entry.BeforeBytes) != *entry.BeforeDigest {
		return false, fmt.Errorf("%w: retained prior bytes do not match their digest", ErrRecoveryInvalid)
	}
	if err := replaceRegularFile(entry.Target, entry.BeforeBytes, fs.FileMode(entry.BeforeMode), entry.AfterDigest, limit); err != nil {
		return false, err
	}
	return true, nil
}

func matchesBeforeState(entry RecoveryMutation, current domain.SHA256Digest, currentMode fs.FileMode, exists bool) bool {
	if entry.BeforeExists != exists {
		return false
	}
	if !exists {
		return true
	}
	return entry.BeforeDigest != nil && current == *entry.BeforeDigest && currentMode.Perm() == fs.FileMode(entry.BeforeMode).Perm()
}

func removeVerifiedBundle(root string, manifest releasebundle.SetupBundleManifest) error {
	paths := make([]string, 0, len(manifest.Files)+1)
	for _, record := range manifest.Files {
		path := filepath.Join(root, filepath.FromSlash(record.Path))
		if !pathWithin(root, path) {
			return fmt.Errorf("%w: bundle record escapes recovery root", ErrRecoveryInvalid)
		}
		paths = append(paths, path)
	}
	paths = append(paths, filepath.Join(root, filepath.FromSlash(manifest.ManifestPath)))
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	for index := len(paths) - 1; index >= 0; index-- {
		directory := filepath.Dir(paths[index])
		for directory != root && pathWithin(root, directory) {
			_ = os.Remove(directory)
			directory = filepath.Dir(directory)
		}
	}
	if err := os.Remove(root); err != nil {
		return fmt.Errorf("%w: bundle root gained unrelated content: %v", ErrRollbackStateChanged, err)
	}
	return nil
}

func replaceRegularFile(path string, raw []byte, mode fs.FileMode, expected domain.SHA256Digest, limit int64) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".goalrail-rollback-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(raw); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(mode.Perm()); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := checkRecoveryAppliedState(path, expected, limit); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func checkRecoveryAppliedState(path string, expected domain.SHA256Digest, limit int64) error {
	digest, _, exists, err := inspectRecoveryFile(path, limit)
	if err != nil {
		return err
	}
	if !exists || digest != expected {
		return fmt.Errorf("%w: %s changed immediately before rollback", ErrRollbackStateChanged, path)
	}
	return nil
}

func inspectRecoveryFile(path string, limit int64) (domain.SHA256Digest, fs.FileMode, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", 0, false, nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", 0, false, fmt.Errorf("%w: %s is not a safe regular file", ErrRollbackStateChanged, path)
	}
	digest, mode, err := digestRegularFile(path, limit)
	return digest, mode, true, err
}

func recoveryDigestPointer(digest domain.SHA256Digest, exists bool) *domain.SHA256Digest {
	if !exists {
		return nil
	}
	value := digest
	return &value
}

func normalizeRollbackOptions(options RollbackOptions) (RollbackOptions, error) {
	if strings.TrimSpace(options.Home) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return RollbackOptions{}, err
		}
		options.Home = home
	}
	for label, value := range map[string]*string{
		"home": &options.Home, "repository": &options.RepositoryRoot, "recovery": &options.RecoveryPath,
	} {
		if strings.TrimSpace(*value) == "" {
			if label == "repository" {
				*value = "."
			} else {
				return RollbackOptions{}, fmt.Errorf("%s path is required", label)
			}
		}
		absolute, err := filepath.Abs(*value)
		if err != nil {
			return RollbackOptions{}, fmt.Errorf("resolve %s path: %w", label, err)
		}
		*value = filepath.Clean(absolute)
	}
	if options.Home == string(filepath.Separator) {
		return RollbackOptions{}, fmt.Errorf("rollback refuses the filesystem root as user home")
	}
	return options, nil
}

func openRecoveryStore(options RollbackOptions) (*recoveryStore, error) {
	raw, digest, set, err := readRecoverySet(options.RecoveryPath)
	if err != nil {
		return nil, err
	}
	_ = raw
	if set.Home != options.Home || set.RepositoryRoot != options.RepositoryRoot {
		return nil, fmt.Errorf("%w: recovery roots do not match the explicit rollback roots", ErrRecoveryInvalid)
	}
	expected, err := recoverySetPath(options.Home, set.PlanDigest, set.AuthorizationDigest)
	if err != nil || expected != options.RecoveryPath {
		return nil, fmt.Errorf("%w: recovery path is not the exact plan path", ErrRecoveryInvalid)
	}
	if err := validateRecoverySet(set); err != nil {
		return nil, err
	}
	return &recoveryStore{path: options.RecoveryPath, digest: digest, set: set}, nil
}

func validateRecoverySet(set RecoverySet) error {
	if set.Schema != RecoverySetSchemaV1 || len(set.Mutations) == 0 || len(set.Mutations) > 3 {
		return ErrRecoveryInvalid
	}
	manifest, err := releasebundle.DecodeSetupBundleManifest(bytes.NewReader(set.ManifestJSON))
	if err != nil {
		return fmt.Errorf("%w: manifest: %v", ErrRecoveryInvalid, err)
	}
	identity, ok := binaryIdentity(manifest, "bin/gr")
	if !ok {
		return fmt.Errorf("%w: manifest has no Goalrail executable", ErrRecoveryInvalid)
	}
	target, err := ambient.RegistrationTarget(set.Scaffold, set.Home, set.RepositoryRoot)
	if err != nil {
		return fmt.Errorf("%w: scaffold target: %v", ErrRecoveryInvalid, err)
	}
	expectedTargets := map[string]string{
		"install-bundle":    filepath.Join(set.Home, ".local", "share", "goalrail", "bundles", manifest.ReleaseVersion, manifest.Platform.Key()),
		"select-executable": filepath.Join(set.Home, ".local", "bin", "gr"),
		"install-hook":      target.Path,
	}
	expectedKinds := map[string]domain.SetupMutationKind{
		"install-bundle":    domain.SetupMutationInstallFile,
		"select-executable": domain.SetupMutationSelectExecutable,
		"install-hook":      domain.SetupMutationInstallHook,
	}
	expectedScopes := map[string]domain.SetupScope{
		"install-bundle":    domain.SetupScopeUserLocal,
		"select-executable": domain.SetupScopeUserLocal,
		"install-hook":      domain.SetupScopeUserLocal,
	}
	if target.Scope == ambient.ScopeRepository {
		expectedScopes["install-hook"] = domain.SetupScopeRepository
	}
	seen := make(map[string]struct{}, len(set.Mutations))
	lastOrder := -1
	mutationOrder := map[string]int{"install-bundle": 0, "select-executable": 1, "install-hook": 2}
	for _, entry := range set.Mutations {
		order, known := mutationOrder[entry.ID]
		if _, duplicate := seen[entry.ID]; duplicate || !known || order <= lastOrder || entry.Target != expectedTargets[entry.ID] || entry.Kind != expectedKinds[entry.ID] || entry.Scope != expectedScopes[entry.ID] {
			return fmt.Errorf("%w: unexpected or duplicate mutation %s", ErrRecoveryInvalid, entry.ID)
		}
		lastOrder = order
		seen[entry.ID] = struct{}{}
		if entry.State != recoveryPending && entry.State != recoveryPrepared && entry.State != recoveryApplied && entry.State != recoveryRolledBack {
			return fmt.Errorf("%w: invalid mutation state", ErrRecoveryInvalid)
		}
		if entry.ID == "select-executable" && entry.AfterDigest != domain.SHA256Digest(identity.SHA256) {
			return fmt.Errorf("%w: executable digest differs from manifest", ErrRecoveryInvalid)
		}
		if entry.BeforeExists != (entry.BeforeDigest != nil) || len(entry.BeforeBytes) > maxAppliedConfig {
			return fmt.Errorf("%w: invalid prior state for %s", ErrRecoveryInvalid, entry.ID)
		}
		if entry.State == recoveryPending && (entry.BeforeExists || entry.BeforeDigest != nil || entry.BeforeMode != 0 || len(entry.BeforeBytes) != 0) {
			return fmt.Errorf("%w: pending mutation %s contains captured state", ErrRecoveryInvalid, entry.ID)
		}
		if entry.ID != "install-hook" && len(entry.BeforeBytes) != 0 {
			return fmt.Errorf("%w: mutation %s retains unneeded prior bytes", ErrRecoveryInvalid, entry.ID)
		}
	}
	return nil
}

func pendingRecoveryMutationIDs(set RecoverySet) []string {
	ids := make([]string, 0, len(set.Mutations))
	for _, entry := range set.Mutations {
		if entry.State == recoveryPrepared || entry.State == recoveryApplied {
			ids = append(ids, entry.ID)
		}
	}
	return ids
}

func recoverySetPath(home string, plan, authorization domain.SHA256Digest) (string, error) {
	planID, err := digestFilename(plan)
	if err != nil {
		return "", err
	}
	authorizationID, err := digestFilename(authorization)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "goalrail", "recovery", planID+"-"+authorizationID+".json"), nil
}

func digestFilename(digest domain.SHA256Digest) (string, error) {
	raw := strings.TrimPrefix(string(digest), "sha256:")
	if len(raw) != 64 {
		return "", fmt.Errorf("%w: invalid digest filename", ErrRecoveryInvalid)
	}
	for _, character := range raw {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return "", fmt.Errorf("%w: invalid digest filename", ErrRecoveryInvalid)
		}
	}
	return raw, nil
}

func encodeRecoverySet(set RecoverySet) ([]byte, error) {
	raw, err := json.Marshal(set)
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	if int64(len(raw)) > maxRecoverySetBytes {
		return nil, fmt.Errorf("%w: artifact exceeds %d bytes", ErrRecoveryInvalid, maxRecoverySetBytes)
	}
	return raw, nil
}

func readRecoverySet(path string) ([]byte, domain.SHA256Digest, RecoverySet, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, "", RecoverySet{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 || info.Size() < 0 || info.Size() > maxRecoverySetBytes {
		return nil, "", RecoverySet{}, fmt.Errorf("%w: recovery path is not a private bounded regular file", ErrRecoveryInvalid)
	}
	raw, err := io.ReadAll(io.LimitReader(file, maxRecoverySetBytes+1))
	if err != nil || int64(len(raw)) != info.Size() {
		return nil, "", RecoverySet{}, fmt.Errorf("%w: recovery artifact changed while being read", ErrRecoveryInvalid)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var set RecoverySet
	if err := decoder.Decode(&set); err != nil {
		return nil, "", RecoverySet{}, fmt.Errorf("%w: decode: %v", ErrRecoveryInvalid, err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, "", RecoverySet{}, fmt.Errorf("%w: trailing JSON value", ErrRecoveryInvalid)
	}
	return raw, digestBytes(raw), set, nil
}
