package projectconfig

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

const (
	Version            = 1
	RelativePath       = ".goalrail/project.yml"
	IgnoreRelativePath = ".goalrail/.gitignore"
	ConflictMessage    = "local .goalrail/project.yml is bound to a different GoalRail project or repository; remove it or use a future repair command"
	UnparseableMessage = "local .goalrail/project.yml could not be parsed as a GoalRail project marker; remove it or use a future repair command"
	StatusWritten      = "written"
	StatusUnchanged    = "unchanged"
	StatusUpdated      = "updated"
)

var localStateIgnoreRules = []string{
	"/local/",
	"/cache/",
	"/state/",
	"/tmp/",
	"*.local.yml",
	"*.local.toml",
	"*.local.json",
}

type Config struct {
	Version        int
	ServerURL      string
	OrganizationID string
	ProjectID      string
	RepoBindingID  string
	Repository     Repository
}

type Repository struct {
	Provider           string
	FullName           string
	URL                string
	WorkflowBaseBranch string
}

func Preflight(gitRoot string, expected Config) error {
	existing, ok, err := Read(gitRoot)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if existing.ServerURL != expected.ServerURL ||
		(expected.ProjectID != "" && existing.ProjectID != expected.ProjectID) ||
		existing.Repository.Provider != expected.Repository.Provider ||
		existing.Repository.FullName != expected.Repository.FullName ||
		existing.Repository.URL != expected.Repository.URL ||
		existing.Repository.WorkflowBaseBranch != expected.Repository.WorkflowBaseBranch {
		return exitcode.ValidationError(errors.New(ConflictMessage))
	}
	return nil
}

func Read(gitRoot string) (Config, bool, error) {
	if strings.TrimSpace(gitRoot) == "" {
		return Config{}, false, exitcode.UsageError(errors.New("server-backed init requires a Git root to read .goalrail/project.yml"))
	}

	path := filepath.Join(gitRoot, RelativePath)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, exitcode.RuntimeError(fmt.Errorf("read local .goalrail/project.yml: %w", err))
	}

	config, err := ParseYAML(string(raw))
	if err != nil {
		return Config{}, false, exitcode.ValidationError(errors.New(UnparseableMessage))
	}
	return config, true, nil
}

func Write(gitRoot string, config Config) (string, error) {
	if strings.TrimSpace(gitRoot) == "" {
		return "", exitcode.UsageError(errors.New("server-backed init requires a Git root to write .goalrail/project.yml"))
	}

	path := filepath.Join(gitRoot, RelativePath)
	content := []byte(RenderYAML(config))
	existing, err := os.ReadFile(path)
	if err == nil {
		if bytes.Equal(existing, content) {
			return StatusUnchanged, nil
		}
		return "", exitcode.ValidationError(errors.New("local .goalrail/project.yml already exists with different content; remove it or re-run with a future repair command"))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", exitcode.RuntimeError(fmt.Errorf("read local .goalrail/project.yml: %w", err))
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("create .goalrail directory: %w", err))
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("write local .goalrail/project.yml: %w", err))
	}
	return StatusWritten, nil
}

func EnsureLocalStateGitignore(gitRoot string) (string, error) {
	if strings.TrimSpace(gitRoot) == "" {
		return "", exitcode.UsageError(errors.New("server-backed init requires a Git root to write .goalrail/.gitignore"))
	}

	path := filepath.Join(gitRoot, IgnoreRelativePath)
	desired := RenderLocalStateGitignore()
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", exitcode.RuntimeError(fmt.Errorf("create .goalrail directory: %w", err))
		}
		if err := os.WriteFile(path, []byte(desired), 0o644); err != nil {
			return "", exitcode.RuntimeError(fmt.Errorf("write local .goalrail/.gitignore: %w", err))
		}
		return StatusWritten, nil
	}
	if err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("read local .goalrail/.gitignore: %w", err))
	}

	missing := missingLocalStateIgnoreRules(string(existing))
	if len(missing) == 0 {
		return StatusUnchanged, nil
	}
	updated := appendMissingIgnoreRules(string(existing), missing)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("write local .goalrail/.gitignore: %w", err))
	}
	return StatusUpdated, nil
}

func RenderLocalStateGitignore() string {
	return strings.Join(localStateIgnoreRules, "\n") + "\n"
}

func RenderYAML(config Config) string {
	version := config.Version
	if version == 0 {
		version = Version
	}

	var b strings.Builder
	fmt.Fprintf(&b, "version: %d\n", version)
	fmt.Fprintf(&b, "server_url: %s\n", QuoteYAMLString(config.ServerURL))
	fmt.Fprintf(&b, "organization_id: %s\n", QuoteYAMLString(config.OrganizationID))
	fmt.Fprintf(&b, "project_id: %s\n", QuoteYAMLString(config.ProjectID))
	fmt.Fprintf(&b, "repo_binding_id: %s\n\n", QuoteYAMLString(config.RepoBindingID))
	b.WriteString("repository:\n")
	fmt.Fprintf(&b, "  provider: %s\n", QuoteYAMLString(config.Repository.Provider))
	fmt.Fprintf(&b, "  full_name: %s\n", QuoteYAMLString(config.Repository.FullName))
	fmt.Fprintf(&b, "  url: %s\n", QuoteYAMLString(config.Repository.URL))
	fmt.Fprintf(&b, "  workflow_base_branch: %s\n", QuoteYAMLString(config.Repository.WorkflowBaseBranch))
	return b.String()
}

func missingLocalStateIgnoreRules(raw string) []string {
	present := map[string]struct{}{}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		present[trimmed] = struct{}{}
	}
	missing := []string{}
	for _, rule := range localStateIgnoreRules {
		if _, ok := present[rule]; !ok {
			missing = append(missing, rule)
		}
	}
	return missing
}

func appendMissingIgnoreRules(raw string, missing []string) string {
	var b strings.Builder
	b.WriteString(strings.TrimRight(raw, "\n"))
	if b.Len() > 0 {
		b.WriteString("\n")
	}
	for _, rule := range missing {
		b.WriteString(rule)
		b.WriteString("\n")
	}
	return b.String()
}

func ParseYAML(raw string) (Config, error) {
	raw = strings.TrimSuffix(raw, "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) != 11 {
		return Config{}, errors.New("unexpected line count")
	}
	if lines[5] != "" || lines[6] != "repository:" {
		return Config{}, errors.New("unexpected repository block")
	}

	version, err := parseVersion(lines[0])
	if err != nil {
		return Config{}, err
	}
	if version != Version {
		return Config{}, fmt.Errorf("unsupported version %d", version)
	}

	serverURL, err := parseString(lines[1], "server_url")
	if err != nil {
		return Config{}, err
	}
	organizationID, err := parseString(lines[2], "organization_id")
	if err != nil {
		return Config{}, err
	}
	projectID, err := parseString(lines[3], "project_id")
	if err != nil {
		return Config{}, err
	}
	repoBindingID, err := parseString(lines[4], "repo_binding_id")
	if err != nil {
		return Config{}, err
	}
	provider, err := parseString(lines[7], "  provider")
	if err != nil {
		return Config{}, err
	}
	fullName, err := parseString(lines[8], "  full_name")
	if err != nil {
		return Config{}, err
	}
	repositoryURL, err := parseString(lines[9], "  url")
	if err != nil {
		return Config{}, err
	}
	workflowBaseBranch, err := parseString(lines[10], "  workflow_base_branch")
	if err != nil {
		return Config{}, err
	}

	return Config{
		Version:        version,
		ServerURL:      serverURL,
		OrganizationID: organizationID,
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		Repository: Repository{
			Provider:           provider,
			FullName:           fullName,
			URL:                repositoryURL,
			WorkflowBaseBranch: workflowBaseBranch,
		},
	}, nil
}

func parseVersion(line string) (int, error) {
	const prefix = "version: "
	if !strings.HasPrefix(line, prefix) {
		return 0, errors.New("missing version")
	}
	return strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
}

func parseString(line string, key string) (string, error) {
	prefix := key + ": "
	if !strings.HasPrefix(line, prefix) {
		return "", fmt.Errorf("missing %s", strings.TrimSpace(key))
	}
	value, err := strconv.Unquote(strings.TrimPrefix(line, prefix))
	if err != nil {
		return "", err
	}
	return value, nil
}

func QuoteYAMLString(value string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range value {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, "\\x%02X", r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
