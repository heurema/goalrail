package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
	lineagestore "github.com/heurema/goalrail/internal/lineage"
	projectstate "github.com/heurema/goalrail/internal/project"
)

func TestLineageBeginAndAttachCommands(t *testing.T) {
	repository := lineageCommandRepository(t)
	var beginOutput bytes.Buffer
	if err := runLineage(context.Background(), []string{
		"begin", "--repo", repository, "--change", "project-lineage-admission-v0", "--actor-ref", "user:test-owner",
	}, &beginOutput, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var begun lineagestore.BeginReceipt
	if err := json.Unmarshal(beginOutput.Bytes(), &begun); err != nil {
		t.Fatal(err)
	}
	if begun.WorkUnitID == "" || len(begun.EventRefs) != 3 {
		t.Fatalf("begin output = %+v", begun)
	}
	target := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "commit", Identity: "git-commit:0123456789abcdef", Version: "1",
		Digest:    domain.DigestCanonicalJSON([]byte("0123456789abcdef")),
		SourceRef: "git:0123456789abcdef", AdapterID: "git",
	}
	source := domain.ContentAddressedEvidenceReference{
		ArtifactKind: "work_unit", Identity: "work-unit:" + string(begun.WorkUnitID), Version: "1",
		Digest: begun.AnchorDigest, SourceRef: "repo:root/" + begun.AnchorRef, AdapterID: "goalrail",
	}
	event := domain.LineageEvent{
		Schema: domain.LineageEventSchemaV1, WorkUnitID: begun.WorkUnitID,
		Relation: domain.LineageCommit, Cardinality: domain.RelationSet,
		Sources: []domain.ContentAddressedEvidenceReference{source}, Targets: []domain.ContentAddressedEvidenceReference{target},
		ActorRef: "user:test-owner", AdapterID: "git", ObservedAt: time.Date(2026, 8, 5, 16, 0, 0, 0, time.UTC),
	}
	artifact, err := domain.FreezeLineageEvent(event)
	if err != nil {
		t.Fatal(err)
	}
	eventPath := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(eventPath, artifact.CanonicalJSON(), 0o600); err != nil {
		t.Fatal(err)
	}
	var attachOutput bytes.Buffer
	if err := runLineage(context.Background(), []string{
		"attach", "--repo", repository, "--event", eventPath,
	}, &attachOutput, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var attached lineagestore.AttachReceipt
	if err := json.Unmarshal(attachOutput.Bytes(), &attached); err != nil {
		t.Fatal(err)
	}
	if !attached.Created || attached.WorkUnitID != begun.WorkUnitID {
		t.Fatalf("attach output = %+v", attached)
	}
	var inspectOutput bytes.Buffer
	if err := runLineage(context.Background(), []string{
		"inspect", "--repo", repository, "--lookup", target.Identity,
	}, &inspectOutput, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	var resolution lineagestore.Resolution
	if err := json.Unmarshal(inspectOutput.Bytes(), &resolution); err != nil {
		t.Fatal(err)
	}
	if resolution.Status != lineagestore.ResolutionFound || len(resolution.WorkUnitIDs) != 1 || resolution.WorkUnitIDs[0] != begun.WorkUnitID {
		t.Fatalf("inspect output = %+v", resolution)
	}
	var help bytes.Buffer
	if err := runLineage(context.Background(), []string{"--help"}, &help, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help.String(), "begin|attach|inspect") {
		t.Fatalf("lineage help = %q", help.String())
	}
}

func lineageCommandRepository(t *testing.T) string {
	t.Helper()
	repository := t.TempDir()
	command := exec.Command("git", "init", "--quiet", repository)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	files, err := projectstate.RenderProjectCanon("prj_ffffffffffffffffffffffffffffffff")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		writeLineageCommandFile(t, filepath.Join(repository, filepath.FromSlash(file.Path)), file.Content)
	}
	_, currentFile, _, _ := runtime.Caller(0)
	source := filepath.Join(
		filepath.Dir(currentFile),
		"..", "..", "openspec", "changes", "archive", "2026-08-05-project-lineage-admission-v0",
	)
	target := filepath.Join(repository, "openspec", "changes", "project-lineage-admission-v0")
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, raw, 0o644)
	}); err != nil {
		t.Fatal(err)
	}
	return repository
}

func writeLineageCommandFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
}
