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
	"github.com/heurema/goalrail/internal/harness"
	"github.com/heurema/goalrail/internal/localrun"
)

// hookEnvironmentSession is the session identifier a scaffold exposes to its
// hooks. It is read opportunistically: a missing value costs attribution
// detail, never the question itself.
const hookEnvironmentSession = "CODEX_SESSION_ID"

// initReport is what initialization tells the user it did. Everything it changed,
// everything it left alone, and every remaining step is named here: an install
// that reports only success leaves the user guessing which half of the harness
// they have.
type initReport struct {
	Repository string `json:"repository"`

	Marker        string    `json:"marker"`
	MarkerCreated bool      `json:"marker_created"`
	InitializedAt time.Time `json:"initialized_at"`

	Canon  string                `json:"canon"`
	Config harness.ConfigOutcome `json:"config"`
	Files  []harness.FileOutcome `json:"files"`
	Ignore []string              `json:"ignore_entries_added,omitempty"`

	Registration *registrationReport `json:"registration,omitempty"`

	// Invocation is the exact command this repository is now driven by, including
	// the explicit schema argument the pinned version's defect makes mandatory.
	Invocation string `json:"invocation"`

	Notices []string `json:"notices,omitempty"`
	Next    []string `json:"next,omitempty"`
}

type registrationReport struct {
	Scaffold  ambient.Scaffold `json:"scaffold"`
	Scope     ambient.Scope    `json:"scope"`
	Path      string           `json:"path"`
	Events    []string         `json:"events"`
	Applied   bool             `json:"applied"`
	Repaired  bool             `json:"repaired,omitempty"`
	Refused   string           `json:"refused,omitempty"`
	ActiveNow bool             `json:"active_now"`
	Notice    string           `json:"notice,omitempty"`
}

func runInit(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("init", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "repository to initialize")
	scaffold := set.String("scaffold", "", "codex or claude-code; omit to detect")
	confirmSchema := set.Bool("confirm-schema-switch", false,
		"switch an OpenSpec configuration that names another custom schema")
	fixIgnore := set.Bool("fix-gitignore", false,
		"add the ignore entries the registration and the marker need")
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
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	report := initReport{
		Repository: root,
		Marker:     ambient.MarkerPath,
		Invocation: harness.PinnedNewChange,
	}

	// The configuration comes first: a foreign custom schema stops initialization
	// before anything is written, so a refusal leaves the repository as it was.
	config, err := harness.EnsureConfig(root, *confirmSchema)
	if err != nil {
		return err
	}
	report.Config = config

	files, err := harness.Materialize(root, false)
	if err != nil {
		return err
	}
	report.Files = files
	canon, err := harness.CurrentCanon()
	if err != nil {
		return err
	}
	report.Canon = canon.ID
	for _, file := range files {
		if file.Action == harness.ActionKept && file.State == harness.FileEdited {
			report.Notices = append(report.Notices,
				file.Path+" differs from the canon and was left alone; `gr doctor` names it")
		}
	}

	marker, created, err := ambient.Initialize(root, time.Now)
	if err != nil {
		return err
	}
	report.MarkerCreated, report.InitializedAt = created, marker.InitializedAt

	if *fixIgnore {
		added, ignoreErr := ambient.AddIgnoreEntries(root, ambient.IgnoreEntries())
		if ignoreErr != nil {
			return ignoreErr
		}
		report.Ignore = added
	}

	selected, err := selectScaffold(*scaffold, home)
	if err != nil {
		return err
	}
	if err := registerSelected(&report, selected, home, root); err != nil {
		return err
	}

	// The marker is per clone, so committing it would make one user's
	// initialization a shared repository fact. Unlike the registration this is not
	// fatal: refusing to install the harness over an ignore rule would be a
	// disproportionate response to a recoverable condition.
	if ignored, _ := ambient.IgnoreState(root, ambient.MarkerPath); !ignored {
		report.Notices = append(report.Notices,
			"the marker at "+ambient.MarkerPath+" is not ignored, so a commit would make this "+
				"repository initialized for everyone; `gr init --fix-gitignore` adds the entry")
	}

	report.Next = append(report.Next,
		"commit the files above; they are yours, and Goalrail does not commit for you")
	return writeJSON(stdout, report)
}

// selectScaffold resolves which scaffold to register, by explicit name or by
// detection. Detection reads configuration presence, never session environment.
func selectScaffold(named, home string) ([]ambient.Scaffold, error) {
	if strings.TrimSpace(named) != "" {
		one, err := parseScaffold(named)
		if err != nil {
			return nil, err
		}
		return []ambient.Scaffold{one}, nil
	}
	return ambient.DetectScaffolds(home), nil
}

// registerSelected registers the repository-scope scaffold among the candidates
// and names the separate command for any that registers at user scope.
//
// Installing the harness never depends on which agent environment happens to be
// present: where nothing is detected, the overlay is still installed and the
// report says which command attaches later.
func registerSelected(
	report *initReport,
	candidates []ambient.Scaffold,
	home, root string,
) error {
	if len(candidates) == 0 {
		report.Next = append(report.Next,
			"no supported scaffold detected; run `gr init --scaffold <name>` or "+
				"`gr connect --scaffold codex --yes` when you have one")
		return nil
	}

	for _, candidate := range candidates {
		scope, err := ambient.ScopeOf(candidate)
		if err != nil {
			return err
		}
		if scope != ambient.ScopeRepository {
			// Registering inside the repository is externally blocked for this
			// scaffold, so its one consented user-scope command stays a separate
			// step rather than being implied as done.
			report.Next = append(report.Next,
				"run `gr connect --scaffold "+string(candidate)+" --yes` to attach "+
					string(candidate)+"; it registers at user scope")
			continue
		}
		if report.Registration != nil {
			continue
		}
		registration, err := registerRepositoryScope(candidate, home, root)
		if err != nil {
			return err
		}
		report.Registration = registration
	}
	return nil
}

func registerRepositoryScope(
	scaffold ambient.Scaffold,
	home, root string,
) (*registrationReport, error) {
	target, err := ambient.RegistrationTarget(scaffold, home, root)
	if err != nil {
		return nil, err
	}
	registration := &registrationReport{
		Scaffold: scaffold,
		Scope:    target.Scope,
		Path:     target.Path,
	}

	// Consent to run a command in one's own sessions is not transferable, so a
	// path a commit could carry is refused rather than written.
	ignored, ignoreErr := ambient.IgnoreState(root, ambient.RepositorySettingsPath)
	if !ignored {
		// The advice follows the cause. An ordinary missing entry is what the
		// flag adds; a tracked file or an unrunnable check is not, and telling the
		// user to re-run with a flag that cannot help would be a remedy that
		// prescribes itself.
		if ignoreErr != nil {
			registration.Refused = ignoreErr.Error() +
				"; --fix-gitignore cannot repair this — untrack the path or make the check runnable first"
		} else {
			registration.Refused = "the registration path is not ignored by version control, so a commit " +
				"would install these hooks in every teammate's sessions; add `" +
				ambient.RepositorySettingsPath + "` to .gitignore, or re-run with --fix-gitignore"
		}
		return registration, nil
	}

	executable, err := currentExecutable()
	if err != nil {
		return nil, err
	}
	plan, err := ambient.PlanRegistration(target, executable)
	if err != nil {
		return nil, err
	}
	registration.Events = plan.Events
	applied, err := ambient.Connect(plan)
	if err != nil {
		return nil, err
	}
	registration.Applied = applied
	registration.Repaired = plan.Repair

	state, inspectErr := ambient.Inspect(scaffold, home, root)
	if inspectErr == nil {
		registration.ActiveNow = state.Working
	}
	// A repair has two possible reasons and they are disclosed separately: saying
	// the registration named a different executable when only the event changed
	// would be a false statement about the user's own configuration.
	var notices []string
	if plan.RegisteredExecutable != "" {
		notices = append(notices, ambient.RepairNotice(scaffold, plan.RegisteredExecutable))
	}
	if len(plan.SupersededPresent) > 0 {
		notices = append(notices, ambient.SupersededEventNotice(scaffold, plan.SupersededPresent))
	}
	if len(notices) == 0 && applied {
		notices = append(notices, ambient.ConnectionNotice(scaffold))
	}
	registration.Notice = strings.Join(notices, " ")
	return registration, nil
}

// currentExecutable resolves the binary a registration must name, following
// symlinks so a registration cannot point at a link that moves.
func currentExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(executable)
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
	resolved, err := currentExecutable()
	if err != nil {
		return err
	}
	// A scaffold whose registration belongs inside the repository is refused here
	// rather than served: writing its hooks at user scope would invoke them in
	// every session the user starts anywhere, which is the arrangement `gr init`
	// replaced.
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
	response := map[string]any{
		"plan":          plan,
		"applied":       changed,
		"trust_surface": ambient.TrustSurface(selected),
	}
	// A repair changed the hook definition, so the trust record still sitting in
	// the configuration was made against the command that was just replaced.
	// Consulting it would report the attachment as active on the strength of a
	// record already known to be stale — which is the failure this repair exists
	// to remove, reappearing one layer down. This is the one moment the staleness
	// is known rather than merely unverifiable, so it is stated here.
	if plan.Repair {
		response["repaired"] = true
		response["active_now"] = false
		response["notice"] = ambient.RepairNotice(selected, plan.RegisteredExecutable)
		return writeJSON(stdout, response)
	}
	// Whether the hooks can run is a fact to observe, not to assume. Judge it by
	// trust alone: connection answers for the scaffold, while whether any given
	// repository participates is what `gr init` decides, and the current
	// directory is not necessarily one the user cares about here.
	if state, inspectErr := ambient.Inspect(selected, home, "."); inspectErr == nil {
		trusted := state.Trust == ambient.TrustRecorded
		response["active_now"] = trusted
		if !trusted {
			// Registration alone does not make the attachment act. Silence here
			// is the worst outcome available: the user connects, works, sees
			// nothing happen, and reasonably concludes Goalrail is broken.
			response["notice"] = ambient.ConnectionNotice(selected)
		}
	} else {
		response["notice"] = ambient.ConnectionNotice(selected)
	}
	return writeJSON(stdout, response)
}

func runHealth(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("health", flag.ContinueOnError)
	set.SetOutput(stderr)
	scaffold := set.String("scaffold", "", "codex or claude-code; omit to check all supported scaffolds")
	repository := set.String("repo", ".", "repository to check")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
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

	// Defaulting to one scaffold would hand a user who connected the other a
	// confident, wrong diagnosis: "nothing is connected, run connect" for a
	// scaffold they never chose. With no way to infer intent, check them all.
	selected := ambient.SupportedScaffolds()
	if strings.TrimSpace(*scaffold) != "" {
		one, parseErr := parseScaffold(*scaffold)
		if parseErr != nil {
			return parseErr
		}
		selected = []ambient.Scaffold{one}
	}

	states := make([]ambient.AttachmentState, 0, len(selected))
	anyWorking := false
	for _, candidate := range selected {
		state, inspectErr := ambient.Inspect(candidate, home, root)
		if inspectErr != nil {
			return inspectErr
		}
		anyWorking = anyWorking || state.Working
		states = append(states, state)
	}
	if len(states) == 1 {
		return writeJSON(stdout, states[0])
	}
	return writeJSON(stdout, map[string]any{"any_working": anyWorking, "scaffolds": states})
}

func runDisconnect(args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("disconnect", flag.ContinueOnError)
	set.SetOutput(stderr)
	scaffold := set.String("scaffold", "", "codex or claude-code")
	repository := set.String("repo", ".", "repository whose registration to remove")
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
	// Removal spans every scope this scaffold's registration may occupy, including
	// the one an earlier arrangement wrote: a registration nobody removes keeps
	// firing, and this command is the migration path off that arrangement.
	removed, err := ambient.Disconnect(selected, home, root)
	if err != nil {
		return err
	}
	return writeJSON(stdout, map[string]any{
		"scaffold":   selected,
		"repository": root,
		"removed":    removed,
	})
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
	case "stop", "sessionend":
		// Both names mean the same thing to retention: the session is over. The
		// first scaffold signals it as Stop; the second fires Stop once per turn
		// and signals the end of a session as SessionEnd, which is the event its
		// registration names. Handling only one of them would make retention
		// silently never fire on the other scaffold while every diagnosis reports
		// the attachment as active.
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
