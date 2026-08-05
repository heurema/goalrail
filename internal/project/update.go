package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/admissionlocal"
	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
)

const (
	maxProjectUpdateAttempts  = 3
	maxProjectUpdateFileBytes = 512 << 10
)

var (
	ErrProjectLocalEdits      = errors.New("project canon-owned content has local edits")
	ErrProjectSemanticChange  = errors.New("project update would change repository-owned governance")
	errProjectChangedOnUpdate = errors.New("project content changed during update")
)

type CanonUpdateAction string

const (
	CanonUpdateCreated   CanonUpdateAction = "created"
	CanonUpdateUpdated   CanonUpdateAction = "updated"
	CanonUpdateUnchanged CanonUpdateAction = "unchanged"
	CanonUpdateRefused   CanonUpdateAction = "refused"
)

type CanonUpdateOutcome struct {
	Path         string              `json:"path"`
	Action       CanonUpdateAction   `json:"action"`
	Ownership    Ownership           `json:"ownership"`
	BeforeDigest domain.SHA256Digest `json:"before_digest,omitempty"`
	Digest       domain.SHA256Digest `json:"digest,omitempty"`
	Reason       string              `json:"reason,omitempty"`
}

type CanonUpdateInput struct {
	RepositoryRoot    string
	StateRoot         string
	DiscardLocalEdits bool
	Now               func() time.Time
}

type CanonUpdateReport struct {
	Repository          string               `json:"repository"`
	ProjectID           domain.ProjectID     `json:"project_id,omitempty"`
	ContractFrom        string               `json:"contract_from,omitempty"`
	ContractTo          string               `json:"contract_to,omitempty"`
	Canon               domain.SHA256Digest  `json:"canon,omitempty"`
	AlreadyCurrent      bool                 `json:"already_current"`
	Files               []CanonUpdateOutcome `json:"files"`
	ManagedBlocks       []CanonUpdateOutcome `json:"managed_blocks"`
	Backup              string               `json:"backup,omitempty"`
	DiscardedLocalEdits []string             `json:"discarded_local_edits,omitempty"`
	Verified            bool                 `json:"verified"`
	Notes               []string             `json:"notes,omitempty"`
}

type canonUpdatePlan struct {
	path      string
	ownership Ownership
	action    CanonUpdateAction
	desired   []byte
	current   []byte
	exists    bool
	discarded bool
	isBlock   bool
	reason    string
}

type canonUpdateTestHooks struct {
	retainedFiles  []RenderedFile
	retainedBlocks [][]byte
	beforeBackup   func()
	afterBackup    func()
	beforeReplace  func(string)
}

// UpdateProjectCanon reconciles only embedded canon-owned bytes and uniquely
// delimited blocks. Policy and setup-profile values are validated against the
// declaration but never rewritten. It runs no runtime, compiler, network, or
// external activation action.
func UpdateProjectCanon(ctx context.Context, input CanonUpdateInput) (CanonUpdateReport, error) {
	return updateProjectCanon(ctx, input, canonUpdateTestHooks{})
}

func updateProjectCanon(ctx context.Context, input CanonUpdateInput, hooks canonUpdateTestHooks) (CanonUpdateReport, error) {
	now := input.Now
	if now == nil {
		now = time.Now
	}
	inspection, err := Inspect(ctx, input.RepositoryRoot)
	if err != nil {
		return CanonUpdateReport{}, err
	}
	report := CanonUpdateReport{Repository: inspection.WorktreeRoot}
	if inspection.State == ClaimUnmanaged {
		return report, ErrClaimNotManaged
	}
	if inspection.State != ClaimManaged {
		return report, fmt.Errorf("%w (%s): %s", ErrClaimInvalid, inspection.Reason, inspection.Detail)
	}
	report.ProjectID = inspection.Declaration.ProjectID
	report.ContractFrom = inspection.Declaration.ContractVersion
	report.ContractTo = inspection.Declaration.ContractVersion
	canon, err := CurrentProjectCanon()
	if err != nil {
		return report, err
	}
	report.Canon = canon.ID
	report.Notes = []string{
		"project update preserves project identity, policy, setup selections, owner bytes, and evidence",
		"project update does not update the gr binary or activate shared admission",
	}

	rendered, err := RenderProjectCanon(inspection.Declaration.ProjectID)
	if err != nil {
		return report, err
	}
	artifacts, err := InspectGoverningArtifacts(inspection)
	if err != nil {
		return report, err
	}
	if artifacts.Policy.State != ArtifactCurrent {
		return report, fmt.Errorf("%w: policy %s is %s", ErrProjectSemanticChange, artifacts.Policy.Path, artifacts.Policy.State)
	}
	if artifacts.SetupProfile.State != ArtifactCurrent {
		return report, fmt.Errorf("%w: setup profile %s is %s", ErrProjectSemanticChange, artifacts.SetupProfile.Path, artifacts.SetupProfile.State)
	}
	if err := verifyProfilePinsCurrentCanon(artifacts.SetupValue, rendered); err != nil {
		return report, fmt.Errorf("%w: %v", ErrProjectSemanticChange, err)
	}
	bootstrap := renderedByPath(rendered, BootstrapPath)
	if bootstrap == nil || inspection.Declaration.Bootstrap.Digest != bootstrap.Digest {
		return report, fmt.Errorf("%w: declaration bootstrap identity requires an explicit contract migration", ErrProjectSemanticChange)
	}

	changed := false
	for attempt := 1; attempt <= maxProjectUpdateAttempts; attempt++ {
		plans, planErr := planProjectCanonUpdate(
			inspection.WorktreeRoot,
			rendered,
			input.DiscardLocalEdits,
			hooks.retainedFiles,
			hooks.retainedBlocks,
		)
		mergePlansIntoReport(&report, plans)
		if planErr != nil {
			return report, planErr
		}
		if !plansHaveChanges(plans) {
			report.AlreadyCurrent = !changed
			report.Verified = true
			return report, nil
		}

		if hooks.beforeBackup != nil {
			hooks.beforeBackup()
			hooks.beforeBackup = nil
		}
		backup, backupErr := backupProjectPlans(
			inspection.WorktreeRoot,
			input.StateRoot,
			plans,
			now,
			report.Backup,
			canon.ID,
		)
		if backup != "" {
			report.Backup = backup
		}
		if hooks.afterBackup != nil {
			hooks.afterBackup()
			hooks.afterBackup = nil
		}
		if errors.Is(backupErr, errProjectChangedOnUpdate) && input.DiscardLocalEdits && attempt < maxProjectUpdateAttempts {
			continue
		}
		if backupErr != nil {
			return report, backupErr
		}

		writeErr := applyProjectCanonPlans(inspection, plans, hooks.beforeReplace, &report)
		if errors.Is(writeErr, errProjectChangedOnUpdate) && input.DiscardLocalEdits && attempt < maxProjectUpdateAttempts {
			changed = changed || reportHasProjectChanges(report)
			continue
		}
		if writeErr != nil {
			return report, writeErr
		}
		changed = true
	}

	after, err := planProjectCanonUpdate(
		inspection.WorktreeRoot,
		rendered,
		false,
		hooks.retainedFiles,
		hooks.retainedBlocks,
	)
	if err != nil || plansHaveChanges(after) {
		if err != nil {
			return report, err
		}
		return report, errors.New("project canon does not match the installed binary after update")
	}
	report.AlreadyCurrent = !changed
	report.Verified = true
	return report, nil
}

func planProjectCanonUpdate(
	root string,
	rendered []RenderedFile,
	discard bool,
	retainedFiles []RenderedFile,
	retainedBlocks [][]byte,
) ([]canonUpdatePlan, error) {
	var plans []canonUpdatePlan
	for _, file := range rendered {
		if file.Ownership != OwnershipCanon {
			continue
		}
		current, exists, err := readProjectUpdatePath(root, file.Path)
		if err != nil {
			return plans, err
		}
		plan := canonUpdatePlan{
			path: file.Path, ownership: OwnershipCanon, desired: file.Content,
			current: current, exists: exists,
		}
		switch {
		case !exists:
			plan.action = CanonUpdateCreated
		case bytes.Equal(current, file.Content):
			plan.action = CanonUpdateUnchanged
		case matchesRetainedFile(file.Path, current, retainedFiles):
			plan.action = CanonUpdateUpdated
		case discard:
			plan.action, plan.discarded = CanonUpdateUpdated, true
		default:
			plan.action = CanonUpdateRefused
			plan.reason = "canon-owned file differs from every retained canon"
		}
		plans = append(plans, plan)
	}

	desired := contentFor(rendered, AgentsSnippetPath)
	for _, path := range []string{AgentsRootPath, ClaudeRootPath} {
		current, exists, err := readProjectUpdatePath(root, path)
		if err != nil {
			return plans, err
		}
		blockPlan := PlanManagedBlock(path, current, exists, desired, retainedBlocks...)
		if discard {
			blockPlan = PlanManagedBlockDiscard(path, current, exists, desired, retainedBlocks...)
		}
		plan := canonUpdatePlan{
			path: path, ownership: OwnershipManaged, current: current, exists: exists,
			desired: blockPlan.Content, isBlock: true, reason: blockPlan.Reason,
		}
		switch blockPlan.Action {
		case ManagedBlockCreated:
			plan.action = CanonUpdateCreated
		case ManagedBlockUpdated:
			plan.action = CanonUpdateUpdated
			plan.discarded = discard && !managedBlockMatchesKnown(current, desired, retainedBlocks)
		case ManagedBlockUnchanged:
			plan.action = CanonUpdateUnchanged
		case ManagedBlockRefused:
			plan.action = CanonUpdateRefused
		}
		plans = append(plans, plan)
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].path < plans[j].path })
	for _, plan := range plans {
		if plan.action == CanonUpdateRefused {
			return plans, fmt.Errorf("%w: %s: %s", ErrProjectLocalEdits, plan.path, plan.reason)
		}
	}
	return plans, nil
}

func verifyProfilePinsCurrentCanon(profile domain.SetupProfile, rendered []RenderedFile) error {
	planning := renderedByPath(rendered, PlanningAdapterDescriptorPath)
	prepared := renderedByPath(rendered, PreparedAdmissionPath)
	block := renderedByPath(rendered, AgentsSnippetPath)
	if planning == nil || prepared == nil || block == nil {
		return errors.New("embedded adapter canon is incomplete")
	}
	if profile.Planning.Adapter.Integrity != planning.Digest {
		return errors.New("planning adapter selection differs from the installed canon")
	}
	for _, adapter := range profile.ScaffoldAdapters {
		if adapter.ID == admissionlocal.PrekAdapterID {
			if adapter.Version != admissionlocal.PrekAdapterVersion ||
				adapter.SourceRef != admissionlocal.PrekAdapterSourceRef ||
				adapter.Integrity != admissionlocal.PrekTemplateDigest() {
				return errors.New("optional prek adapter selection differs from the installed canon")
			}
			continue
		}
		if adapter.Integrity != block.Digest {
			return fmt.Errorf("scaffold adapter %s selection differs from the installed canon", adapter.ID)
		}
	}
	if profile.SharedAdmissionAdapter == nil || profile.SharedAdmissionAdapter.Integrity != prepared.Digest {
		return errors.New("shared-admission adapter selection differs from the installed canon")
	}
	return nil
}

func renderedByPath(files []RenderedFile, path string) *RenderedFile {
	for index := range files {
		if files[index].Path == path {
			return &files[index]
		}
	}
	return nil
}

func matchesRetainedFile(path string, current []byte, retained []RenderedFile) bool {
	for _, file := range retained {
		if file.Path == path && bytes.Equal(file.Content, current) {
			return true
		}
	}
	return false
}

func managedBlockMatchesKnown(current, desired []byte, retained [][]byte) bool {
	if PlanManagedBlock("", current, true, desired, retained...).Action == ManagedBlockUpdated {
		return true
	}
	return false
}

func readProjectUpdatePath(root, relative string) ([]byte, bool, error) {
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	if err := ambient.EnsureWriteWithinRepository(root, absolute); err != nil {
		return nil, false, err
	}
	info, err := os.Lstat(absolute)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("inspect %s: %w", relative, err)
	}
	if err := ensureSafeRegularPath(root, absolute, info); err != nil {
		return nil, false, fmt.Errorf("inspect %s: %w", relative, err)
	}
	raw, opened, err := boundedio.ReadRegularFileWithInfo(absolute, relative, maxProjectUpdateFileBytes)
	if err != nil {
		return nil, false, err
	}
	after, err := os.Lstat(absolute)
	if err != nil || !sameFileSnapshot(info, opened) || !sameFileSnapshot(opened, after) {
		return nil, false, fmt.Errorf("%w: %s changed while it was read", errProjectChangedOnUpdate, relative)
	}
	return raw, true, nil
}

func plansHaveChanges(plans []canonUpdatePlan) bool {
	for _, plan := range plans {
		if plan.action == CanonUpdateCreated || plan.action == CanonUpdateUpdated {
			return true
		}
	}
	return false
}

func applyProjectCanonPlans(
	inspection Inspection,
	plans []canonUpdatePlan,
	beforeReplace func(string),
	report *CanonUpdateReport,
) error {
	for _, plan := range plans {
		if plan.action != CanonUpdateCreated && plan.action != CanonUpdateUpdated {
			continue
		}
		if beforeReplace != nil {
			beforeReplace(plan.path)
		}
		current, exists, err := readProjectUpdatePath(inspection.WorktreeRoot, plan.path)
		if err != nil {
			return err
		}
		if exists != plan.exists || (exists && !bytes.Equal(current, plan.current)) {
			return fmt.Errorf("%w: %s differs from the inspected bytes", errProjectChangedOnUpdate, plan.path)
		}
		if err := inspection.Revalidate(); err != nil {
			return err
		}
		if !plan.exists {
			err = writeNewRegularFile(inspection.WorktreeRoot, plan.path, plan.desired)
		} else {
			err = replaceRegularFile(inspection.WorktreeRoot, plan.path, plan.desired)
		}
		if err != nil {
			return err
		}
		if plan.discarded {
			report.DiscardedLocalEdits = appendUniqueString(report.DiscardedLocalEdits, plan.path)
		}
		mergeOutcome(report, outcomeForPlan(plan))
	}
	return nil
}

type projectBackupManifest struct {
	Schema string                 `json:"schema"`
	Canon  domain.SHA256Digest    `json:"replaced_toward_canon"`
	Files  []projectBackupFinding `json:"files"`
}

type projectBackupFinding struct {
	Path      string              `json:"path"`
	Ownership Ownership           `json:"ownership"`
	Digest    domain.SHA256Digest `json:"digest"`
}

func backupProjectPlans(
	root, stateRoot string,
	plans []canonUpdatePlan,
	now func() time.Time,
	existing string,
	canon domain.SHA256Digest,
) (string, error) {
	var replaced []canonUpdatePlan
	for _, plan := range plans {
		if plan.action == CanonUpdateUpdated && plan.exists {
			replaced = append(replaced, plan)
		}
	}
	if len(replaced) == 0 {
		return existing, nil
	}
	if strings.TrimSpace(stateRoot) == "" {
		return existing, errors.New("project update needs a state root to retain replaced bytes")
	}
	directory := existing
	manifest := projectBackupManifest{Schema: "goalrail.project-update-backup/v1", Canon: canon}
	if directory == "" {
		base := filepath.Join(stateRoot, filepath.FromSlash(harness.BackupDirectory), now().UTC().Format("20060102T150405Z"))
		if err := os.MkdirAll(filepath.Dir(base), 0o700); err != nil {
			return "", fmt.Errorf("create project backup parent: %w", err)
		}
		directory = base
		for suffix := 2; ; suffix++ {
			if err := os.Mkdir(directory, 0o700); err == nil {
				break
			} else if !errors.Is(err, fs.ErrExist) {
				return "", fmt.Errorf("create project backup: %w", err)
			}
			directory = fmt.Sprintf("%s-%d", base, suffix)
		}
	} else {
		raw, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
		if err != nil {
			return directory, fmt.Errorf("read cumulative project backup manifest: %w", err)
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return directory, fmt.Errorf("decode cumulative project backup manifest: %w", err)
		}
	}

	var changed []string
	for _, plan := range replaced {
		actual, exists, err := readProjectUpdatePath(root, plan.path)
		if err != nil {
			return directory, err
		}
		if !exists || !bytes.Equal(actual, plan.current) {
			changed = append(changed, plan.path)
		}
		if !exists {
			continue
		}
		destination := filepath.Join(directory, filepath.FromSlash(plan.path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return directory, fmt.Errorf("create backup parent for %s: %w", plan.path, err)
		}
		if err := os.WriteFile(destination, actual, 0o600); err != nil {
			return directory, fmt.Errorf("back up %s: %w", plan.path, err)
		}
		manifest.Files = mergeProjectBackupFinding(manifest.Files, projectBackupFinding{
			Path: plan.path, Ownership: plan.ownership, Digest: domain.DigestCanonicalJSON(actual),
		})
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return directory, err
	}
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), append(encoded, '\n'), 0o600); err != nil {
		return directory, fmt.Errorf("write project backup manifest: %w", err)
	}
	if len(changed) > 0 {
		return directory, fmt.Errorf("%w: %s", errProjectChangedOnUpdate, strings.Join(changed, ", "))
	}
	return directory, nil
}

func mergeProjectBackupFinding(existing []projectBackupFinding, finding projectBackupFinding) []projectBackupFinding {
	for index := range existing {
		if existing[index].Path == finding.Path {
			existing[index] = finding
			return existing
		}
	}
	return append(existing, finding)
}

func mergePlansIntoReport(report *CanonUpdateReport, plans []canonUpdatePlan) {
	for _, plan := range plans {
		mergeOutcome(report, outcomeForPlan(plan))
	}
}

func outcomeForPlan(plan canonUpdatePlan) CanonUpdateOutcome {
	outcome := CanonUpdateOutcome{
		Path: plan.path, Action: plan.action, Ownership: plan.ownership,
		Reason: plan.reason,
	}
	if len(plan.desired) > 0 {
		outcome.Digest = domain.DigestCanonicalJSON(plan.desired)
	}
	if plan.exists {
		outcome.BeforeDigest = domain.DigestCanonicalJSON(plan.current)
	}
	return outcome
}

func mergeOutcome(report *CanonUpdateReport, outcome CanonUpdateOutcome) {
	target := &report.Files
	if outcome.Ownership == OwnershipManaged {
		target = &report.ManagedBlocks
	}
	for index := range *target {
		if (*target)[index].Path != outcome.Path {
			continue
		}
		if outcome.Action == CanonUpdateUpdated || outcome.Action == CanonUpdateCreated || (*target)[index].Action == CanonUpdateUnchanged {
			(*target)[index] = outcome
		}
		return
	}
	*target = append(*target, outcome)
	sort.Slice(*target, func(i, j int) bool { return (*target)[i].Path < (*target)[j].Path })
}

func reportHasProjectChanges(report CanonUpdateReport) bool {
	for _, outcomes := range [][]CanonUpdateOutcome{report.Files, report.ManagedBlocks} {
		for _, outcome := range outcomes {
			if outcome.Action == CanonUpdateCreated || outcome.Action == CanonUpdateUpdated {
				return true
			}
		}
	}
	return false
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
