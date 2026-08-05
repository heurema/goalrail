package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
)

var (
	ErrMigrationRequired = errors.New("Goalrail v0.1.8 project migration is required")
	ErrMigrationEvidence = errors.New("exact Goalrail v0.1.8 migration evidence is required")
)

type ProjectFileAction string

const (
	ProjectFileCreated   ProjectFileAction = "created"
	ProjectFileUnchanged ProjectFileAction = "unchanged"
	ProjectFileKept      ProjectFileAction = "kept"
)

type ProjectFileOutcome struct {
	Path      string              `json:"path"`
	Action    ProjectFileAction   `json:"action"`
	Digest    domain.SHA256Digest `json:"digest"`
	Ownership Ownership           `json:"ownership"`
}

type InitializeOptions struct {
	ConfirmForeignSchema bool
	RequestedScaffold    string
}

type InitializeReport struct {
	Repository              string                `json:"repository"`
	ProjectID               domain.ProjectID      `json:"project_id"`
	DeclarationDigest       domain.SHA256Digest   `json:"declaration_digest"`
	ProjectCanon            domain.SHA256Digest   `json:"project_canon"`
	LegacyOverlayCanon      string                `json:"legacy_overlay_canon"`
	Config                  harness.ConfigOutcome `json:"config"`
	OpenSpecFiles           []harness.FileOutcome `json:"openspec_files"`
	ProjectFiles            []ProjectFileOutcome  `json:"project_files"`
	ManagedBlocks           []ManagedBlockPlan    `json:"managed_blocks"`
	RequestedScaffold       string                `json:"requested_scaffold,omitempty"`
	Managed                 bool                  `json:"managed"`
	LocallyReady            bool                  `json:"locally_ready"`
	SetupRequired           bool                  `json:"setup_required"`
	SetupReason             string                `json:"setup_reason"`
	SharedAdmissionPrepared bool                  `json:"shared_admission_prepared"`
	SharedAdmissionActive   bool                  `json:"shared_admission_active"`
	Invocation              string                `json:"invocation"`
	CommitRequired          bool                  `json:"commit_required"`
	Migration               bool                  `json:"migration"`
	Next                    []string              `json:"next"`
}

type filePlan struct {
	file    RenderedFile
	action  ProjectFileAction
	current []byte
}

// Initialize creates portable project identity and repository-owned contract
// content. It deliberately performs no checkout-local attachment, marker,
// provider-trust, ignore, commit, or external activation action.
func Initialize(ctx context.Context, start string, options InitializeOptions) (InitializeReport, error) {
	root, err := ResolveWorktreeRoot(ctx, start)
	if err != nil {
		return InitializeReport{}, err
	}
	inspection, err := Inspect(ctx, root)
	if err != nil {
		return InitializeReport{}, err
	}
	if inspection.State == ClaimDeclaredInvalid {
		return InitializeReport{}, fmt.Errorf("reserved project declaration is invalid (%s): %s", inspection.Reason, inspection.Detail)
	}
	if inspection.State == ClaimUnmanaged {
		legacy, err := DetectLegacyV018(ctx, root)
		if err != nil {
			return InitializeReport{}, err
		}
		if legacy.Overlay == LegacyOverlayComplete && legacy.Marker != LegacyMarkerInvalid {
			return InitializeReport{}, fmt.Errorf("%w; run gr migrate --repo %s", ErrMigrationRequired, root)
		}
	}

	configPlan, err := harness.PlanConfig(root, options.ConfirmForeignSchema)
	if err != nil {
		return InitializeReport{}, err
	}
	overlay, err := preflightFreshOverlay(root)
	if err != nil {
		return InitializeReport{}, err
	}

	projectID := inspection.Declaration.ProjectID
	if inspection.State == ClaimUnmanaged {
		projectID, err = domain.NewProjectID()
		if err != nil {
			return InitializeReport{}, err
		}
	}
	return applyProject(ctx, root, projectID, inspection.State == ClaimManaged, false, options, &configPlan, overlay)
}

// Migrate performs the one explicit v0.1.8 project migration. The retained
// marker, OpenSpec overlay, config, intent, run, review, and receipt evidence
// are read-only inputs and are never rewritten.
func Migrate(ctx context.Context, start string, options InitializeOptions) (InitializeReport, error) {
	root, err := ResolveWorktreeRoot(ctx, start)
	if err != nil {
		return InitializeReport{}, err
	}
	inspection, err := Inspect(ctx, root)
	if err != nil {
		return InitializeReport{}, err
	}
	if inspection.State == ClaimDeclaredInvalid {
		return InitializeReport{}, fmt.Errorf("reserved project declaration is invalid (%s): %s", inspection.Reason, inspection.Detail)
	}
	if inspection.State == ClaimManaged {
		return applyProject(ctx, root, inspection.Declaration.ProjectID, true, true, options, nil, nil)
	}
	legacy, err := DetectLegacyV018(ctx, root)
	if err != nil {
		return InitializeReport{}, err
	}
	if legacy.Overlay != LegacyOverlayComplete || legacy.Marker == LegacyMarkerInvalid {
		return InitializeReport{}, fmt.Errorf("%w: marker=%s overlay=%s (%s)", ErrMigrationEvidence, legacy.Marker, legacy.Overlay, legacy.Detail)
	}
	projectID, err := domain.NewProjectID()
	if err != nil {
		return InitializeReport{}, err
	}
	return applyProject(ctx, root, projectID, false, true, options, nil, nil)
}

func preflightFreshOverlay(root string) (*harness.OverlayState, error) {
	canon, err := harness.CurrentCanon()
	if err != nil {
		return nil, err
	}
	for _, file := range canon.Files {
		absolute := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := ambient.EnsureWriteWithinRepository(root, absolute); err != nil {
			return nil, err
		}
	}
	state, err := harness.InspectOverlay(root)
	if err != nil {
		return nil, err
	}
	if state.Edited {
		return nil, fmt.Errorf("OpenSpec overlay contains owner edits; resolve them before initialization")
	}
	if state.Behind {
		return nil, fmt.Errorf("OpenSpec overlay is from a retained earlier canon; use the explicit migration/update path")
	}
	return &state, nil
}

func applyProject(
	ctx context.Context,
	root string,
	projectID domain.ProjectID,
	alreadyManaged bool,
	migration bool,
	options InitializeOptions,
	configPlan *harness.ConfigPlan,
	overlay *harness.OverlayState,
) (InitializeReport, error) {
	_ = ctx
	canon, err := CurrentProjectCanon()
	if err != nil {
		return InitializeReport{}, err
	}
	files, err := RenderProjectCanon(projectID)
	if err != nil {
		return InitializeReport{}, err
	}
	projectPlans, err := planProjectFiles(root, files, alreadyManaged)
	if err != nil {
		return InitializeReport{}, err
	}
	blockPlans, err := planRootBlocks(root, files)
	if err != nil {
		return InitializeReport{}, err
	}

	report := InitializeReport{
		Repository: root, ProjectID: projectID, ProjectCanon: canon.ID,
		LegacyOverlayCanon: canon.LegacyV018OverlayCanon,
		RequestedScaffold:  options.RequestedScaffold,
		Managed:            true, LocallyReady: false, SetupRequired: true,
		SetupReason: "GOALRAIL_SETUP_REQUIRED", SharedAdmissionPrepared: true,
		SharedAdmissionActive: false, Invocation: harness.PinnedNewChange,
		Migration: migration,
	}

	if configPlan != nil {
		outcome, err := configPlan.Apply()
		if err != nil {
			return InitializeReport{}, err
		}
		report.Config = outcome
		openSpecOutcomes, err := harness.Materialize(root, false)
		if err != nil {
			return InitializeReport{}, err
		}
		report.OpenSpecFiles = openSpecOutcomes
		_ = overlay
	}

	for _, block := range blockPlans {
		if block.Action == ManagedBlockCreated {
			if err := writeNewRegularFile(root, block.Path, block.Content); err != nil {
				return InitializeReport{}, err
			}
			report.CommitRequired = true
		} else if block.Action == ManagedBlockUpdated {
			if err := replaceRegularFile(root, block.Path, block.Content); err != nil {
				return InitializeReport{}, err
			}
			report.CommitRequired = true
		}
		block.Content = nil
		report.ManagedBlocks = append(report.ManagedBlocks, block)
	}
	for _, plan := range projectPlans {
		if plan.file.Path == domain.ProjectDeclarationPath {
			continue
		}
		outcome, err := applyFilePlan(root, plan)
		if err != nil {
			return InitializeReport{}, err
		}
		if outcome.Action == ProjectFileCreated {
			report.CommitRequired = true
		}
		report.ProjectFiles = append(report.ProjectFiles, outcome)
	}
	for _, plan := range projectPlans {
		if plan.file.Path != domain.ProjectDeclarationPath {
			continue
		}
		outcome, err := applyFilePlan(root, plan)
		if err != nil {
			return InitializeReport{}, err
		}
		if outcome.Action == ProjectFileCreated {
			report.CommitRequired = true
		}
		report.ProjectFiles = append(report.ProjectFiles, outcome)
		report.DeclarationDigest = plan.file.Digest
	}
	sort.Slice(report.ProjectFiles, func(i, j int) bool { return report.ProjectFiles[i].Path < report.ProjectFiles[j].Path })
	if report.CommitRequired {
		report.Next = append(report.Next, "review and commit the reported repository files; Goalrail does not commit them")
	}
	report.Next = append(report.Next, "run gr doctor, then authorize only the exact local setup plan if GOALRAIL_SETUP_REQUIRED is reported")
	if hasBlockRefusal(report.ManagedBlocks) {
		report.Next = append(report.Next, "reconcile the canonical .goalrail/adapters snippets into the refused owner instruction files")
	}
	return report, nil
}

func planProjectFiles(root string, files []RenderedFile, alreadyManaged bool) ([]filePlan, error) {
	plans := make([]filePlan, 0, len(files))
	for _, file := range files {
		absolute := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := ambient.EnsureWriteWithinRepository(root, absolute); err != nil {
			return nil, err
		}
		raw, err := os.ReadFile(absolute)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			plans = append(plans, filePlan{file: file, action: ProjectFileCreated})
		case err != nil:
			return nil, fmt.Errorf("read %s: %w", file.Path, err)
		case bytes.Equal(raw, file.Content):
			plans = append(plans, filePlan{file: file, action: ProjectFileUnchanged, current: raw})
		case alreadyManaged:
			plans = append(plans, filePlan{file: file, action: ProjectFileKept, current: raw})
		default:
			return nil, fmt.Errorf("reserved project path %s already contains different owner content", file.Path)
		}
	}
	return plans, nil
}

func planRootBlocks(root string, files []RenderedFile) ([]ManagedBlockPlan, error) {
	desired := contentFor(files, AgentsSnippetPath)
	paths := []string{AgentsRootPath, ClaudeRootPath}
	plans := make([]ManagedBlockPlan, 0, len(paths))
	for _, relative := range paths {
		absolute := filepath.Join(root, relative)
		if err := ambient.EnsureWriteWithinRepository(root, absolute); err != nil {
			return nil, err
		}
		info, err := os.Lstat(absolute)
		if errors.Is(err, fs.ErrNotExist) {
			plans = append(plans, PlanManagedBlock(relative, nil, false, desired))
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect %s: %w", relative, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular owner instruction file", relative)
		}
		raw, err := boundedio.ReadRegularFile(absolute, "owner instruction file", MaxOwnerInstructionBytes)
		if err != nil {
			if info.Size() > MaxOwnerInstructionBytes {
				plans = append(plans, ManagedBlockPlan{Path: relative, Action: ManagedBlockRefused, Reason: "owner instruction file exceeds the managed-block bound"})
				continue
			}
			return nil, err
		}
		plans = append(plans, PlanManagedBlock(relative, raw, true, desired))
	}
	return plans, nil
}

func contentFor(files []RenderedFile, path string) []byte {
	for _, file := range files {
		if file.Path == path {
			return append([]byte(nil), file.Content...)
		}
	}
	return nil
}

func applyFilePlan(root string, plan filePlan) (ProjectFileOutcome, error) {
	outcome := ProjectFileOutcome{Path: plan.file.Path, Action: plan.action, Digest: plan.file.Digest, Ownership: plan.file.Ownership}
	if plan.action != ProjectFileCreated {
		return outcome, nil
	}
	if err := writeNewRegularFile(root, plan.file.Path, plan.file.Content); err != nil {
		return outcome, err
	}
	return outcome, nil
}

func writeNewRegularFile(root, relative string, content []byte) error {
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	if err := ambient.EnsureWriteWithinRepository(root, absolute); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return fmt.Errorf("create parent for %s: %w", relative, err)
	}
	file, err := os.OpenFile(absolute, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create %s: %w", relative, err)
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return fmt.Errorf("write %s: %w", relative, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("write %s: %w", relative, err)
	}
	return nil
}

func replaceRegularFile(root, relative string, content []byte) error {
	absolute := filepath.Join(root, filepath.FromSlash(relative))
	if err := ambient.EnsureWriteWithinRepository(root, absolute); err != nil {
		return err
	}
	if err := os.WriteFile(absolute, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", relative, err)
	}
	return nil
}

func hasBlockRefusal(plans []ManagedBlockPlan) bool {
	for _, plan := range plans {
		if plan.Action == ManagedBlockRefused {
			return true
		}
	}
	return false
}
