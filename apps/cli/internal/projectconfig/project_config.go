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
	"github.com/heurema/goalrail/apps/cli/internal/gitctx"
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

	if !matchesKnownIdentity(existing, expected) {
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
		existingConfig, parseErr := ParseYAML(string(existing))
		if parseErr == nil && matchesKnownIdentity(existingConfig, config) {
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
	if err := writeFileAtomic(path, content, 0o644); err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("write local .goalrail/project.yml: %w", err))
	}
	return StatusWritten, nil
}

func matchesKnownIdentity(existing Config, expected Config) bool {
	if expected.ServerURL != "" && existing.ServerURL != expected.ServerURL {
		return false
	}
	if expected.OrganizationID != "" && existing.OrganizationID != expected.OrganizationID {
		return false
	}
	if expected.ProjectID != "" && existing.ProjectID != expected.ProjectID {
		return false
	}
	if expected.RepoBindingID != "" && existing.RepoBindingID != expected.RepoBindingID {
		return false
	}
	if expected.Repository.Provider != "" && existing.Repository.Provider != expected.Repository.Provider {
		return false
	}
	if expected.Repository.FullName != "" && existing.Repository.FullName != expected.Repository.FullName {
		return false
	}
	if expected.Repository.URL != "" && !repositoryURLIdentityMatches(existing.Repository.URL, expected.Repository.URL) {
		return false
	}
	if expected.Repository.WorkflowBaseBranch != "" && existing.Repository.WorkflowBaseBranch != expected.Repository.WorkflowBaseBranch {
		return false
	}
	return true
}

func repositoryURLIdentityMatches(existing string, expected string) bool {
	if existing == "" {
		return false
	}
	existingRemote := gitctx.ParseRemoteURL(existing)
	expectedRemote := gitctx.ParseRemoteURL(expected)
	if existingRemote.ProviderHost == "" || expectedRemote.ProviderHost == "" ||
		existingRemote.RepositoryFullName == "" || expectedRemote.RepositoryFullName == "" {
		return existing == expected
	}
	return existingRemote.ProviderHost == expectedRemote.ProviderHost &&
		existingRemote.RepositoryFullName == expectedRemote.RepositoryFullName
}

func writeFileAtomic(path string, content []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() {
		_ = os.Remove(tempName)
	}()

	if err := temp.Chmod(perm); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	tempName = ""

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		_ = dirFile.Close()
	}()
	return dirFile.Sync()
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
	parser := markerParser{}
	if err := parser.parse(raw); err != nil {
		return Config{}, err
	}
	if !parser.seenVersion {
		return Config{}, errors.New("missing version")
	}
	version := parser.config.Version
	if version != Version {
		return Config{}, fmt.Errorf("unsupported version %d", version)
	}
	for _, field := range []struct {
		name string
		seen bool
	}{
		{name: "server_url", seen: parser.seenServerURL},
		{name: "organization_id", seen: parser.seenOrganizationID},
		{name: "project_id", seen: parser.seenProjectID},
		{name: "repo_binding_id", seen: parser.seenRepoBindingID},
		{name: "repository.provider", seen: parser.seenProvider},
		{name: "repository.full_name", seen: parser.seenFullName},
		{name: "repository.url", seen: parser.seenRepositoryURL},
		{name: "repository.workflow_base_branch", seen: parser.seenWorkflowBaseBranch},
	} {
		if !field.seen {
			return Config{}, fmt.Errorf("missing %s", field.name)
		}
	}
	return parser.config, nil
}

type markerParser struct {
	config                 Config
	section                string
	seenVersion            bool
	seenServerURL          bool
	seenOrganizationID     bool
	seenProjectID          bool
	seenRepoBindingID      bool
	seenProvider           bool
	seenFullName           bool
	seenRepositoryURL      bool
	seenWorkflowBaseBranch bool
}

func (p *markerParser) parse(raw string) error {
	for lineNumber, rawLine := range strings.Split(raw, "\n") {
		line, err := stripYAMLComment(rawLine)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNumber+1, err)
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		if indent > 0 {
			if p.section != "repository" {
				continue
			}
			if err := p.parseRepositoryField(trimmed); err != nil {
				return fmt.Errorf("line %d: %w", lineNumber+1, err)
			}
			continue
		}

		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			return fmt.Errorf("line %d: expected key/value", lineNumber+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "repository" {
			if value != "" {
				return fmt.Errorf("line %d: repository must be a block", lineNumber+1)
			}
			p.section = "repository"
			continue
		}
		p.section = ""
		if value == "" {
			p.section = "__unknown_block__"
			continue
		}
		if err := p.parseTopLevelField(key, value); err != nil {
			return fmt.Errorf("line %d: %w", lineNumber+1, err)
		}
	}
	return nil
}

func (p *markerParser) parseTopLevelField(key string, value string) error {
	switch key {
	case "version":
		version, err := parseYAMLInt(value)
		if err != nil {
			return err
		}
		if p.seenVersion {
			return errors.New("duplicate version")
		}
		p.config.Version = version
		p.seenVersion = true
	case "server_url":
		return p.setString(value, &p.seenServerURL, &p.config.ServerURL, key)
	case "organization_id":
		return p.setString(value, &p.seenOrganizationID, &p.config.OrganizationID, key)
	case "project_id":
		return p.setString(value, &p.seenProjectID, &p.config.ProjectID, key)
	case "repo_binding_id":
		return p.setString(value, &p.seenRepoBindingID, &p.config.RepoBindingID, key)
	}
	return nil
}

func (p *markerParser) parseRepositoryField(line string) error {
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return errors.New("expected repository key/value")
	}
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch key {
	case "provider":
		return p.setString(value, &p.seenProvider, &p.config.Repository.Provider, "repository.provider")
	case "full_name":
		return p.setString(value, &p.seenFullName, &p.config.Repository.FullName, "repository.full_name")
	case "url":
		return p.setString(value, &p.seenRepositoryURL, &p.config.Repository.URL, "repository.url")
	case "workflow_base_branch":
		return p.setString(value, &p.seenWorkflowBaseBranch, &p.config.Repository.WorkflowBaseBranch, "repository.workflow_base_branch")
	}
	return nil
}

func (p *markerParser) setString(value string, seen *bool, target *string, name string) error {
	if *seen {
		return fmt.Errorf("duplicate %s", name)
	}
	parsed, err := parseYAMLString(value)
	if err != nil {
		return err
	}
	*target = parsed
	*seen = true
	return nil
}

func stripYAMLComment(line string) (string, error) {
	inQuote := false
	escaped := false
	for index, r := range line {
		if escaped {
			escaped = false
			continue
		}
		switch r {
		case '\\':
			if inQuote {
				escaped = true
			}
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return strings.TrimRight(line[:index], " \t"), nil
			}
		}
	}
	if inQuote {
		return "", errors.New("unterminated quoted string")
	}
	return line, nil
}

func parseYAMLInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, `"`) {
		parsed, err := strconv.Unquote(value)
		if err != nil {
			return 0, err
		}
		value = parsed
	}
	return strconv.Atoi(strings.TrimSpace(value))
}

func parseYAMLString(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, `"`) {
		return value, nil
	}
	return strconv.Unquote(value)
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
