package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/adapters/codex"
	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/localrun"
)

// hookEnvironmentSession is the session identifier a scaffold exposes to its
// hooks. It is read opportunistically: a missing value costs attribution
// detail, never the question itself.
const hookEnvironmentSession = "CODEX_SESSION_ID"

func runInit(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("init", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "repository to initialize")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("init accepts no positional arguments")
	}
	root, err := filepath.Abs(*repository)
	if err != nil {
		return err
	}
	marker, created, err := ambient.Initialize(root, time.Now)
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{
		"repository":     root,
		"created":        created,
		"initialized_at": marker.InitializedAt,
		"marker":         ambient.MarkerPath,
	})
}

func runConnect(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("connect", flag.ContinueOnError)
	set.SetOutput(stderr)
	scaffold := set.String("scaffold", "", "codex or claude-code")
	confirm := set.Bool("yes", false, "consent to modifying the scaffold configuration")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	selected, err := parseScaffold(*scaffold)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	plan, err := ambient.PlanConnection(selected, home, resolved)
	if err != nil {
		return err
	}
	// Consent is to a concrete act, not a promise: without --yes the plan is
	// shown and nothing is written.
	if !*confirm {
		return writeJSON(stdout, map[string]any{
			"plan":       plan,
			"applied":    false,
			"next":       "re-run with --yes to apply",
			"registers":  []string{"SessionStart", "Stop"},
			"reversible": "gr disconnect --scaffold " + string(selected),
		})
	}
	changed, err := ambient.Connect(plan)
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{
		"plan":    plan,
		"applied": changed,
		// Registration alone does not make the attachment act. Saying nothing
		// here is the worst outcome available: the user connects, works, sees
		// nothing happen, and reasonably concludes Goalrail is broken.
		"notice":        ambient.ConnectionNotice(selected),
		"trust_surface": ambient.TrustSurface(selected),
		"active_now":    false,
	})
}

func runHealth(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("health", flag.ContinueOnError)
	set.SetOutput(stderr)
	scaffold := set.String("scaffold", string(ambient.ScaffoldCodex), "codex or claude-code")
	repository := set.String("repo", ".", "repository to check")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	selected, err := parseScaffold(*scaffold)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	root, err := filepath.Abs(*repository)
	if err != nil {
		return err
	}
	state, err := ambient.Inspect(selected, home, root)
	if err != nil {
		return err
	}
	return writeJSON(stdout, state)
}

func runDisconnect(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("disconnect", flag.ContinueOnError)
	set.SetOutput(stderr)
	scaffold := set.String("scaffold", "", "codex or claude-code")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	selected, err := parseScaffold(*scaffold)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	removed, err := ambient.Disconnect(selected, home)
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{"scaffold": selected, "removed": removed})
}

// runHook is the ambient entry point. It is fail-quiet by contract: every
// internal error ends as a clean exit toward the scaffold, because a hook that
// failed loudly would break an ordinary session that has nothing to do with
// Goalrail. Errors worth keeping are recorded in the state root.
func runHook(args []string, stdin io.Reader, stdout io.Writer) error {
	// The marker check comes before the payload is read, not merely before it
	// is acted on. A hook payload can carry prompts, transcript paths, and
	// authorization fields; reading one from a repository the user never
	// connected would observe unrelated work regardless of what happened next.
	root, err := os.Getwd()
	if err != nil {
		return nil
	}
	if !ambient.IsInitialized(root) {
		return nil
	}
	event, sessionRef := readHookEvent(stdin)
	if event == "" {
		return nil
	}
	store, err := localrun.NewStore("")
	if err != nil {
		return nil
	}

	switch strings.ToLower(event) {
	case "sessionstart":
		announcement, _, startErr := ambient.StartSession(store, root, time.Now)
		if startErr != nil || announcement == "" {
			recordAmbientError(store, "session-start", startErr)
			return nil
		}
		rendered, renderErr := codex.RenderSessionStartHookResponse(
			announcement,
			codex.StartupHookSource,
		)
		if renderErr != nil {
			recordAmbientError(store, "render", renderErr)
			return nil
		}
		fmt.Fprintln(stdout, rendered)
	case "stop":
		if _, stopErr := ambient.StopSession(
			store,
			root,
			sessionRef,
			ambient.OpenSpecIntents{},
			time.Now,
		); stopErr != nil {
			recordAmbientError(store, "session-stop", stopErr)
		}
	}
	return nil
}

// readHookEvent extracts the event name and session reference from the payload
// the scaffold writes to stdin. An unreadable payload yields an empty event,
// which the caller treats as "do nothing".
func readHookEvent(stdin io.Reader) (event string, sessionRef string) {
	if stdin == nil {
		return "", ""
	}
	raw, err := io.ReadAll(io.LimitReader(stdin, 1<<20))
	if err != nil || len(raw) == 0 {
		return "", ""
	}
	var payload struct {
		HookEventName string `json:"hook_event_name"`
		EventName     string `json:"hookEventName"`
		SessionID     string `json:"session_id"`
		SessionIDAlt  string `json:"sessionId"`
		Source        string `json:"source"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", ""
	}
	event = payload.HookEventName
	if event == "" {
		event = payload.EventName
	}
	sessionRef = payload.SessionID
	if sessionRef == "" {
		sessionRef = payload.SessionIDAlt
	}
	if sessionRef == "" {
		sessionRef = os.Getenv(hookEnvironmentSession)
	}
	// Only the source that opens a session carries the announcement; the
	// renderer enforces this too, and agreeing here keeps resumption silent
	// even if a scaffold omits the source.
	if strings.EqualFold(event, "SessionStart") &&
		payload.Source != "" &&
		!strings.EqualFold(payload.Source, codex.StartupHookSource) {
		return "", ""
	}
	return event, sessionRef
}

func recordAmbientError(store *localrun.Store, stage string, cause error) {
	if store == nil || cause == nil {
		return
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	_ = store.WriteJSONOnce(
		filepath.ToSlash(filepath.Join("ambient", "errors", stamp+".json")),
		map[string]string{"stage": stage, "error": cause.Error()},
		true,
	)
}

func parseScaffold(value string) (ambient.Scaffold, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(ambient.ScaffoldCodex):
		return ambient.ScaffoldCodex, nil
	case string(ambient.ScaffoldClaudeCode):
		return ambient.ScaffoldClaudeCode, nil
	default:
		return "", fmt.Errorf("connect requires --scaffold codex|claude-code")
	}
}

// nowUTC is the ambient clock. It exists so tests can construct markers
// through the same call the commands use.
func nowUTC() time.Time { return time.Now().UTC() }
