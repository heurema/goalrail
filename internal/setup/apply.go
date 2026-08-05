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
	"reflect"
	"strconv"
	"syscall"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/releasebundle"
)

const (
	ApplyResultSchemaV1 = "goalrail.setup-apply-result/v1"
	maxAppliedConfig    = 1 << 20
)

var (
	ErrApplyInvalid      = errors.New("verified setup plan cannot be applied")
	ErrApplyStateChanged = errors.New("setup target state changed")
)

// ApplyResult is the bounded mutation inventory used to construct the canonical
// setup receipt. RecoveryPath identifies the private machine-local recovery set;
// prior configuration bytes never enter this result or the receipt.
type ApplyResult struct {
	Schema              string              `json:"schema"`
	Applied             bool                `json:"applied"`
	Changed             bool                `json:"changed"`
	ProjectID           domain.ProjectID    `json:"project_id"`
	PlanDigest          domain.SHA256Digest `json:"plan_digest"`
	AuthorizationDigest domain.SHA256Digest `json:"authorization_digest"`
	ReleaseVersion      string              `json:"release_version"`
	Platform            string              `json:"platform"`
	AppliedMutationIDs  []string            `json:"applied_mutation_ids"`
	PendingMutationIDs  []string            `json:"pending_mutation_ids"`
	FailedMutationID    string              `json:"failed_mutation_id,omitempty"`
	FailureStage        string              `json:"failure_stage,omitempty"`
	RecoveryPath        string              `json:"recovery_path,omitempty"`
	RecoveryDigest      domain.SHA256Digest `json:"recovery_digest,omitempty"`
	PendingTrustStepIDs []string            `json:"pending_trust_step_ids"`
}

type applyHooks struct {
	beforeMutation func(string) error
}

type applyContract struct {
	home             string
	repository       string
	versionRoot      string
	stableExecutable string
	target           ambient.Target
	goalrailFile     releasebundle.ManifestFile
	mutations        map[string]domain.SetupMutation
}

// Apply revalidates a non-constructible VerifiedPlan, then installs only the
// exact generated mutations. It never performs provider trust or an external
// service action.
func Apply(ctx context.Context, verified VerifiedPlan) (ApplyResult, error) {
	return applyVerified(ctx, verified, applyHooks{})
}

func applyVerified(ctx context.Context, verified VerifiedPlan, hooks applyHooks) (ApplyResult, error) {
	if !verified.verification.Verified || verified.options.BundleRoot == "" {
		return ApplyResult{}, ErrApplyInvalid
	}

	current, err := VerifyPlan(ctx, verified.options)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("%w: final plan verification: %v", ErrApplyStateChanged, err)
	}
	if current.verification != verified.verification {
		return ApplyResult{}, fmt.Errorf("%w: verified identities changed", ErrApplyStateChanged)
	}
	plan, err := domain.DecodeSetupPlan(bytes.NewReader(current.attempt.PlanArtifact().CanonicalJSON()))
	if err != nil {
		return ApplyResult{}, fmt.Errorf("%w: decode verified plan: %v", ErrApplyInvalid, err)
	}
	contract, err := buildApplyContract(plan, current)
	if err != nil {
		return ApplyResult{}, err
	}

	result := ApplyResult{
		Schema:              ApplyResultSchemaV1,
		ProjectID:           plan.ProjectID,
		PlanDigest:          current.attempt.PlanArtifact().Digest(),
		AuthorizationDigest: current.attempt.AuthorizationArtifact().Digest(),
		ReleaseVersion:      current.manifest.ReleaseVersion,
		Platform:            plan.Platform,
		AppliedMutationIDs:  []string{},
		PendingMutationIDs:  orderedMutationIDs(contract),
		PendingTrustStepIDs: make([]string, 0, len(plan.TrustSteps)),
	}
	for _, step := range plan.TrustSteps {
		result.PendingTrustStepIDs = append(result.PendingTrustStepIDs, step.ID)
	}

	var recovery *recoveryStore
	if len(result.PendingMutationIDs) != 0 {
		recovery, err = createRecoveryStore(plan, current, contract)
		if err != nil {
			result.FailureStage = "create-recovery-set"
			return result, fmt.Errorf("create setup recovery set: %w", err)
		}
		result.RecoveryPath = recovery.path
		result.RecoveryDigest = recovery.digest
	}

	for _, mutationID := range orderedMutationIDs(contract) {
		mutation, ok := contract.mutations[mutationID]
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			markApplyFailure(&result, mutationID)
			result.FailureStage = "before-mutation"
			return result, err
		}
		if hooks.beforeMutation != nil {
			if err := hooks.beforeMutation(mutationID); err != nil {
				markApplyFailure(&result, mutationID)
				result.FailureStage = "before-mutation"
				return result, err
			}
		}
		if err := recovery.prepare(contract, current, mutationID, mutation); err != nil {
			markApplyFailure(&result, mutationID)
			result.FailureStage = "prepare-recovery"
			result.RecoveryDigest = recovery.digest
			return result, fmt.Errorf("prepare recovery for mutation %s: %w", mutationID, err)
		}
		result.RecoveryDigest = recovery.digest

		switch mutationID {
		case "install-bundle":
			err = installVersionedBundle(contract.home, contract.versionRoot, current.bundleRoot, current.manifestRaw, current.manifest)
		case "select-executable":
			err = selectStableExecutable(contract.home, contract.versionRoot, contract.stableExecutable, contract.goalrailFile, mutation)
		case "install-hook":
			err = applyScaffoldConfiguration(contract, mutation)
		}
		if err != nil {
			markApplyFailure(&result, mutationID)
			result.FailureStage = "apply-mutation"
			result.RecoveryDigest = recovery.digest
			return result, fmt.Errorf("apply mutation %s: %w", mutationID, err)
		}
		result.AppliedMutationIDs = append(result.AppliedMutationIDs, mutationID)
		result.Changed = true
		result.PendingMutationIDs = pendingMutationIDs(contract, result.AppliedMutationIDs)
		if err := recovery.markApplied(mutationID); err != nil {
			markApplyFailure(&result, mutationID)
			result.FailureStage = "persist-recovery-state"
			return result, fmt.Errorf("persist recovery after mutation %s: %w", mutationID, err)
		}
		result.RecoveryDigest = recovery.digest
	}

	if err := verifyInstalledBundle(contract.versionRoot, current.manifest, current.manifestRaw); err != nil {
		result.FailureStage = "verify-durable-bundle"
		return result, fmt.Errorf("verify durable bundle: %w", err)
	}
	if err := verifyManifestFile(contract.stableExecutable, contract.goalrailFile); err != nil {
		result.FailureStage = "verify-selected-executable"
		return result, fmt.Errorf("verify selected executable: %w", err)
	}
	connection, err := ambient.PlanRegistration(contract.target, contract.stableExecutable)
	if err != nil {
		result.FailureStage = "verify-scaffold-configuration"
		return result, fmt.Errorf("verify scaffold configuration: %w", err)
	}
	if !connection.AlreadyPresent {
		result.FailureStage = "verify-scaffold-configuration"
		return result, fmt.Errorf("verify scaffold configuration: registration is not current")
	}

	result.Applied = true
	return result, nil
}

func orderedMutationIDs(contract applyContract) []string {
	ids := make([]string, 0, len(contract.mutations))
	for _, mutationID := range []string{"install-bundle", "select-executable", "install-hook"} {
		if _, ok := contract.mutations[mutationID]; ok {
			ids = append(ids, mutationID)
		}
	}
	return ids
}

func pendingMutationIDs(contract applyContract, applied []string) []string {
	done := make(map[string]struct{}, len(applied))
	for _, mutationID := range applied {
		done[mutationID] = struct{}{}
	}
	pending := make([]string, 0, len(contract.mutations)-len(done))
	for _, mutationID := range orderedMutationIDs(contract) {
		if _, ok := done[mutationID]; !ok {
			pending = append(pending, mutationID)
		}
	}
	return pending
}

func markApplyFailure(result *ApplyResult, mutationID string) {
	result.Applied = false
	result.Changed = len(result.AppliedMutationIDs) != 0
	result.FailedMutationID = mutationID
}

func buildApplyContract(plan domain.SetupPlan, verified VerifiedPlan) (applyContract, error) {
	if plan.State != domain.SetupPlanComplete || plan.ProjectCodeWrites != 0 {
		return applyContract{}, fmt.Errorf("%w: plan is not complete or permits project code writes", ErrApplyInvalid)
	}
	for _, prerequisite := range plan.Prerequisites {
		if !prerequisite.Satisfied {
			return applyContract{}, fmt.Errorf("%w: prerequisite %s is unsatisfied", ErrApplyInvalid, prerequisite.ID)
		}
	}

	versionRoot := filepath.Join(verified.options.Home, ".local", "share", "goalrail", "bundles", verified.manifest.ReleaseVersion, verified.manifest.Platform.Key())
	stableExecutable := filepath.Join(verified.options.Home, ".local", "bin", "gr")
	expectedComponents, err := componentsFromManifest(verified.manifest, versionRoot)
	if err != nil || !reflect.DeepEqual(plan.Components, expectedComponents) {
		return applyContract{}, fmt.Errorf("%w: component destinations or identities differ from the verified manifest", ErrApplyInvalid)
	}
	identity, ok := binaryIdentity(verified.manifest, "bin/gr")
	if !ok {
		return applyContract{}, fmt.Errorf("%w: verified bundle has no Goalrail executable identity", ErrApplyInvalid)
	}
	goalrailFile, ok := manifestFile(verified.manifest, identity.Path)
	if !ok || goalrailFile.SHA256 != identity.SHA256 {
		return applyContract{}, fmt.Errorf("%w: Goalrail executable identity disagrees with the manifest", ErrApplyInvalid)
	}

	target, err := ambient.RegistrationTarget(verified.options.Scaffold, verified.options.Home, verified.options.RepositoryRoot)
	if err != nil {
		return applyContract{}, fmt.Errorf("%w: resolve scaffold target: %v", ErrApplyInvalid, err)
	}
	hookPath := target.Path
	hookScope := domain.SetupScopeUserLocal
	if target.Scope == ambient.ScopeRepository {
		relative, relErr := filepath.Rel(verified.options.RepositoryRoot, target.Path)
		if relErr != nil || relative == "." || !pathWithin(verified.options.RepositoryRoot, target.Path) {
			return applyContract{}, fmt.Errorf("%w: scaffold target escapes the repository", ErrApplyInvalid)
		}
		hookPath = filepath.ToSlash(relative)
		hookScope = domain.SetupScopeRepository
	}

	metadata, err := releasebundle.DecodeReleaseMetadata(bytes.NewReader(verified.metadataRaw))
	if err != nil {
		return applyContract{}, fmt.Errorf("%w: decode release metadata: %v", ErrApplyInvalid, err)
	}
	entry, ok := platformEntry(metadata, verified.manifest.Platform)
	if !ok || metadata.ReleaseVersion != verified.manifest.ReleaseVersion {
		return applyContract{}, fmt.Errorf("%w: release metadata no longer identifies the verified bundle", ErrApplyInvalid)
	}

	expected := map[string]domain.SetupMutation{
		"install-bundle": {
			ID: "install-bundle", Kind: domain.SetupMutationInstallFile, Scope: domain.SetupScopeUserLocal,
			Path: versionRoot, DesiredDigest: domain.SHA256Digest(entry.SetupArchive.SHA256),
		},
		"select-executable": {
			ID: "select-executable", Kind: domain.SetupMutationSelectExecutable, Scope: domain.SetupScopeUserLocal,
			Path: stableExecutable, DesiredDigest: domain.SHA256Digest(identity.SHA256),
		},
		"install-hook": {
			ID: "install-hook", Kind: domain.SetupMutationInstallHook, Scope: hookScope,
			Path: hookPath,
		},
	}
	mutations := make(map[string]domain.SetupMutation, len(plan.Mutations))
	for _, mutation := range plan.Mutations {
		want, known := expected[mutation.ID]
		if !known || mutation.Kind != want.Kind || mutation.Scope != want.Scope || mutation.Path != want.Path {
			return applyContract{}, fmt.Errorf("%w: mutation %s is outside the generated setup boundary", ErrApplyInvalid, mutation.ID)
		}
		if mutation.ID == "install-bundle" {
			if mutation.ExpectedBeforeDigest != nil || mutation.DesiredDigest != want.DesiredDigest {
				return applyContract{}, fmt.Errorf("%w: bundle mutation identity differs", ErrApplyInvalid)
			}
		}
		if mutation.ID == "select-executable" && mutation.DesiredDigest != want.DesiredDigest {
			return applyContract{}, fmt.Errorf("%w: executable mutation identity differs", ErrApplyInvalid)
		}
		mutations[mutation.ID] = mutation
	}
	rollback := make(map[string]domain.SetupRollbackAction, len(plan.Rollback))
	for _, action := range plan.Rollback {
		if _, duplicate := rollback[action.MutationID]; duplicate {
			return applyContract{}, fmt.Errorf("%w: duplicate rollback action for %s", ErrApplyInvalid, action.MutationID)
		}
		mutation, known := mutations[action.MutationID]
		if !known || action.Target != mutation.Path || !equalDigestPointers(action.PriorStateDigest, mutation.ExpectedBeforeDigest) {
			return applyContract{}, fmt.Errorf("%w: rollback action %s is outside the generated setup boundary", ErrApplyInvalid, action.MutationID)
		}
		expectedAction := "remove_owned_path"
		if mutation.ExpectedBeforeDigest != nil {
			expectedAction = "restore_prior_file"
		}
		if action.Action != expectedAction {
			return applyContract{}, fmt.Errorf("%w: rollback action %s has the wrong inverse", ErrApplyInvalid, action.MutationID)
		}
		rollback[action.MutationID] = action
	}
	if len(rollback) != len(mutations) {
		return applyContract{}, fmt.Errorf("%w: rollback inventory does not cover every mutation", ErrApplyInvalid)
	}

	return applyContract{
		home:             verified.options.Home,
		repository:       verified.options.RepositoryRoot,
		versionRoot:      versionRoot,
		stableExecutable: stableExecutable,
		target:           target,
		goalrailFile:     goalrailFile,
		mutations:        mutations,
	}, nil
}

func installVersionedBundle(home, destination, source string, manifestRaw []byte, manifest releasebundle.SetupBundleManifest) (err error) {
	if err := verifyTemporaryBundleFiles(source, manifestRaw, manifest); err != nil {
		return err
	}
	if _, exists, inspectErr := inspectSafePath(home, destination); inspectErr != nil {
		return inspectErr
	} else if exists {
		return fmt.Errorf("%w: bundle destination now exists", ErrApplyStateChanged)
	}
	createdParents, err := ensureDirectoryTree(home, filepath.Dir(destination))
	if err != nil {
		return err
	}
	createdDestination := false
	defer func() {
		if err != nil && createdDestination {
			_ = os.RemoveAll(destination)
		}
		if err != nil {
			removeEmptyDirectories(createdParents)
		}
	}()
	if err = os.Mkdir(destination, 0o755); err != nil {
		return fmt.Errorf("%w: reserve bundle destination: %v", ErrApplyStateChanged, err)
	}
	createdDestination = true

	for _, record := range manifest.Files {
		input := filepath.Join(source, filepath.FromSlash(record.Path))
		output := filepath.Join(destination, filepath.FromSlash(record.Path))
		if !pathWithin(source, input) || !pathWithin(destination, output) {
			return fmt.Errorf("bundle file %s escapes its root", record.Path)
		}
		if _, mkdirErr := ensureDirectoryTree(destination, filepath.Dir(output)); mkdirErr != nil {
			return mkdirErr
		}
		if err = copyManifestRecord(input, output, record); err != nil {
			return err
		}
	}
	manifestPath := filepath.Join(destination, filepath.FromSlash(manifest.ManifestPath))
	if _, err = ensureDirectoryTree(destination, filepath.Dir(manifestPath)); err != nil {
		return err
	}
	if err = writeExclusiveRegularFile(manifestPath, manifestRaw, 0o644); err != nil {
		return err
	}
	if err = verifyInstalledBundle(destination, manifest, manifestRaw); err != nil {
		return err
	}
	return nil
}

func selectStableExecutable(home, versionRoot, destination string, record releasebundle.ManifestFile, mutation domain.SetupMutation) (err error) {
	source := filepath.Join(versionRoot, filepath.FromSlash(record.Path))
	if !pathWithin(versionRoot, source) {
		return fmt.Errorf("%w: executable source escapes the durable bundle", ErrApplyInvalid)
	}
	if err := verifyManifestFile(source, record); err != nil {
		return fmt.Errorf("verify durable executable source: %w", err)
	}
	if err := checkExpectedFileState(destination, mutation.ExpectedBeforeDigest); err != nil {
		return err
	}
	createdParents, err := ensureDirectoryTree(home, filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			removeEmptyDirectories(createdParents)
		}
	}()
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".gr-select-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := copyManifestRecordInto(source, temporary, record); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := checkExpectedFileState(destination, mutation.ExpectedBeforeDigest); err != nil {
		return err
	}
	if mutation.ExpectedBeforeDigest == nil {
		if err := os.Link(temporaryPath, destination); err != nil {
			return fmt.Errorf("%w: publish selected executable: %v", ErrApplyStateChanged, err)
		}
	} else if err := os.Rename(temporaryPath, destination); err != nil {
		return err
	}
	if err := verifyManifestFile(destination, record); err != nil {
		return fmt.Errorf("verify selected executable: %w", err)
	}
	return nil
}

func applyScaffoldConfiguration(contract applyContract, mutation domain.SetupMutation) (err error) {
	connection, err := ambient.PlanRegistration(contract.target, contract.stableExecutable)
	if err != nil {
		return err
	}
	if connection.AlreadyPresent {
		return fmt.Errorf("%w: scaffold registration became current", ErrApplyStateChanged)
	}
	preview, err := ambient.PreviewConnection(connection)
	if err != nil {
		return err
	}
	if len(preview.After) > maxAppliedConfig {
		return fmt.Errorf("scaffold configuration exceeds %d bytes", maxAppliedConfig)
	}
	beforeDigest := digestPointerOrNil(preview.Before, preview.BeforeExists)
	if !equalDigestPointers(beforeDigest, mutation.ExpectedBeforeDigest) || digestBytes(preview.After) != mutation.DesiredDigest {
		return fmt.Errorf("%w: scaffold configuration no longer matches the authorized mutation", ErrApplyStateChanged)
	}

	allowedRoot := contract.home
	if contract.target.Scope == ambient.ScopeRepository {
		allowedRoot = contract.repository
	}
	info, exists, err := inspectSafePath(allowedRoot, contract.target.Path)
	if err != nil {
		return err
	}
	mode := fs.FileMode(0o644)
	if exists {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%w: scaffold target is not regular", ErrApplyStateChanged)
		}
		mode = info.Mode().Perm()
	}
	createdParents, err := ensureDirectoryTree(allowedRoot, filepath.Dir(contract.target.Path))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			removeEmptyDirectories(createdParents)
		}
	}()
	temporary, err := os.CreateTemp(filepath.Dir(contract.target.Path), ".goalrail-config-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(preview.After); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(mode); err != nil {
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

	currentConnection, err := ambient.PlanRegistration(contract.target, contract.stableExecutable)
	if err != nil {
		return err
	}
	currentPreview, err := ambient.PreviewConnection(currentConnection)
	if err != nil {
		return err
	}
	if currentPreview.BeforeExists != preview.BeforeExists || !bytes.Equal(currentPreview.Before, preview.Before) || !bytes.Equal(currentPreview.After, preview.After) {
		return fmt.Errorf("%w: scaffold configuration changed before replacement", ErrApplyStateChanged)
	}
	if !preview.BeforeExists {
		if err := os.Link(temporaryPath, contract.target.Path); err != nil {
			return fmt.Errorf("%w: publish scaffold configuration: %v", ErrApplyStateChanged, err)
		}
	} else if err := os.Rename(temporaryPath, contract.target.Path); err != nil {
		return err
	}
	after, _, err := digestRegularFile(contract.target.Path, maxAppliedConfig)
	if err != nil {
		return err
	}
	if after != mutation.DesiredDigest {
		return fmt.Errorf("scaffold configuration digest differs after write")
	}
	return nil
}

func ensureDirectoryTree(allowedRoot, target string) ([]string, error) {
	root := filepath.Clean(allowedRoot)
	directory := filepath.Clean(target)
	if !filepath.IsAbs(root) || !filepath.IsAbs(directory) || (directory != root && !pathWithin(root, directory)) {
		return nil, fmt.Errorf("directory %q is outside allowed root", directory)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, fmt.Errorf("allowed root %s is not a real directory", root)
	}
	if directory == root {
		return nil, nil
	}
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		return nil, err
	}
	current := root
	var created []string
	for _, part := range splitRelativePath(relative) {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, fs.ErrNotExist) {
			if mkdirErr := os.Mkdir(current, 0o755); mkdirErr != nil {
				removeEmptyDirectories(created)
				return nil, mkdirErr
			}
			created = append(created, current)
			continue
		}
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			removeEmptyDirectories(created)
			return nil, fmt.Errorf("directory path contains an unsafe component %s", current)
		}
	}
	return created, nil
}

func splitRelativePath(relative string) []string {
	var parts []string
	for relative != "." && relative != "" {
		next := filepath.Base(relative)
		parts = append([]string{next}, parts...)
		relative = filepath.Dir(relative)
	}
	return parts
}

func removeEmptyDirectories(paths []string) {
	for index := len(paths) - 1; index >= 0; index-- {
		_ = os.Remove(paths[index])
	}
}

func copyManifestRecord(source, destination string, record releasebundle.ManifestFile) error {
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	if err := copyManifestRecordInto(source, output, record); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}

func copyManifestRecordInto(source string, output *os.File, record releasebundle.ManifestFile) error {
	input, err := os.OpenFile(source, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != record.SizeBytes {
		return fmt.Errorf("bundle source %s is not the expected regular file", record.Path)
	}
	mode, err := strconv.ParseUint(record.Mode, 8, 32)
	if err != nil {
		return err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(output, hash), io.LimitReader(input, record.SizeBytes+1))
	if err != nil || written != record.SizeBytes {
		return fmt.Errorf("copy bundle source %s", record.Path)
	}
	digest := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if digest != record.SHA256 {
		return fmt.Errorf("bundle source digest differs for %s", record.Path)
	}
	if err := output.Chmod(fs.FileMode(mode)); err != nil {
		return err
	}
	return output.Sync()
}

func writeExclusiveRegularFile(path string, raw []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(raw); err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		return err
	}
	return file.Sync()
}

func checkExpectedFileState(path string, expected *domain.SHA256Digest) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		if expected == nil {
			return nil
		}
		return fmt.Errorf("%w: expected executable disappeared", ErrApplyStateChanged)
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: executable target is not a safe regular file", ErrApplyStateChanged)
	}
	if expected == nil {
		return fmt.Errorf("%w: executable target appeared", ErrApplyStateChanged)
	}
	digest, _, err := digestRegularFile(path, maxVerifiedBundleFileBytes)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrApplyStateChanged, err)
	}
	if digest != *expected {
		return fmt.Errorf("%w: executable target digest differs", ErrApplyStateChanged)
	}
	return nil
}

func digestRegularFile(path string, limit int64) (domain.SHA256Digest, fs.FileMode, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > limit {
		return "", 0, fmt.Errorf("path is not a bounded regular file")
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, info.Size()+1))
	if err != nil || written != info.Size() {
		return "", 0, fmt.Errorf("file changed while being read")
	}
	return domain.SHA256Digest("sha256:" + hex.EncodeToString(hash.Sum(nil))), info.Mode().Perm(), nil
}

func digestPointerOrNil(raw []byte, exists bool) *domain.SHA256Digest {
	if !exists {
		return nil
	}
	digest := digestBytes(raw)
	return &digest
}

func equalDigestPointers(left, right *domain.SHA256Digest) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
