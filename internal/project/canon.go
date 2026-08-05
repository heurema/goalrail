package project

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/harness"
)

// projectCanonFS contains the portable project contract shipped by this
// binary. Dynamic project-bound JSON is rendered through the domain encoders;
// these embedded inputs bind every stable adapter and owner-facing byte.
//
//go:embed canon
var projectCanonFS embed.FS

const (
	ProjectCanonRoot              = "canon"
	BootstrapPath                 = domain.DefaultProjectBootstrapPath
	PreparedAdmissionPath         = ".goalrail/admission/github-actions.yml"
	PlanningAdapterDescriptorPath = ".goalrail/adapters/openspec.json"
	AgentsSnippetPath             = ".goalrail/adapters/AGENTS.md"
	ClaudeSnippetPath             = ".goalrail/adapters/CLAUDE.md"
	AgentsRootPath                = "AGENTS.md"
	ClaudeRootPath                = "CLAUDE.md"
	ProjectCanonSchemaV1          = "goalrail.project-canon/v1"
)

type Ownership string

const (
	OwnershipCanon      Ownership = "canon"
	OwnershipRepository Ownership = "repository"
	OwnershipManaged    Ownership = "managed_block"
)

type RenderedFile struct {
	Path      string              `json:"path"`
	Content   []byte              `json:"-"`
	Digest    domain.SHA256Digest `json:"digest"`
	Ownership Ownership           `json:"ownership"`
}

type ProjectCanon struct {
	Schema                 string              `json:"schema"`
	ID                     domain.SHA256Digest `json:"id"`
	LegacyV018OverlayCanon string              `json:"legacy_v0_1_8_overlay_canon"`
	Files                  []RenderedFile      `json:"files"`
}

// CurrentProjectCanon identifies the embedded, project-independent template
// inputs. Its ID does not depend on a generated project ID.
func CurrentProjectCanon() (ProjectCanon, error) {
	var files []RenderedFile
	err := fs.WalkDir(projectCanonFS, ProjectCanonRoot, func(walked string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := projectCanonFS.ReadFile(walked)
		if err != nil {
			return err
		}
		files = append(files, RenderedFile{
			Path:      strings.TrimPrefix(walked, ProjectCanonRoot+"/"),
			Content:   append([]byte(nil), raw...),
			Digest:    domain.DigestCanonicalJSON(raw),
			Ownership: OwnershipCanon,
		})
		return nil
	})
	if err != nil {
		return ProjectCanon{}, fmt.Errorf("read embedded project canon: %w", err)
	}
	if len(files) == 0 {
		return ProjectCanon{}, fmt.Errorf("embedded project canon is empty")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	var identity strings.Builder
	for _, file := range files {
		fmt.Fprintf(&identity, "%s %s\n", file.Path, file.Digest)
	}
	return ProjectCanon{
		Schema:                 ProjectCanonSchemaV1,
		ID:                     domain.DigestCanonicalJSON([]byte(identity.String())),
		LegacyV018OverlayCanon: harness.LegacyV018Canon().ID,
		Files:                  files,
	}, nil
}

// RenderProjectCanon creates the project-bound v1 files. The declaration is
// returned last so callers can make it the final claim-establishing write.
func RenderProjectCanon(projectID domain.ProjectID) ([]RenderedFile, error) {
	bootstrap, err := projectCanonFS.ReadFile(ProjectCanonRoot + "/bootstrap.md")
	if err != nil {
		return nil, err
	}
	agentBody, err := projectCanonFS.ReadFile(ProjectCanonRoot + "/agent-block.md")
	if err != nil {
		return nil, err
	}
	preparedAdmission, err := projectCanonFS.ReadFile(ProjectCanonRoot + "/prepared-admission.yml")
	if err != nil {
		return nil, err
	}
	planningDescriptor, err := projectCanonFS.ReadFile(ProjectCanonRoot + "/planning-adapter.json")
	if err != nil {
		return nil, err
	}

	managedBlock := RenderManagedBlock(agentBody)
	policyTemplate, err := projectCanonFS.ReadFile(ProjectCanonRoot + "/policy.json.tmpl")
	if err != nil {
		return nil, err
	}
	policy, err := domain.DecodeProjectPolicy(bytes.NewReader(replaceTemplate(policyTemplate, map[string]string{
		"{{PROJECT_ID}}": string(projectID),
	})))
	if err != nil {
		return nil, fmt.Errorf("decode embedded default policy template: %w", err)
	}
	policyArtifact, err := domain.FreezeProjectPolicy(policy)
	if err != nil {
		return nil, fmt.Errorf("render default policy: %w", err)
	}
	profileTemplate, err := projectCanonFS.ReadFile(ProjectCanonRoot + "/setup-profile.json.tmpl")
	if err != nil {
		return nil, err
	}
	profile, err := domain.DecodeSetupProfile(bytes.NewReader(replaceTemplate(profileTemplate, map[string]string{
		"{{PROJECT_ID}}":       string(projectID),
		"{{PLANNING_DIGEST}}":  string(domain.DigestCanonicalJSON(planningDescriptor)),
		"{{BLOCK_DIGEST}}":     string(domain.DigestCanonicalJSON(managedBlock)),
		"{{ADMISSION_DIGEST}}": string(domain.DigestCanonicalJSON(preparedAdmission)),
	})))
	if err != nil {
		return nil, fmt.Errorf("decode embedded setup-profile template: %w", err)
	}
	profileArtifact, err := domain.FreezeSetupProfile(profile)
	if err != nil {
		return nil, fmt.Errorf("render setup profile: %w", err)
	}

	files := []RenderedFile{
		rendered(BootstrapPath, bootstrap, OwnershipCanon),
		rendered(PreparedAdmissionPath, preparedAdmission, OwnershipCanon),
		rendered(PlanningAdapterDescriptorPath, planningDescriptor, OwnershipCanon),
		rendered(AgentsSnippetPath, managedBlock, OwnershipCanon),
		rendered(ClaudeSnippetPath, managedBlock, OwnershipCanon),
		rendered(domain.DefaultProjectPolicyPath, policyArtifact.CanonicalJSON(), OwnershipRepository),
		rendered(domain.DefaultProjectSetupProfilePath, profileArtifact.CanonicalJSON(), OwnershipRepository),
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	declarationTemplate, err := projectCanonFS.ReadFile(ProjectCanonRoot + "/project.json.tmpl")
	if err != nil {
		return nil, err
	}
	declaration, err := domain.DecodeProjectDeclaration(bytes.NewReader(replaceTemplate(declarationTemplate, map[string]string{
		"{{PROJECT_ID}}":       string(projectID),
		"{{POLICY_DIGEST}}":    string(policyArtifact.Digest()),
		"{{BOOTSTRAP_DIGEST}}": string(domain.DigestCanonicalJSON(bootstrap)),
		"{{SETUP_DIGEST}}":     string(profileArtifact.Digest()),
	})))
	if err != nil {
		return nil, fmt.Errorf("decode embedded project template: %w", err)
	}
	declarationArtifact, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil {
		return nil, fmt.Errorf("render project declaration: %w", err)
	}
	files = append(files, rendered(domain.ProjectDeclarationPath, declarationArtifact.CanonicalJSON(), OwnershipRepository))
	return files, nil
}

func replaceTemplate(template []byte, replacements map[string]string) []byte {
	keys := make([]string, 0, len(replacements))
	for key := range replacements {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		pairs = append(pairs, key, replacements[key])
	}
	return []byte(strings.NewReplacer(pairs...).Replace(string(template)))
}

func rendered(path string, content []byte, ownership Ownership) RenderedFile {
	return RenderedFile{Path: path, Content: append([]byte(nil), content...), Digest: domain.DigestCanonicalJSON(content), Ownership: ownership}
}
