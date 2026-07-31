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

	// The two writes are reported apart because they are not the same kind of
	// thing. Ignore names entries added to the rules the repository shares, which
	// only the explicit flag produces and which a commit carries to everyone.
	// CloneIgnore names entries added to this clone's own rule, which stays here.
	Ignore          []string `json:"ignore_entries_added,omitempty"`
	CloneIgnore     []string `json:"clone_ignore_entries_added,omitempty"`
	CloneIgnoreFile string   `json:"clone_ignore_file,omitempty"`

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
		"add the entries to the ignore rules the repository shares, for a repository whose own rules override this clone's")
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
	if err := refuseWhereThereIsNoWorkTree(root); err != nil {
		return err
	}
	// An explicitly named scaffold is validated before the first write: a typo in
	// the flag must produce a usage error against an untouched repository, not a
	// half-installed harness followed by one.
	var explicit []ambient.Scaffold
	if strings.TrimSpace(*scaffold) != "" {
		one, parseErr := parseScaffold(*scaffold)
		if parseErr != nil {
			return parseErr
		}
		explicit = []ambient.Scaffold{one}
	}
	// The home directory matters only for detection and user-scope reporting; a
	// repository-scope installation must not be blocked by its absence.
	home, homeErr := os.UserHomeDir()
	homeKnown := homeErr == nil

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

	// Making the registration path unshareable is part of registering, not a
	// separate thing to ask the user for: this rule belongs to the clone alone,
	// no commit can carry it, and it is nobody else's to agree to. It is written
	// before the scaffold is selected so registration reads the state it produced.
	//
	// A failure here is never fatal. Not having made a path unshareable is a
	// condition the registration already knows how to refuse, and turning it into
	// an aborted command would leave the overlay and the marker on disk with no
	// report at all.
	cloneIgnore, ignoreErr := ambient.AddCloneIgnoreEntries(root, ambient.IgnoreEntries())
	if ignoreErr != nil {
		report.Notices = append(report.Notices,
			"this clone's own ignore rule could not be written: "+ignoreErr.Error())
	}
	report.CloneIgnore = cloneIgnore
	if len(cloneIgnore) > 0 {
		if path, _ := ambient.CloneIgnoreTarget(root); path != "" {
			report.CloneIgnoreFile = path
		}
	}

	// The flag still writes the rules the repository shares, which is repository
	// content exactly as the overlay is — and the only remedy where one of those
	// rules overrides the clone's own.
	if *fixIgnore {
		added, sharedErr := ambient.AddIgnoreEntries(root, ambient.IgnoreEntries())
		if sharedErr != nil {
			return sharedErr
		}
		report.Ignore = added
	}

	selected := explicit
	if selected == nil {
		if homeKnown {
			selected = ambient.DetectScaffolds(home)
		} else {
			report.Notices = append(report.Notices,
				"the home directory could not be resolved, so no scaffold was detected; "+
					"re-run with --scaffold <name> to register one")
		}
	}
	if err := registerSelected(&report, selected, home, homeKnown, root); err != nil {
		return err
	}

	// The marker is per clone, so committing it would make one user's
	// initialization a shared repository fact. Unlike the registration this is not
	// fatal: refusing to install the harness over an ignore rule would be a
	// disproportionate response to a recoverable condition.
	if ignored, markerErr := ambient.IgnoreState(root, ambient.MarkerPath); !ignored {
		// The advice follows the cause, exactly as it does for the registration
		// path: an ignore entry cannot protect a tracked file, and prescribing the
		// flag there would be advice that changes nothing.
		if markerErr != nil {
			report.Notices = append(report.Notices,
				"the marker at "+ambient.MarkerPath+" is committable and an ignore entry cannot "+
					"protect it: "+markerErr.Error())
		} else {
			report.Notices = append(report.Notices,
				"the marker at "+ambient.MarkerPath+" is not ignored, so a commit would make this "+
					"repository initialized for everyone; this clone's own ignore rule could not cover it, "+
					"so `gr init --fix-gitignore` is what adds the entry")
		}
	}

	// Nothing can commit out of a directory that is not a repository, so nothing
	// is refused and no rule is written. But the exposure that earns a refusal
	// elsewhere arrives here the moment the directory becomes one, and it should
	// not arrive in silence.
	if _, ignoreTarget := ambient.CloneIgnoreTarget(root); ignoreTarget == ambient.IgnoreTargetNotARepository {
		report.Notices = append(report.Notices,
			"this directory is not under version control, so nothing here can be committed and no ignore "+
				"rule was written; running `git init` would make the registration at "+
				ambient.RepositorySettingsPath+" and the marker at "+ambient.MarkerPath+
				" committable, and re-running `gr init` then covers both")
	}

	report.Next = append(report.Next,
		"commit the files above; they are yours, and Goalrail does not commit for you")
	return writeJSON(stdout, report)
}

// registerSelected registers the repository-scope scaffold among the candidates
// and names the separate command for any that registers at user scope.
//
// Installing the harness never depends on which agent environment happens to be
// present: where nothing is detected, the overlay is still installed and the
// report says which command attaches later.
// refuseWhereThereIsNoWorkTree stops the harness being written into a
// repository that has nowhere to put it. A bare repository is one such; so is
// the repository's own directory, which is not bare and is equally wrong. The
// question that covers both is whether a work tree exists at all, and it is
// asked before the first write so the directory is left as it was.
func refuseWhereThereIsNoWorkTree(root string) error {
	if !ambient.IsRepository(root) {
		return nil
	}
	if _, hasWorkTree := ambient.WorkTreeRoot(root); hasWorkTree {
		return nil
	}
	return fmt.Errorf("%s is a repository with no work tree, so there is nowhere for the harness to live; "+
		"initialize the clone you work in", root)
}

// whyTheCloneRuleWasNotEnough names the cause the refusal actually has.
//
// The usual one is a rule the repository shares overriding this clone's, and
// saying so unconditionally would be an assertion about the user's repository
// that is false whenever something else was in the way. Version control can name
// the file whose rule decided, so it is asked rather than assumed.
func whyTheCloneRuleWasNotEnough(root string) string {
	source := ambient.IgnoreSource(root, ambient.RepositorySettingsPath)
	switch {
	case source == "":
		return "this clone's own ignore rule did not cover it, and no rule was found that decides otherwise; " +
			"`gr init --fix-gitignore` adds the entry to the rules this repository shares"
	case strings.HasSuffix(filepath.ToSlash(source), ".gitignore"):
		return "this clone's own ignore rule was written and did not take effect, because `" + source +
			"` overrides it — add `" + ambient.RepositorySettingsPath +
			"` to that file, or re-run with --fix-gitignore"
	default:
		return "`" + source + "` decides otherwise and overrides this clone's own ignore rule; " +
			"re-run with --fix-gitignore to add the entry to the rules this repository shares"
	}
}

func registerSelected(
	report *initReport,
	candidates []ambient.Scaffold,
	home string,
	homeKnown bool,
	root string,
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
		registration, err := registerRepositoryScope(candidate, home, homeKnown, root)
		if err != nil {
			return err
		}
		report.Registration = registration
	}
	return nil
}

func registerRepositoryScope(
	scaffold ambient.Scaffold,
	home string,
	homeKnown bool,
	root string,
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
	// path a commit could carry is refused rather than written. By the time this
	// runs, initialization has already tried to make the path unshareable with
	// this clone's own rule, so reaching here means that rule was not enough.
	ignored, ignoreErr := ambient.IgnoreState(root, ambient.RepositorySettingsPath)
	if !ignored {
		// The advice follows the cause. A rule the repository shares can override
		// this clone's, and there the flag is the only remaining move; a tracked
		// file or an unrunnable check is not, and telling the user to re-run with a
		// flag that cannot help would be a remedy that prescribes itself.
		if ignoreErr != nil {
			registration.Refused = ignoreErr.Error() +
				"; --fix-gitignore cannot repair this — untrack the path or make the check runnable first"
		} else {
			registration.Refused = "the registration path is not ignored by version control, so a commit " +
				"would install these hooks in every teammate's sessions; " +
				whyTheCloneRuleWasNotEnough(root)
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

	// Attachment state consults the home directory for the superseded user-scope
	// arrangement; where the home is unknown, the state is left unclaimed rather
	// than computed against a fabricated path.
	if homeKnown {
		if state, inspectErr := ambient.Inspect(scaffold, home, root); inspectErr == nil {
			registration.ActiveNow = state.Working
		}
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
