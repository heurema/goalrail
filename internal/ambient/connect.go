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

	// Repair reports a registration that is recognisably ours but does not name
	// the executable this connection was invoked with. It is tracked apart from
	// plain absence because replacing a handler changes a hook definition the
	// user may already have reviewed, which the caller has to disclose.
	Repair bool `json:"repair"`

	// RegisteredExecutable is the path the existing registration names, when it
	// differs from the one being registered. Reported detail rather than a
	// promoted contract: it says what a repair replaces.
	RegisteredExecutable string `json:"registered_executable,omitempty"`
}

// managedEvents names the session events a connection registers. Every event
// listed here must be present for the attachment to work, so presence, repair,
// and removal all walk the same list.
func managedEvents() []string { return []string{sessionStartEvent, stopEvent} }

const (
	sessionStartEvent = "SessionStart"
	stopEvent         = "Stop"
)

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
	// Presence and currency are separate questions. The marker test answers "is
	// this registration ours", which is what health asks and must keep asking
	// from whatever binary it happens to run as. Only connection can ask "does
	// it name the executable I am", because only connection has that executable
	// to compare against — and a registration naming a different one is not the
	// registration this command would write.
	stale, err := staleExecutable(scaffold, configPath, executable)
	if err != nil {
		return ConnectionPlan{}, err
	}
	return ConnectionPlan{
		Scaffold:             scaffold,
		ConfigPath:           configPath,
		AlreadyPresent:       present && stale == "",
		Executable:           executable,
		Repair:               stale != "",
		RegisteredExecutable: stale,
	}, nil
}

// staleExecutable returns the executable a managed handler names when it is not
// the given one, and an empty string when every managed handler names it or
// there is no managed handler at all.
//
// A handler whose executable cannot be extracted counts as neither: there is
// nothing to compare, and rewriting a registration we cannot read would be a
// guess.
func staleExecutable(scaffold Scaffold, configPath, executable string) (string, error) {
	registered, err := registeredExecutables(scaffold, configPath)
	if err != nil {
		return "", err
	}
	for _, candidate := range registered {
		if candidate != executable {
			return candidate, nil
		}
	}
	return "", nil
}

// registeredExecutables returns the executable named by every managed handler,
// read the way each scaffold stores it: from file content for the bracketed
// block, and from decoded command strings for the settings object.
func registeredExecutables(scaffold Scaffold, configPath string) ([]string, error) {
	switch scaffold {
	case ScaffoldCodex:
		raw, err := os.ReadFile(configPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}
		return executablesInText(string(raw)), nil
	case ScaffoldClaudeCode:
		settings, err := readJSONObject(configPath)
		if err != nil {
			return nil, err
		}
		var found []string
		for _, event := range managedEvents() {
			forEachManagedCommand(settings, event, func(command string) {
				found = append(found, executablesInText(command)...)
			})
		}
		return found, nil
	default:
		return nil, fmt.Errorf("unsupported scaffold %q", scaffold)
	}
}

// executablesInText extracts the executable of every managed command appearing
// in the given text, which may be a whole configuration file or one command.
func executablesInText(content string) []string {
	var found []string
	for offset := 0; ; {
		index := strings.Index(content[offset:], managedMarker)
		if index < 0 {
			return found
		}
		index += offset
		if path := executableBefore(content[:index]); path != "" {
			found = append(found, path)
		}
		offset = index + len(managedMarker)
	}
}

// executableBefore extracts the quoted executable a managed command opens with,
// given everything that precedes the marker.
func executableBefore(prefix string) string {
	closing := strings.LastIndex(prefix, "'")
	if closing < 0 {
		return ""
	}
	opening := strings.LastIndex(prefix[:closing], "'")
	if opening < 0 {
		return ""
	}
	return prefix[opening+1 : closing]
}

// forEachManagedCommand visits every managed handler command registered for one
// event in a settings object.
func forEachManagedCommand(settings map[string]any, event string, visit func(command string)) {
	hooks, _ := settings["hooks"].(map[string]any)
	groups, _ := hooks[event].([]any)
	for _, group := range groups {
		asMap, _ := group.(map[string]any)
		handlers, _ := asMap["hooks"].([]any)
		for _, handler := range handlers {
			handlerMap, _ := handler.(map[string]any)
			command, _ := handlerMap["command"].(string)
			if isManagedCommand(command) {
				visit(command)
			}
		}
	}
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
		content := string(raw)
		if !strings.Contains(content, blockBegin) {
			return false, nil
		}
		// The opening marker alone is not the registration. If a stanza is
		// removed while the marker survives, the announcement or the question
		// retention is gone while everything still looks installed.
		for _, event := range []string{"[[hooks.SessionStart.hooks]]", "[[hooks.Stop.hooks]]"} {
			if !strings.Contains(content, event) {
				return false, nil
			}
		}
		return strings.Contains(content, managedMarker), nil
	case ScaffoldClaudeCode:
		// An empty settings file is an empty configuration, exactly as the
		// write path treats it; failing here would make connection impossible
		// for a user whose scaffold created a zero-length file.
		if len(strings.TrimSpace(string(raw))) == 0 {
			return false, nil
		}
		var settings map[string]any
		if err := json.Unmarshal(raw, &settings); err != nil {
			return false, fmt.Errorf("scaffold settings are malformed: %w", err)
		}
		// The connection is present only when every event is registered. A
		// partial registration — say Stop was hand-removed — must read as
		// absent, or re-running the consented command could never repair it.
		return claudeCodeSessionStartIsScoped(settings) &&
			claudeCodeHasGoalrail(settings, stopEvent), nil
	default:
		return false, fmt.Errorf("unsupported scaffold %q", scaffold)
	}
}

func connectCodex(plan ConnectionPlan) error {
	// Remove any existing managed block before writing, unconditionally rather
	// than only when repairing. This path is reached exactly when something must
	// be written, so keeping an existing block is never right: appending beside
	// it would leave two handlers per event — every hook firing twice — and a
	// removal that finds only the first, so disconnection would leave residue
	// that still looks like an attachment. Removal refuses on a block whose end
	// marker is missing, which fails the repair loudly instead of writing beside
	// something whose extent is unknown.
	if _, err := removeManagedBlock(plan.ConfigPath); err != nil {
		return err
	}
	command := managedCommand(plan.Executable)
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

// openingSessionMatcher names the only session-start occurrence that opens a
// session on a scaffold that distinguishes them. Resumption, clearing,
// compaction, and forking all recur inside a session that has already started.
//
// An omitted matcher means "every occurrence" there, which would repeat the
// announcement inside one session. The first supported scaffold avoids that in
// its transport by inspecting the occurrence; this one gives the transport
// nothing to inspect, so the constraint has to live in the registration.
const openingSessionMatcher = "startup"

// managedMarker travels inside our handler command so removal identifies the
// exact handler connection installed, never a lookalike. The hook entry point
// ignores arguments, so the marker is inert at run time.
const managedMarker = "--managed-by=goalrail"

func managedCommand(executable string) string {
	return shellQuote(executable) + " hook " + managedMarker
}

func isManagedCommand(command string) bool {
	return strings.Contains(command, managedMarker)
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
	handler := func() any {
		return map[string]any{
			"type":    "command",
			"command": managedCommand(plan.Executable),
		}
	}

	// Session start is registered against the opening occurrence only, naming the
	// current executable. An existing registration of ours that fails either test
	// is replaced rather than left alone: an unscoped one keeps firing on every
	// occurrence, and one naming a moved binary cannot run, and in both cases a
	// user who connected with an earlier version or from a different binary would
	// otherwise have no way to repair it.
	if !claudeCodeSessionStartIsScoped(settings) ||
		!claudeCodeEventIsCurrent(settings, sessionStartEvent, plan.Executable) {
		groups := stripManagedHandlers(hooks[sessionStartEvent])
		hooks[sessionStartEvent] = append(groups, map[string]any{
			"matcher": openingSessionMatcher,
			"hooks":   []any{handler()},
		})
	}

	// Stop has no occurrence to distinguish, so it is registered plainly.
	// Reconcile per event: an event that is missing gains a registration, a stale
	// one is replaced, and one that is already correct keeps its exact bytes —
	// and with them whatever review the user has already given it.
	if !claudeCodeEventIsCurrent(settings, stopEvent, plan.Executable) {
		groups := stripManagedHandlers(hooks[stopEvent])
		hooks[stopEvent] = append(groups, map[string]any{
			"hooks": []any{handler()},
		})
	}

	settings["hooks"] = hooks
	return writeJSONObject(plan.ConfigPath, settings)
}

// claudeCodeSessionStartIsScoped reports whether our session-start handler is
// registered against the opening occurrence and nowhere else.
func claudeCodeSessionStartIsScoped(settings map[string]any) bool {
	hooks, _ := settings["hooks"].(map[string]any)
	groups, _ := hooks["SessionStart"].([]any)
	scoped := false
	for _, group := range groups {
		asMap, _ := group.(map[string]any)
		if !groupHasManagedHandler(group) {
			continue
		}
		matcher, _ := asMap["matcher"].(string)
		if matcher != openingSessionMatcher {
			// Ours, but firing on occurrences that do not open a session.
			return false
		}
		scoped = true
	}
	return scoped
}

func groupHasManagedHandler(group any) bool {
	asMap, _ := group.(map[string]any)
	handlers, _ := asMap["hooks"].([]any)
	for _, entry := range handlers {
		entryMap, _ := entry.(map[string]any)
		command, _ := entryMap["command"].(string)
		if isManagedCommand(command) {
			return true
		}
	}
	return false
}

// stripManagedHandlers removes our handlers from a group list, preserving
// foreign handlers and any group that still holds one.
func stripManagedHandlers(value any) []any {
	groups, _ := value.([]any)
	kept := make([]any, 0, len(groups))
	for _, group := range groups {
		asMap, _ := group.(map[string]any)
		handlers, _ := asMap["hooks"].([]any)
		keptHandlers := make([]any, 0, len(handlers))
		for _, entry := range handlers {
			entryMap, _ := entry.(map[string]any)
			command, _ := entryMap["command"].(string)
			if isManagedCommand(command) {
				continue
			}
			keptHandlers = append(keptHandlers, entry)
		}
		if len(keptHandlers) == 0 && len(handlers) > 0 {
			continue
		}
		if asMap != nil && len(handlers) > 0 {
			asMap["hooks"] = keptHandlers
		}
		kept = append(kept, group)
	}
	return kept
}

func claudeCodeHasGoalrail(settings map[string]any, event string) bool {
	found := false
	forEachManagedCommand(settings, event, func(string) { found = true })
	return found
}

// claudeCodeEventIsCurrent reports whether one event carries a registration of
// ours and every handler in it names the given executable. An event with no
// managed handler is not current: there is nothing there to keep.
func claudeCodeEventIsCurrent(settings map[string]any, event, executable string) bool {
	present, current := false, true
	forEachManagedCommand(settings, event, func(command string) {
		present = true
		for _, path := range executablesInText(command) {
			if path != executable {
				current = false
			}
		}
	})
	return present && current
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
	for _, event := range managedEvents() {
		groups, _ := hooks[event].([]any)
		keptGroups := make([]any, 0, len(groups))
		for _, group := range groups {
			// Filter at the handler level: a group can mix our handler with a
			// foreign one, and disconnection may remove only what connection
			// added.
			asMap, _ := group.(map[string]any)
			handlers, _ := asMap["hooks"].([]any)
			keptHandlers := make([]any, 0, len(handlers))
			for _, handler := range handlers {
				handlerMap, _ := handler.(map[string]any)
				command, _ := handlerMap["command"].(string)
				if isManagedCommand(command) {
					removed = true
					continue
				}
				keptHandlers = append(keptHandlers, handler)
			}
			if len(keptHandlers) == 0 && len(handlers) > 0 {
				continue
			}
			if len(handlers) > 0 {
				asMap["hooks"] = keptHandlers
			}
			keptGroups = append(keptGroups, group)
		}
		if len(keptGroups) == 0 {
			delete(hooks, event)
			continue
		}
		hooks[event] = keptGroups
	}
	if !removed {
		return false, nil
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	} else {
		settings["hooks"] = hooks
	}
	if len(settings) == 0 {
		// Connection created this file for its own registration; an empty
		// object left behind is residue.
		if err := os.Remove(configPath); err != nil {
			return false, err
		}
		_ = os.Remove(filepath.Dir(configPath))
		return true, nil
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
	remaining := trimmed + content[tail:]
	if strings.TrimSpace(remaining) == "" {
		// Connection created this file for its own block; leaving an empty
		// file and directory behind is residue.
		if err := os.Remove(configPath); err != nil {
			return false, err
		}
		_ = os.Remove(filepath.Dir(configPath))
		return true, nil
	}
	return true, os.WriteFile(configPath, []byte(remaining), 0o644)
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
