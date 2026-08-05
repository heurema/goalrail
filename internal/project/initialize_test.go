package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
)

func TestProjectCanonRendersDeterministicBoundContracts(t *testing.T) {
	projectID := domain.ProjectID("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	first, err := RenderProjectCanon(projectID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderProjectCanon(projectID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same project ID rendered different canon bytes")
	}
	canon, err := CurrentProjectCanon()
	if err != nil {
		t.Fatal(err)
	}
	if canon.LegacyV018OverlayCanon != harness.LegacyV018Canon().ID || !domain.IsSHA256Digest(canon.ID) {
		t.Fatalf("project canon metadata = %#v", canon)
	}

	declaration, err := domain.DecodeProjectDeclaration(bytes.NewReader(contentFor(first, domain.ProjectDeclarationPath)))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := domain.DecodeProjectPolicy(bytes.NewReader(contentFor(first, domain.DefaultProjectPolicyPath)))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := domain.DecodeSetupProfile(bytes.NewReader(contentFor(first, domain.DefaultProjectSetupProfilePath)))
	if err != nil {
		t.Fatal(err)
	}
	if declaration.ProjectID != projectID || policy.ProjectID != projectID || profile.ProjectID != projectID {
		t.Fatal("rendered contracts are not bound to one project ID")
	}
	if profile.Planning.Compiler != "@fission-ai/openspec" || profile.Planning.CompilerVersion != "1.6.0" || profile.Planning.RuntimeVersion != "22.18.0" {
		t.Fatalf("planning adapter is not exactly pinned: %#v", profile.Planning)
	}
	if declaration.Policy.Digest != domain.DigestCanonicalJSON(contentFor(first, domain.DefaultProjectPolicyPath)) ||
		declaration.Bootstrap.Digest != domain.DigestCanonicalJSON(contentFor(first, BootstrapPath)) ||
		declaration.SetupProfile.Digest != domain.DigestCanonicalJSON(contentFor(first, domain.DefaultProjectSetupProfilePath)) {
		t.Fatal("declaration references do not bind the rendered bytes")
	}
}

func TestGeneratedSupportedAgentBootstrapPreservesSetupAuthority(t *testing.T) {
	files, err := RenderProjectCanon(domain.ProjectID("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	if err != nil {
		t.Fatal(err)
	}

	bootstrap := string(contentFor(files, BootstrapPath))
	for _, required := range []string{
		"GOALRAIL_SETUP_REQUIRED",
		"`goalrail.setup-plan/v1`",
		"complete canonical JSON plan and its digest",
		"every `component`, `mutation`,",
		"`prerequisite`, `trust_step`, `network_access`, `rollback`, and",
		"`verification` item",
		"`project_code_writes` must be\n  `0`",
		"ask exactly one authorization question",
		"Do you authorize Goalrail setup plan `<plan_digest>` exactly as shown?",
		"Do not repeat the unchanged question",
		"run the bundled `gr setup\n   verify-plan`",
		"Run bundled `gr setup apply`",
		"Never manufacture scaffold or provider trust",
		"enabling action absent from the displayed plan, stop before that action",
		"canonical `goalrail.setup-receipt/v1` output",
		"Resume the initiating request only after a terminal setup receipt exists",
		"candidate Intent Snapshot",
		"does not confirm intent, authorize implementation",
	} {
		if !strings.Contains(bootstrap, required) {
			t.Fatalf("canonical bootstrap does not contain %q:\n%s", required, bootstrap)
		}
	}
	for _, forbidden := range []string{
		"curl | sh",
		"curl -fsSL",
		"enable branch protection after setup",
		"setup permission confirms intent",
	} {
		if strings.Contains(bootstrap, forbidden) {
			t.Fatalf("canonical bootstrap contains prohibited instruction %q", forbidden)
		}
	}

	agents := contentFor(files, AgentsSnippetPath)
	claude := contentFor(files, ClaudeSnippetPath)
	if !bytes.Equal(agents, claude) {
		t.Fatal("supported agent adapters diverged from one canonical block")
	}
	adapter := string(agents)
	for _, required := range []string{
		"read and follow `.goalrail/bootstrap.md`",
		"GOALRAIL_SETUP_REQUIRED",
		"make zero project-code writes",
		"Do not initialize an already declared clone",
	} {
		if !strings.Contains(adapter, required) {
			t.Fatalf("thin supported-agent adapter does not contain %q:\n%s", required, adapter)
		}
	}
	for _, duplicatedSemanticDetail := range []string{
		"goalrail.setup-plan/v1",
		"Do you authorize Goalrail setup plan",
		"gr setup verify-plan",
		"goalrail.setup-receipt/v1",
	} {
		if strings.Contains(adapter, duplicatedSemanticDetail) {
			t.Fatalf("thin adapter duplicates canonical bootstrap detail %q", duplicatedSemanticDetail)
		}
	}
}

func TestManagedBlockPlanningPreservesOwnerBytesAndRefusesAmbiguity(t *testing.T) {
	desired := RenderManagedBlock([]byte("current body"))
	previous := RenderManagedBlock([]byte("previous body"))
	owner := append([]byte("owner before\n"), previous...)
	owner = append(owner, []byte("owner after\n")...)
	plan := PlanManagedBlock("AGENTS.md", owner, true, desired, previous)
	if plan.Action != ManagedBlockUpdated || !bytes.HasPrefix(plan.Content, []byte("owner before\n")) || !bytes.HasSuffix(plan.Content, []byte("owner after\n")) {
		t.Fatalf("safe known-block update = %#v", plan)
	}
	if plan := PlanManagedBlock("AGENTS.md", []byte("owner only\n"), true, desired); plan.Action != ManagedBlockRefused {
		t.Fatalf("owner-only file action = %s", plan.Action)
	}
	for _, malformed := range [][]byte{
		append(append([]byte(nil), desired...), desired...),
		[]byte("<!-- goalrail:managed-end -->\n"),
		append([]byte(managedBlockStartPrefix), []byte("bad\n")...),
	} {
		if plan := PlanManagedBlock("AGENTS.md", malformed, true, desired); plan.Action != ManagedBlockRefused {
			t.Fatalf("malformed managed block action = %s", plan.Action)
		}
	}
	large := bytes.Repeat([]byte("x"), MaxOwnerInstructionBytes+1)
	if plan := PlanManagedBlock("AGENTS.md", large, true, desired); plan.Action != ManagedBlockRefused {
		t.Fatalf("unbounded file action = %s", plan.Action)
	}
}

func TestFreshInitializeIsPortableIdempotentAndCreatesNoLocalIdentity(t *testing.T) {
	ctx := context.Background()
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
	report, err := Initialize(ctx, repository, InitializeOptions{RequestedScaffold: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Managed || report.LocallyReady || !report.SetupRequired || report.SharedAdmissionActive || !report.SharedAdmissionPrepared {
		t.Fatalf("readiness boundaries collapsed: %#v", report)
	}
	if _, err := os.Lstat(filepath.Join(repository, ".goalrail", "ambient.json")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("fresh initialization created checkout-local identity: %v", err)
	}
	inspection, err := Inspect(ctx, repository)
	if err != nil || inspection.State != ClaimManaged || inspection.Declaration.ProjectID != report.ProjectID {
		t.Fatalf("managed inspection = %#v, %v", inspection, err)
	}
	before := snapshotTree(t, repository)
	repeated, err := Initialize(ctx, filepath.Join(repository, ".goalrail"), InitializeOptions{RequestedScaffold: "codex"})
	if err != nil {
		t.Fatal(err)
	}
	after := snapshotTree(t, repository)
	if !reflect.DeepEqual(before, after) || repeated.ProjectID != report.ProjectID || repeated.CommitRequired {
		t.Fatal("repeated initialization was not byte-idempotent")
	}
	clone := filepath.Join(t.TempDir(), "clone")
	gitRun(t, repository, "add", ".")
	gitRun(t, repository, "commit", "-qm", "initialize project")
	gitRun(t, repository, "clone", "-q", repository, clone)
	cloned, err := Inspect(ctx, clone)
	if err != nil || cloned.State != ClaimManaged || cloned.Declaration.ProjectID != report.ProjectID {
		t.Fatalf("fresh clone lost project identity: %#v, %v", cloned, err)
	}
}

func TestInitializeRefusesForeignSchemaAndUnsafeLinkBeforeWrites(t *testing.T) {
	ctx := context.Background()
	t.Run("foreign schema", func(t *testing.T) {
		repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
		config := filepath.Join(repository, "openspec", "config.yaml")
		writeTestFile(t, config, []byte("schema: owner-flow\nowner: keep\n"))
		before := snapshotTree(t, repository)
		if _, err := Initialize(ctx, repository, InitializeOptions{}); !errors.Is(err, harness.ErrForeignSchema) {
			t.Fatalf("foreign schema error = %v", err)
		}
		if after := snapshotTree(t, repository); !reflect.DeepEqual(before, after) {
			t.Fatal("foreign schema refusal wrote repository content")
		}
	})
	t.Run("unsafe link", func(t *testing.T) {
		repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(repository, ".goalrail")); err != nil {
			t.Fatal(err)
		}
		if _, err := Initialize(ctx, repository, InitializeOptions{}); err == nil {
			t.Fatal("unsafe .goalrail link was followed")
		}
		entries, err := os.ReadDir(outside)
		if err != nil || len(entries) != 0 {
			t.Fatalf("unsafe link wrote outside repository: %v, %v", entries, err)
		}
	})
}

func TestInitializePreservesConflictingOwnerInstructionFiles(t *testing.T) {
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
	owner := []byte("# Owner authority\nDo not replace me.\n")
	writeTestFile(t, filepath.Join(repository, AgentsRootPath), owner)
	report, err := Initialize(context.Background(), repository, InitializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(repository, AgentsRootPath))
	if err != nil || !bytes.Equal(after, owner) {
		t.Fatal("owner AGENTS.md was modified")
	}
	if !hasBlockRefusal(report.ManagedBlocks) {
		t.Fatalf("owner reconciliation gate missing: %#v", report.ManagedBlocks)
	}
	if _, err := os.Stat(filepath.Join(repository, filepath.FromSlash(AgentsSnippetPath))); err != nil {
		t.Fatalf("canonical reconciliation snippet missing: %v", err)
	}
}

func TestMigrateExactV018PreservesEvidenceAndIsRepeatable(t *testing.T) {
	ctx := context.Background()
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
	installLegacyV018(t, repository)
	writeTestFile(t, filepath.Join(repository, ".goalrail", "ambient.json"), []byte(`{"schema":"goalrail.ambient-marker/v0","initialized_at":"2026-08-04T12:00:00Z"}`))
	writeTestFile(t, filepath.Join(repository, "openspec", "changes", "retained", "intent.md"), []byte("retained intent bytes\n"))
	writeTestFile(t, filepath.Join(repository, ".goalrail", "runs", "receipt.json"), []byte("retained receipt bytes\n"))
	legacyBefore := selectedSnapshot(t, repository, []string{
		".goalrail/ambient.json", ".goalrail/runs/receipt.json", "openspec/changes/retained/intent.md",
		"openspec/schemas/goalrail-intent/schema.yaml",
	})
	report, err := Migrate(ctx, repository, InitializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Migration || !report.CommitRequired || report.LocallyReady || !report.SetupRequired {
		t.Fatalf("migration report boundaries = %#v", report)
	}
	if after := selectedSnapshot(t, repository, mapsKeys(legacyBefore)); !reflect.DeepEqual(legacyBefore, after) {
		t.Fatal("migration rewrote retained v0.1.8 evidence")
	}
	beforeRepeat := snapshotTree(t, repository)
	repeated, err := Migrate(ctx, repository, InitializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ProjectID != report.ProjectID || repeated.CommitRequired || !reflect.DeepEqual(beforeRepeat, snapshotTree(t, repository)) {
		t.Fatal("repeated migration was not byte-stable")
	}
}

func installLegacyV018(t *testing.T, root string) {
	t.Helper()
	legacy := harness.LegacyV018Canon()
	for _, file := range legacy.Files {
		relative := strings.TrimPrefix(file.Path, harness.OverlayDirectory+"/")
		raw, err := os.ReadFile(filepath.Join("..", "harness", "testdata", "canon-v1", filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		if harness.Digest(raw) != file.Digest {
			t.Fatalf("legacy fixture digest mismatch for %s", file.Path)
		}
		writeTestFile(t, filepath.Join(root, filepath.FromSlash(file.Path)), raw)
	}
}

func snapshotTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	result := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." || strings.HasPrefix(filepath.ToSlash(relative), ".git/") || relative == ".git" {
			return err
		}
		if entry.Type().IsRegular() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			result[filepath.ToSlash(relative)] = raw
		} else if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			result[filepath.ToSlash(relative)] = []byte("symlink:" + target)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func selectedSnapshot(t *testing.T, root string, paths []string) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte, len(paths))
	for _, path := range paths {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		result[path] = raw
	}
	return result
}

func mapsKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func writeTestFile(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitRun(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func TestInitializeRequiresARealWorktree(t *testing.T) {
	if _, err := Initialize(context.Background(), t.TempDir(), InitializeOptions{}); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("unmanaged directory error = %v", err)
	}
	bare := filepath.Join(t.TempDir(), "bare.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("git", "-C", bare, "init", "--bare", "--quiet")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, output)
	}
	if _, err := Initialize(context.Background(), bare, InitializeOptions{}); !errors.Is(err, ErrNoWorktree) {
		t.Fatalf("bare repository error = %v", err)
	}
}

func TestPreparedAdmissionTemplateDoesNotClaimActivation(t *testing.T) {
	files, err := RenderProjectCanon(domain.ProjectID("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	if err != nil {
		t.Fatal(err)
	}
	raw := contentFor(files, PreparedAdmissionPath)
	if !bytes.Contains(raw, []byte("state: prepared")) || !bytes.Contains(raw, []byte("presence is not evidence")) {
		t.Fatalf("prepared admission semantics missing: %s", raw)
	}
	var declaration map[string]any
	if err := json.Unmarshal(contentFor(files, domain.ProjectDeclarationPath), &declaration); err != nil {
		t.Fatal(err)
	}
}
