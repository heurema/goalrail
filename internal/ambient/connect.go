package ambient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Scaffold is one supported agent environment Goalrail can attach to.
type Scaffold string

const (
	ScaffoldCodex      Scaffold = "codex"
	ScaffoldClaudeCode Scaffold = "claude-code"
)

// SupportedScaffolds is the fixed set this version can connect to.
func SupportedScaffolds() []Scaffold {
	return []Scaffold{ScaffoldCodex, ScaffoldClaudeCode}
}

// ConnectionPlan is what a connection would change, computed before anything
// is written so the user consents to a concrete act rather than a promise.
type ConnectionPlan struct {
	Scaffold       Scaffold `json:"scaffold"`
	ConfigPath     string   `json:"config_path"`
	AlreadyPresent bool     `json:"already_present"`
	Executable     string   `json:"executable"`
}

// marker lines bracket everything a connection adds, so disconnection can
// remove exactly that and nothing else. Editing a user's own configuration
// demands a removal that is provably complete.
const (
	blockBegin = "# >>> goalrail ambient (managed) >>>"
	blockEnd   = "# <<< goalrail ambient (managed) <<<"
)

// ConfigPath returns the user-level configuration file a scaffold reads.
func ConfigPath(scaffold Scaffold, home string) (string, error) {
	switch scaffold {
	case ScaffoldCodex:
		return filepath.Join(home, ".codex", "config.toml"), nil
	case ScaffoldClaudeCode:
		return filepath.Join(home, ".claude", "settings.json"), nil
	default:
		return "", fmt.Errorf("unsupported scaffold %q", scaffold)
	}
}

// PlanConnection reports what connecting would do, without doing it.
func PlanConnection(scaffold Scaffold, home, executable string) (ConnectionPlan, error) {
	configPath, err := ConfigPath(scaffold, home)
	if err != nil {
		return ConnectionPlan{}, err
	}
	if !filepath.IsAbs(executable) {
		return ConnectionPlan{}, errors.New("connection requires the absolute gr executable path")
	}
	present, err := isConnected(scaffold, configPath)
	if err != nil {
		return ConnectionPlan{}, err
	}
	return ConnectionPlan{
		Scaffold:       scaffold,
		ConfigPath:     configPath,
		AlreadyPresent: present,
		Executable:     executable,
	}, nil
}

// Connect registers the persistent session hooks. It is idempotent: a second
// connection changes nothing.
func Connect(plan ConnectionPlan) (bool, error) {
	if plan.AlreadyPresent {
		return false, nil
	}
	switch plan.Scaffold {
	case ScaffoldCodex:
		return true, connectCodex(plan)
	case ScaffoldClaudeCode:
		return true, connectClaudeCode(plan)
	default:
		return false, fmt.Errorf("unsupported scaffold %q", plan.Scaffold)
	}
}

// Disconnect removes everything a connection added, leaving no residue.
func Disconnect(scaffold Scaffold, home string) (bool, error) {
	configPath, err := ConfigPath(scaffold, home)
	if err != nil {
		return false, err
	}
	switch scaffold {
	case ScaffoldCodex:
		return removeManagedBlock(configPath)
	case ScaffoldClaudeCode:
		return removeClaudeCodeHooks(configPath)
	default:
		return false, fmt.Errorf("unsupported scaffold %q", scaffold)
	}
}

func isConnected(scaffold Scaffold, configPath string) (bool, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	switch scaffold {
	case ScaffoldCodex:
		return strings.Contains(string(raw), blockBegin), nil
	case ScaffoldClaudeCode:
		var settings map[string]any
		if err := json.Unmarshal(raw, &settings); err != nil {
			return false, fmt.Errorf("scaffold settings are malformed: %w", err)
		}
		return claudeCodeHasGoalrail(settings), nil
	default:
		return false, fmt.Errorf("unsupported scaffold %q", scaffold)
	}
}

func connectCodex(plan ConnectionPlan) error {
	command := shellQuote(plan.Executable) + " hook"
	block := strings.Join([]string{
		"",
		blockBegin,
		"[[hooks.SessionStart]]",
		"",
		"[[hooks.SessionStart.hooks]]",
		`type = "command"`,
		"command = " + tomlQuote(command),
		"",
		"[[hooks.Stop]]",
		"",
		"[[hooks.Stop.hooks]]",
		`type = "command"`,
		"command = " + tomlQuote(command),
		blockEnd,
		"",
	}, "\n")
	return appendToFile(plan.ConfigPath, block)
}

func connectClaudeCode(plan ConnectionPlan) error {
	settings, err := readJSONObject(plan.ConfigPath)
	if err != nil {
		return err
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	entry := func() any {
		return []any{map[string]any{
			"hooks": []any{map[string]any{
				"type":    "command",
				"command": shellQuote(plan.Executable) + " hook",
			}},
		}}
	}
	for _, event := range []string{"SessionStart", "Stop"} {
		existing, _ := hooks[event].([]any)
		hooks[event] = append(existing, entry().([]any)...)
	}
	settings["hooks"] = hooks
	return writeJSONObject(plan.ConfigPath, settings)
}

func claudeCodeHasGoalrail(settings map[string]any) bool {
	hooks, _ := settings["hooks"].(map[string]any)
	for _, event := range []string{"SessionStart", "Stop"} {
		groups, _ := hooks[event].([]any)
		for _, group := range groups {
			if commandGroupMentionsGoalrail(group) {
				return true
			}
		}
	}
	return false
}

func commandGroupMentionsGoalrail(group any) bool {
	asMap, _ := group.(map[string]any)
	handlers, _ := asMap["hooks"].([]any)
	for _, handler := range handlers {
		handlerMap, _ := handler.(map[string]any)
		command, _ := handlerMap["command"].(string)
		if strings.Contains(command, "gr' hook") || strings.HasSuffix(command, "gr hook") {
			return true
		}
	}
	return false
}

func removeClaudeCodeHooks(configPath string) (bool, error) {
	settings, err := readJSONObject(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		return false, nil
	}
	removed := false
	for _, event := range []string{"SessionStart", "Stop"} {
		groups, _ := hooks[event].([]any)
		kept := make([]any, 0, len(groups))
		for _, group := range groups {
			if commandGroupMentionsGoalrail(group) {
				removed = true
				continue
			}
			kept = append(kept, group)
		}
		if len(kept) == 0 {
			delete(hooks, event)
			continue
		}
		hooks[event] = kept
	}
	if !removed {
		return false, nil
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}
	return true, writeJSONObject(configPath, settings)
}

func removeManagedBlock(configPath string) (bool, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	content := string(raw)
	begin := strings.Index(content, blockBegin)
	if begin < 0 {
		return false, nil
	}
	end := strings.Index(content[begin:], blockEnd)
	if end < 0 {
		return false, errors.New("managed block is not terminated; refusing to guess its extent")
	}
	tail := begin + end + len(blockEnd)
	for tail < len(content) && content[tail] == '\n' {
		tail++
	}
	trimmed := strings.TrimRight(content[:begin], "\n")
	if trimmed != "" {
		trimmed += "\n"
	}
	return true, os.WriteFile(configPath, []byte(trimmed+content[tail:]), 0o644)
}

func readJSONObject(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]any{}, nil
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf("scaffold settings are malformed: %w", err)
	}
	return settings, nil
}

func writeJSONObject(path string, settings map[string]any) error {
	encoded, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}

func appendToFile(path, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(block)
	return err
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func tomlQuote(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
