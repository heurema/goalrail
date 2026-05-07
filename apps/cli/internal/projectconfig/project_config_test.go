package projectconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

func TestParseYAMLCanonicalMarker(t *testing.T) {
	t.Parallel()

	want := testConfig()
	got, err := ParseYAML(RenderYAML(want))
	if err != nil {
		t.Fatalf("ParseYAML canonical marker error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseYAML canonical marker = %#v, want %#v", got, want)
	}
}

func TestParseYAMLToleratesCommentsBlankLinesAndUnknownFields(t *testing.T) {
	t.Parallel()

	raw := `# Goalrail committed project marker
version: 1

server_url: "https://goalrail.example.test" # server identity
organization_id: "018f0000-0000-7000-8000-000000000002"
project_id: "018f0000-0000-7000-8000-000000000003"
repo_binding_id: "018f0000-0000-7000-8000-000000000004"
unknown_top_level: "ignored"

repository:
  provider: "github"
  full_name: "heurema/goalrail"
  ignored_nested: "ignored"
  url: "git@github.com:heurema/goalrail.git"
  workflow_base_branch: "main"
`

	got, err := ParseYAML(raw)
	if err != nil {
		t.Fatalf("ParseYAML tolerant marker error = %v", err)
	}
	if !reflect.DeepEqual(got, testConfig()) {
		t.Fatalf("ParseYAML tolerant marker = %#v, want %#v", got, testConfig())
	}
}

func TestParseYAMLToleratesSafeReordering(t *testing.T) {
	t.Parallel()

	raw := `repository:
  workflow_base_branch: "main"
  url: "git@github.com:heurema/goalrail.git"
  full_name: "heurema/goalrail"
  provider: "github"

repo_binding_id: "018f0000-0000-7000-8000-000000000004"
project_id: "018f0000-0000-7000-8000-000000000003"
organization_id: "018f0000-0000-7000-8000-000000000002"
server_url: "https://goalrail.example.test"
version: 1
`

	got, err := ParseYAML(raw)
	if err != nil {
		t.Fatalf("ParseYAML reordered marker error = %v", err)
	}
	if !reflect.DeepEqual(got, testConfig()) {
		t.Fatalf("ParseYAML reordered marker = %#v, want %#v", got, testConfig())
	}
}

func TestParseYAMLMissingRequiredFieldFails(t *testing.T) {
	t.Parallel()

	raw := strings.Replace(RenderYAML(testConfig()), `repo_binding_id: "018f0000-0000-7000-8000-000000000004"`+"\n", "", 1)
	if _, err := ParseYAML(raw); err == nil {
		t.Fatal("ParseYAML missing repo_binding_id error = nil, want error")
	}
}

func TestPreflightClassifiesIdentityConflicts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(Config) Config
	}{
		{
			name: "server_url",
			mutate: func(config Config) Config {
				config.ServerURL = "https://other.example.test"
				return config
			},
		},
		{
			name: "organization_id",
			mutate: func(config Config) Config {
				config.OrganizationID = "018f0000-0000-7000-8000-000000000999"
				return config
			},
		},
		{
			name: "project_id",
			mutate: func(config Config) Config {
				config.ProjectID = "018f0000-0000-7000-8000-000000000999"
				return config
			},
		},
		{
			name: "repo_binding_id",
			mutate: func(config Config) Config {
				config.RepoBindingID = "018f0000-0000-7000-8000-000000000999"
				return config
			},
		},
		{
			name: "provider",
			mutate: func(config Config) Config {
				config.Repository.Provider = "gitlab"
				return config
			},
		},
		{
			name: "repository_full_name",
			mutate: func(config Config) Config {
				config.Repository.FullName = "heurema/other"
				return config
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gitRoot := t.TempDir()
			writeRawProjectConfig(t, gitRoot, RenderYAML(tt.mutate(testConfig())))
			err := Preflight(gitRoot, testConfig())
			if err == nil {
				t.Fatal("Preflight error = nil, want conflict")
			}
			if got := exitcode.ForError(err); got != exitcode.Validation {
				t.Fatalf("exit code = %d, want validation", got)
			}
			if !strings.Contains(err.Error(), ConflictMessage) {
				t.Fatalf("Preflight error = %q, want conflict message", err.Error())
			}
		})
	}
}

func TestPreflightAllowsRepositoryURLVariationForSameProviderAndFullName(t *testing.T) {
	t.Parallel()

	gitRoot := t.TempDir()
	existing := testConfig()
	existing.Repository.URL = "https://github.com/heurema/goalrail.git"
	writeRawProjectConfig(t, gitRoot, RenderYAML(existing))

	if err := Preflight(gitRoot, testConfig()); err != nil {
		t.Fatalf("Preflight SSH/HTTPS variation error = %v, want nil", err)
	}
}

func TestPreflightRejectsRepositoryURLDifferentHostForCustomGit(t *testing.T) {
	t.Parallel()

	gitRoot := t.TempDir()
	existing := testConfig()
	existing.Repository.Provider = "custom_git"
	existing.Repository.FullName = "team/repo"
	existing.Repository.URL = "git@code1.example.test:team/repo.git"
	writeRawProjectConfig(t, gitRoot, RenderYAML(existing))

	expected := testConfig()
	expected.Repository.Provider = "custom_git"
	expected.Repository.FullName = "team/repo"
	expected.Repository.URL = "git@code2.example.test:team/repo.git"
	err := Preflight(gitRoot, expected)
	if err == nil {
		t.Fatal("Preflight custom host variation error = nil, want conflict")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
}

func TestWriteProjectConfigAtomicWritesCompleteMarkerAndRemovesTemp(t *testing.T) {
	t.Parallel()

	gitRoot := t.TempDir()
	status, err := Write(gitRoot, testConfig())
	if err != nil {
		t.Fatalf("Write error = %v", err)
	}
	if status != StatusWritten {
		t.Fatalf("Write status = %q, want %q", status, StatusWritten)
	}

	got := readProjectConfig(t, gitRoot)
	if want := RenderYAML(testConfig()); got != want {
		t.Fatalf("project config =\n%s\nwant:\n%s", got, want)
	}
	info, err := os.Stat(filepath.Join(gitRoot, RelativePath))
	if err != nil {
		t.Fatalf("stat project config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("project config mode = %v, want 0644", got)
	}

	entries, err := os.ReadDir(filepath.Join(gitRoot, ".goalrail"))
	if err != nil {
		t.Fatalf("read .goalrail dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".project.yml.tmp-") {
			t.Fatalf("temporary marker file left behind on success: %s", entry.Name())
		}
	}
}

func TestWriteProjectConfigLeavesEquivalentExistingMarkerUnchanged(t *testing.T) {
	t.Parallel()

	gitRoot := t.TempDir()
	existing := `# local comment
version: 1
server_url: "https://goalrail.example.test"
organization_id: "018f0000-0000-7000-8000-000000000002"
project_id: "018f0000-0000-7000-8000-000000000003"
repo_binding_id: "018f0000-0000-7000-8000-000000000004"

repository:
  provider: "github"
  full_name: "heurema/goalrail"
  url: "https://github.com/heurema/goalrail.git"
  workflow_base_branch: "main"
  unknown: "ignored"
`
	writeRawProjectConfig(t, gitRoot, existing)

	status, err := Write(gitRoot, testConfig())
	if err != nil {
		t.Fatalf("Write equivalent existing marker error = %v", err)
	}
	if status != StatusUnchanged {
		t.Fatalf("Write status = %q, want %q", status, StatusUnchanged)
	}
	if got := readProjectConfig(t, gitRoot); got != existing {
		t.Fatalf("project config changed =\n%s\nwant original:\n%s", got, existing)
	}
}

func TestEnsureLocalStateGitignorePreservesTrackedMarkerBehavior(t *testing.T) {
	t.Parallel()

	gitRoot := t.TempDir()
	writeRawFile(t, gitRoot, IgnoreRelativePath, "# existing\n/local/\n")

	status, err := EnsureLocalStateGitignore(gitRoot)
	if err != nil {
		t.Fatalf("EnsureLocalStateGitignore error = %v", err)
	}
	if status != StatusUpdated {
		t.Fatalf("EnsureLocalStateGitignore status = %q, want %q", status, StatusUpdated)
	}
	got := readRawFile(t, gitRoot, IgnoreRelativePath)
	if !strings.Contains(got, "/local/\n") {
		t.Fatalf(".goalrail/.gitignore = %q, want existing /local/ rule preserved", got)
	}
	for _, want := range strings.Split(strings.TrimSpace(RenderLocalStateGitignore()), "\n") {
		if !strings.Contains(got, want+"\n") {
			t.Fatalf(".goalrail/.gitignore = %q, want rule %q", got, want)
		}
	}
}

func testConfig() Config {
	return Config{
		Version:        Version,
		ServerURL:      "https://goalrail.example.test",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Repository: Repository{
			Provider:           "github",
			FullName:           "heurema/goalrail",
			URL:                "git@github.com:heurema/goalrail.git",
			WorkflowBaseBranch: "main",
		},
	}
}

func writeRawProjectConfig(t *testing.T, gitRoot string, content string) {
	t.Helper()

	writeRawFile(t, gitRoot, RelativePath, content)
}

func writeRawFile(t *testing.T, gitRoot string, relativePath string, content string) {
	t.Helper()

	path := filepath.Join(gitRoot, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relativePath, err)
	}
}

func readProjectConfig(t *testing.T, gitRoot string) string {
	t.Helper()

	return readRawFile(t, gitRoot, RelativePath)
}

func readRawFile(t *testing.T, gitRoot string, relativePath string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(gitRoot, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("read %s: %v", relativePath, err)
	}
	return string(raw)
}
