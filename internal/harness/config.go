package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrForeignSchema reports an OpenSpec configuration that names a custom schema
// which is not Goalrail's. Switching it silently would change how every
// subsequent change in that repository is compiled, and the user would discover
// it from the results rather than from the act.
var ErrForeignSchema = errors.New("the OpenSpec configuration names a custom schema")

// stockSchemas are the schema names the stock CLI ships. A configuration naming
// one of these is a default rather than a decision, so switching it is
// adoption rather than a rewrite of someone's workflow.
var stockSchemas = map[string]bool{"spec-driven": true}

// ConfigAction is what ensuring the configuration did.
type ConfigAction string

const (
	ConfigCreated   ConfigAction = "created"
	ConfigSwitched  ConfigAction = "switched"
	ConfigUnchanged ConfigAction = "unchanged"
)

// ConfigOutcome reports the one key this package manages.
type ConfigOutcome struct {
	Path   string       `json:"path"`
	Action ConfigAction `json:"action"`

	// PreviousSchema is what the configuration named before a switch.
	PreviousSchema string `json:"previous_schema,omitempty"`
}

// newConfig is what an absent configuration is created as: the managed key, and a
// statement that the rest of the file belongs to the repository. No placeholder
// prose is written, because a placeholder that nobody edits ends up describing
// every project as "<describe this project>".
const newConfig = `schema: ` + SchemaName + `

# Goalrail manages the schema key above and nothing else in this file.
# Everything you add here — project context, schema rules — is yours to keep.
`

// EnsureConfig creates the OpenSpec configuration or points its schema key at
// the Goalrail schema, leaving every other byte of the file alone.
//
// The edit is deliberately line-level rather than a parse-and-rewrite: the file
// carries the project's own comments and prose, and round-tripping it through a
// serializer would silently reformat content this package does not own.
func EnsureConfig(repositoryRoot string, confirmForeignSwitch bool) (ConfigOutcome, error) {
	absolute := filepath.Join(repositoryRoot, filepath.FromSlash(ConfigPath))
	outcome := ConfigOutcome{Path: ConfigPath}

	raw, err := os.ReadFile(absolute)
	if errors.Is(err, fs.ErrNotExist) {
		if mkdirErr := os.MkdirAll(filepath.Dir(absolute), 0o755); mkdirErr != nil {
			return outcome, fmt.Errorf("create %s: %w", filepath.Dir(ConfigPath), mkdirErr)
		}
		if writeErr := os.WriteFile(absolute, []byte(newConfig), 0o644); writeErr != nil {
			return outcome, fmt.Errorf("write %s: %w", ConfigPath, writeErr)
		}
		outcome.Action = ConfigCreated
		return outcome, nil
	}
	if err != nil {
		return outcome, fmt.Errorf("read %s: %w", ConfigPath, err)
	}

	content := string(raw)
	named, index := schemaAssignment(content)
	outcome.PreviousSchema = named

	if named == SchemaName {
		outcome.Action = ConfigUnchanged
		outcome.PreviousSchema = ""
		return outcome, nil
	}
	if named != "" && !stockSchemas[named] && !confirmForeignSwitch {
		return outcome, fmt.Errorf("%w (%s); confirm the switch explicitly to adopt the Goalrail schema",
			ErrForeignSchema, named)
	}

	updated := content
	if index >= 0 {
		lines := strings.SplitAfter(content, "\n")
		lines[index] = replacedSchemaLine(lines[index])
		updated = strings.Join(lines, "")
	} else {
		// No key at all: prepend it, keeping the rest byte-identical.
		updated = "schema: " + SchemaName + "\n" + content
	}
	if writeErr := os.WriteFile(absolute, []byte(updated), 0o644); writeErr != nil {
		return outcome, fmt.Errorf("write %s: %w", ConfigPath, writeErr)
	}
	outcome.Action = ConfigSwitched
	return outcome, nil
}

// ConfiguredSchema reports the schema a repository's configuration names, and
// whether the configuration exists at all.
func ConfiguredSchema(repositoryRoot string) (string, bool, error) {
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(ConfigPath)))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", ConfigPath, err)
	}
	named, _ := schemaAssignment(string(raw))
	return named, true, nil
}

// schemaAssignment finds the top-level schema key and returns its value together
// with the index of the line it lives on, or -1 when there is none.
//
// Only a line starting at column zero counts: an indented `schema:` belongs to
// some nested structure, and rewriting it would change a different setting.
func schemaAssignment(content string) (string, int) {
	for index, line := range strings.SplitAfter(content, "\n") {
		trimmed := strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(trimmed, "schema:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "schema:"))
		value = strings.Trim(value, `"'`)
		if commented := strings.Index(value, " #"); commented >= 0 {
			value = strings.TrimSpace(value[:commented])
		}
		return value, index
	}
	return "", -1
}

// replacedSchemaLine rewrites one assignment, preserving the line ending so the
// rest of the file is untouched.
func replacedSchemaLine(line string) string {
	ending := ""
	switch {
	case strings.HasSuffix(line, "\r\n"):
		ending = "\r\n"
	case strings.HasSuffix(line, "\n"):
		ending = "\n"
	}
	return "schema: " + SchemaName + ending
}
