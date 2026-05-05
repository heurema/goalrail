package initcmd

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
	projectConfigVersion            = 1
	projectConfigRelativePath       = ".goalrail/project.yml"
	projectConfigConflictMessage    = "local .goalrail/project.yml is bound to a different GoalRail project or repository; remove it or use a future repair command"
	projectConfigUnparseableMessage = "local .goalrail/project.yml could not be parsed as a GoalRail project marker; remove it or use a future repair command"
	localConfigStatusWritten        = "written"
	localConfigStatusUnchanged      = "unchanged"
)

type projectConfig struct {
	Version        int
	ServerURL      string
	OrganizationID string
	ProjectID      string
	RepoBindingID  string
	Repository     projectConfigRepository
}

type projectConfigRepository struct {
	Provider           string
	FullName           string
	URL                string
	WorkflowBaseBranch string
}

func preflightProjectConfig(gitRoot string, expected projectConfig) error {
	existing, ok, err := readProjectConfig(gitRoot)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if existing.ServerURL != expected.ServerURL ||
		existing.ProjectID != expected.ProjectID ||
		existing.Repository.Provider != expected.Repository.Provider ||
		existing.Repository.FullName != expected.Repository.FullName ||
		existing.Repository.URL != expected.Repository.URL ||
		existing.Repository.WorkflowBaseBranch != expected.Repository.WorkflowBaseBranch {
		return exitcode.ValidationError(errors.New(projectConfigConflictMessage))
	}
	return nil
}

func readProjectConfig(gitRoot string) (projectConfig, bool, error) {
	if strings.TrimSpace(gitRoot) == "" {
		return projectConfig{}, false, exitcode.UsageError(errors.New("server-backed init requires a Git root to read .goalrail/project.yml"))
	}

	path := filepath.Join(gitRoot, projectConfigRelativePath)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return projectConfig{}, false, nil
	}
	if err != nil {
		return projectConfig{}, false, exitcode.RuntimeError(fmt.Errorf("read local .goalrail/project.yml: %w", err))
	}

	config, err := parseProjectConfigYAML(string(raw))
	if err != nil {
		return projectConfig{}, false, exitcode.ValidationError(errors.New(projectConfigUnparseableMessage))
	}
	return config, true, nil
}

func writeProjectConfig(gitRoot string, config projectConfig) (string, error) {
	if strings.TrimSpace(gitRoot) == "" {
		return "", exitcode.UsageError(errors.New("server-backed init requires a Git root to write .goalrail/project.yml"))
	}

	path := filepath.Join(gitRoot, projectConfigRelativePath)
	content := []byte(renderProjectConfigYAML(config))
	existing, err := os.ReadFile(path)
	if err == nil {
		if bytes.Equal(existing, content) {
			return localConfigStatusUnchanged, nil
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
	return localConfigStatusWritten, nil
}

func renderProjectConfigYAML(config projectConfig) string {
	version := config.Version
	if version == 0 {
		version = projectConfigVersion
	}

	var b strings.Builder
	fmt.Fprintf(&b, "version: %d\n", version)
	fmt.Fprintf(&b, "server_url: %s\n", quoteYAMLString(config.ServerURL))
	fmt.Fprintf(&b, "organization_id: %s\n", quoteYAMLString(config.OrganizationID))
	fmt.Fprintf(&b, "project_id: %s\n", quoteYAMLString(config.ProjectID))
	fmt.Fprintf(&b, "repo_binding_id: %s\n\n", quoteYAMLString(config.RepoBindingID))
	b.WriteString("repository:\n")
	fmt.Fprintf(&b, "  provider: %s\n", quoteYAMLString(config.Repository.Provider))
	fmt.Fprintf(&b, "  full_name: %s\n", quoteYAMLString(config.Repository.FullName))
	fmt.Fprintf(&b, "  url: %s\n", quoteYAMLString(config.Repository.URL))
	fmt.Fprintf(&b, "  workflow_base_branch: %s\n", quoteYAMLString(config.Repository.WorkflowBaseBranch))
	return b.String()
}

func parseProjectConfigYAML(raw string) (projectConfig, error) {
	raw = strings.TrimSuffix(raw, "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) != 11 {
		return projectConfig{}, errors.New("unexpected line count")
	}
	if lines[5] != "" || lines[6] != "repository:" {
		return projectConfig{}, errors.New("unexpected repository block")
	}

	version, err := parseProjectConfigVersion(lines[0])
	if err != nil {
		return projectConfig{}, err
	}
	if version != projectConfigVersion {
		return projectConfig{}, fmt.Errorf("unsupported version %d", version)
	}

	serverURL, err := parseProjectConfigString(lines[1], "server_url")
	if err != nil {
		return projectConfig{}, err
	}
	organizationID, err := parseProjectConfigString(lines[2], "organization_id")
	if err != nil {
		return projectConfig{}, err
	}
	projectID, err := parseProjectConfigString(lines[3], "project_id")
	if err != nil {
		return projectConfig{}, err
	}
	repoBindingID, err := parseProjectConfigString(lines[4], "repo_binding_id")
	if err != nil {
		return projectConfig{}, err
	}
	provider, err := parseProjectConfigString(lines[7], "  provider")
	if err != nil {
		return projectConfig{}, err
	}
	fullName, err := parseProjectConfigString(lines[8], "  full_name")
	if err != nil {
		return projectConfig{}, err
	}
	repositoryURL, err := parseProjectConfigString(lines[9], "  url")
	if err != nil {
		return projectConfig{}, err
	}
	workflowBaseBranch, err := parseProjectConfigString(lines[10], "  workflow_base_branch")
	if err != nil {
		return projectConfig{}, err
	}

	return projectConfig{
		Version:        version,
		ServerURL:      serverURL,
		OrganizationID: organizationID,
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		Repository: projectConfigRepository{
			Provider:           provider,
			FullName:           fullName,
			URL:                repositoryURL,
			WorkflowBaseBranch: workflowBaseBranch,
		},
	}, nil
}

func parseProjectConfigVersion(line string) (int, error) {
	const prefix = "version: "
	if !strings.HasPrefix(line, prefix) {
		return 0, errors.New("missing version")
	}
	return strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
}

func parseProjectConfigString(line string, key string) (string, error) {
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

func quoteYAMLString(value string) string {
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
