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
	Scope          Scope    `json:"scope"`
	ConfigPath     string   `json:"config_path"`
	AlreadyPresent bool     `json:"already_present"`
	Executable     string   `json:"executable"`

	// Events are the events this registration writes, and SupersededPresent
	// reports a handler of ours sitting on an event this arrangement no longer
	// writes — which fires on a cadence the retention rule was never written for.
	Events            []string `json:"events"`
	SupersededPresent []string `json:"superseded_present,omitempty"`

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

// marker lines bracket everything a connection adds, so disconnection can
// remove exactly that and nothing else. Editing a user's own configuration
// demands a removal that is provably complete.
const (
	blockBegin = "# >>> goalrail ambient (managed) >>>"
	blockEnd   = "# <<< goalrail ambient (managed) <<<"
)

// ConfigPath returns the user-level configuration file a scaffold reads.
//
// It stays meaningful for every scaffold, including one whose registration has
// moved into the repository: that file is where a registration from the earlier
// arrangement still sits, and reporting it is how the user learns to remove it.
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

// ErrRegistersPerRepository reports a scaffold whose registration belongs inside
// a repository, so the connection command has nothing to do for it.
var ErrRegistersPerRepository = errors.New("this scaffold registers per repository")

// PlanConnection reports what the user-scope connection command would do,
// without doing it.
//
// A scaffold that registers inside the repository is refused here rather than
// served: writing its hooks at user scope would invoke them in every session the
// user starts anywhere, which is the arrangement this version replaced.
func PlanConnection(scaffold Scaffold, home, executable string) (ConnectionPlan, error) {
	scope, err := ScopeOf(scaffold)
	if err != nil {
		return ConnectionPlan{}, err
	}
	if scope != ScopeUser {
		return ConnectionPlan{}, fmt.Errorf("%w: run `gr init` in the repository instead", ErrRegistersPerRepository)
	}
	target, err := RegistrationTarget(scaffold, home, "")
	if err != nil {
		return ConnectionPlan{}, err
	}
	return PlanRegistration(target, executable)
}

// PlanRegistration reports what registering at one concrete target would do.
func PlanRegistration(target Target, executable string) (ConnectionPlan, error) {
	if !filepath.IsAbs(executable) {
		return ConnectionPlan{}, errors.New("registration requires the absolute gr executable path")
	}
	if target.Scope == ScopeRepository {
		// A repository-scope write must land inside the repository. A settings
		// directory or file that is a symlink resolves the write somewhere else —
		// a repository shipping such a link would receive the registration into
		// whatever it points at, including the user's own configuration.
		if err := EnsureWriteWithinRepository(target.Repository, target.Path); err != nil {
			return ConnectionPlan{}, err
		}
	}
	present, err := isConnected(target.Scaffold, target.Path)
	if err != nil {
		return ConnectionPlan{}, err
	}
	// Presence and currency are separate questions. The marker test answers "is
	// this registration ours", which is what health asks and must keep asking
	// from whatever binary it happens to run as. Only registration can ask "does
	// it name the executable I am", because only it has that executable
	// to compare against — and a registration naming a different one is not the
	// registration this command would write.
	stale, err := staleExecutable(target.Scaffold, target.Path, executable)
	if err != nil {
		return ConnectionPlan{}, err
	}
	// A handler of ours on an event this arrangement no longer writes is its own
	// kind of staleness. On the scaffold that moved, the event it used to name
	// fires once per turn, so a question left at the reserved path would be
	// retained again on every turn — one session's single question multiplied
	// rather than two sessions' questions separated.
	superseded, err := supersededPresent(target.Scaffold, target.Path)
	if err != nil {
		return ConnectionPlan{}, err
	}
	return ConnectionPlan{
		Scaffold:             target.Scaffold,
		Scope:                target.Scope,
		ConfigPath:           target.Path,
		AlreadyPresent:       present && stale == "" && len(superseded) == 0,
		Executable:           executable,
		Events:               managedEvents(target.Scaffold),
		SupersededPresent:    superseded,
		Repair:               stale != "" || len(superseded) > 0,
		RegisteredExecutable: stale,
	}, nil
}

// supersededPresent returns the events this arrangement no longer writes that
// still carry a handler of ours.
func supersededPresent(scaffold Scaffold, configPath string) ([]string, error) {
	events := SupersededEvents(scaffold)
	if len(events) == 0 {
		return nil, nil
	}
	settings, err := readJSONObject(configPath)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, event := range events {
		if claudeCodeHasGoalrail(settings, event) {
			found = append(found, event)
		}
	}
	return found, nil
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
// read the way each scaffold stores it: from the commands inside the bracketed
// block, and from the decoded command strings of the settings object.
//
// Both paths reach the command through that scaffold's own quoting rather than
// scanning the file as text. Text scanning would also find a marker in a
// commented-out or hand-kept copy of an old registration, and report a perfectly
// current attachment as stale forever.
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
		return executablesInCommands(managedBlockCommands(string(raw))), nil
	case ScaffoldClaudeCode:
		settings, err := readJSONObject(configPath)
		if err != nil {
			return nil, err
		}
		var commands []string
		for _, event := range knownEvents(scaffold) {
			forEachManagedCommand(settings, event, func(command string) {
				commands = append(commands, command)
			})
		}
		return executablesInCommands(commands), nil
	default:
		return nil, fmt.Errorf("unsupported scaffold %q", scaffold)
	}
}

func executablesInCommands(commands []string) []string {
	var found []string
	for _, command := range commands {
		if path := executableFromCommand(command); path != "" {
			found = append(found, path)
		}
	}
	return found
}

// managedBlockCommands returns the managed handler commands registered inside the
// marker-bracketed block, decoded from the file's own escaping.
//
// When the end marker is missing the block has no knowable extent, so everything
// after the opening marker is considered. That is deliberate: it keeps a stale
// registration visible in a corrupt file, and the write path refuses to act on
// such a block rather than writing beside content it cannot delimit.
func managedBlockCommands(content string) []string {
	begin := strings.Index(content, blockBegin)
	if begin < 0 {
		return nil
	}
	block := content[begin:]
	if end := strings.Index(block, blockEnd); end >= 0 {
		block = block[:end]
	}
	var found []string
	for _, line := range strings.Split(block, "\n") {
		if command, ok := tomlCommandValue(line); ok && isManagedCommand(command) {
			found = append(found, command)
		}
	}
	return found
}

// tomlCommandValue decodes a `command = "..."` assignment, reversing tomlQuote.
func tomlCommandValue(line string) (string, bool) {
	rest := strings.TrimSpace(line)
	if !strings.HasPrefix(rest, "command") {
		return "", false
	}
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "command"))
	if !strings.HasPrefix(rest, "=") {
		return "", false
	}
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
	if len(rest) < 2 || !strings.HasPrefix(rest, `"`) || !strings.HasSuffix(rest, `"`) {
		return "", false
	}
	body := rest[1 : len(rest)-1]
	var decoded strings.Builder
	for index := 0; index < len(body); index++ {
		if body[index] == '\\' && index+1 < len(body) &&
			(body[index+1] == '\\' || body[index+1] == '"') {
			decoded.WriteByte(body[index+1])
			index++
			continue
		}
		decoded.WriteByte(body[index])
	}
	return decoded.String(), true
}

// executableFromCommand decodes the executable a managed command opens with,
// reversing the shell quoting the command was written with.
//
// Reading the text between the last two apostrophes is wrong for a path that
// contains one: shellQuote encodes such a path with the '"'"' sequence, the last
// pair of apostrophes then lands inside that sequence, and only the tail of the
// path comes back. A user whose home directory holds an apostrophe would have
// every connection report their current registration as stale and rewrite it.
func executableFromCommand(command string) string {
	if !strings.HasPrefix(command, "'") {
		return ""
	}
	const literalQuote = `'"'"'`
	var decoded strings.Builder
	for index := 1; index < len(command); {
		if command[index] != '\'' {
			decoded.WriteByte(command[index])
			index++
			continue
		}
		// An apostrophe here either closes the quoted path or opens the sequence
		// that stands for one literal apostrophe inside it.
		if strings.HasPrefix(command[index:], literalQuote) {
			decoded.WriteByte('\'')
			index += len(literalQuote)
			continue
		}
		return decoded.String()
	}
	// Unterminated quoting: nothing can be read with confidence.
	return ""
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

// Disconnect removes everything a registration added, leaving no residue.
//
// It spans every scope a scaffold's registration may occupy, not only the one
// this version writes: a registration left behind by the earlier arrangement
// still fires, and a disconnection that missed it would report success while the
// hooks kept running. This is also the migration path off that arrangement, which
// is why it belongs to the consented command that owns user configuration rather
// than to initialization.
func Disconnect(scaffold Scaffold, home, repositoryRoot string) (bool, error) {
	target, err := RegistrationTarget(scaffold, home, repositoryRoot)
	if err != nil {
		// A repository-scope scaffold with no repository can still have a
		// user-scope registration to clean up, so this is not fatal.
		if superseded, present := SupersededTarget(scaffold, home); present {
			return unregister(superseded)
		}
		return false, err
	}
	if target.Scope == ScopeRepository {
		// Removal edits the same file registration writes, so it honours the same
		// containment: a checkout shipping a symlinked settings path must not
		// redirect a disconnect into the user's own configuration.
		if containErr := EnsureWriteWithinRepository(target.Repository, target.Path); containErr != nil {
			return false, containErr
		}
	}
	removed, err := unregister(target)
	if err != nil {
		return removed, err
	}
	if superseded, present := SupersededTarget(scaffold, home); present {
		alsoRemoved, supersededErr := unregister(superseded)
		if supersededErr != nil {
			return removed, supersededErr
		}
		removed = removed || alsoRemoved
	}
	return removed, nil
}

// unregister removes our handlers from one concrete target.
func unregister(target Target) (bool, error) {
	switch target.Scaffold {
	case ScaffoldCodex:
		return removeManagedBlock(target.Path)
	case ScaffoldClaudeCode:
		return removeClaudeCodeHooks(target.Scaffold, target.Path)
	default:
		return false, fmt.Errorf("unsupported scaffold %q", target.Scaffold)
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
		for _, event := range managedEvents(scaffold) {
			if !strings.Contains(content, "[[hooks."+event+".hooks]]") {
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
		// The connection is present only when every event this arrangement
		// registers carries a handler. A partial registration — say one event was
		// hand-removed — must read as absent, or re-running the consented command
		// could never repair it.
		if !claudeCodeSessionStartIsScoped(settings) {
			return false, nil
		}
		for _, event := range managedEvents(scaffold) {
			if event == sessionStartEvent {
				continue
			}
			if !claudeCodeHasGoalrail(settings, event) {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported scaffold %q", scaffold)
	}
}

func connectCodex(plan ConnectionPlan) error {
	// Remove every existing managed block before writing, unconditionally rather
	// than only when repairing. This path is reached exactly when something must
	// be written, so keeping an existing block is never right: appending beside
	// one would leave two handlers per event — every hook firing twice — and a
	// removal that finds only the first, so disconnection would leave residue
	// that still looks like an attachment.
	//
	// The loop matters because removal handles one block per call, and a
	// configuration can already carry more than one: an earlier write path
	// appended a complete block beside a partial one. Removing a single block and
	// appending a fresh one would leave that state exactly as broken as it was.
	// Removal refuses on a block whose end marker is missing, which fails the
	// write loudly instead of writing beside something it cannot delimit.
	for {
		removed, err := removeManagedBlock(plan.ConfigPath)
		if err != nil {
			return err
		}
		if !removed {
			break
		}
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

	// The remaining events have no occurrence to distinguish, so they are
	// registered plainly. Reconcile per event: an event that is missing gains a
	// registration, a stale one is replaced, and one that is already correct keeps
	// its exact bytes — and with them whatever review the user has already given
	// it.
	for _, event := range managedEvents(plan.Scaffold) {
		if event == sessionStartEvent {
			continue
		}
		if claudeCodeEventIsCurrent(settings, event, plan.Executable) {
			continue
		}
		groups := stripManagedHandlers(hooks[event])
		hooks[event] = append(groups, map[string]any{
			"hooks": []any{handler()},
		})
	}

	// A handler of ours on an event this arrangement no longer writes is removed
	// rather than left beside the new one. On this scaffold that event fires once
	// per turn, so leaving it would retain one session's single question again on
	// every turn — and the user would see a growing pile of records for one
	// question they asked once.
	for _, event := range SupersededEvents(plan.Scaffold) {
		if !claudeCodeHasGoalrail(settings, event) {
			continue
		}
		remaining := stripManagedHandlers(hooks[event])
		if len(remaining) == 0 {
			delete(hooks, event)
			continue
		}
		hooks[event] = remaining
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
		// A command whose executable cannot be read is neither current nor
		// stale, matching staleExecutable: there is nothing to compare, and
		// rewriting a registration we cannot read would be a guess.
		if path := executableFromCommand(command); path != "" && path != executable {
			current = false
		}
	})
	return present && current
}

func removeClaudeCodeHooks(scaffold Scaffold, configPath string) (bool, error) {
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
	// Removal walks every event a registration of ours may occupy, including one
	// this arrangement no longer writes: an event nobody looks at keeps firing.
	for _, event := range knownEvents(scaffold) {
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
