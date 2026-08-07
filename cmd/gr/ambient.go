package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/adapters/codex"
	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/localrun"
	projectstate "github.com/heurema/goalrail/internal/project"
)

// hookEnvironmentSession is the session identifier a scaffold exposes to its
// hooks. It is read opportunistically: a missing value costs attribution
// detail, never the question itself.
const hookEnvironmentSession = "CODEX_SESSION_ID"

// initReport keeps the portable project initialization report intact and adds
// the one bounded local receipt needed when an existing custom OpenSpec schema
// is explicitly adopted. The optional marker records that adoption advisory;
// managed-project identity never depends on it.
type initReport struct {
	projectstate.InitializeReport
	Adoption    *adoptionReport   `json:"adoption,omitempty"`
	Attachments []attachmentWrite `json:"attachments,omitempty"`
	Notices     []string          `json:"notices,omitempty"`
}

// attachmentWrite is the disclosure initialization owes for a local mutation it
// performed: the exact path, the events, whether anything changed, what was done
// to keep the path out of a commit, and what the scaffold still requires of the
// user.
type attachmentWrite struct {
	Scaffold      string   `json:"scaffold"`
	Scope         string   `json:"scope"`
	Path          string   `json:"path"`
	Events        []string `json:"events,omitempty"`
	Action        string   `json:"action"`
	IgnoreEntries []string `json:"ignore_entries,omitempty"`
	Trust         string   `json:"trust,omitempty"`
	Reason        string   `json:"reason,omitempty"`

	// Superseded and ReplacedExecutable describe what a repair replaced: a
	// handler of ours on an event this arrangement no longer writes, and a
	// registration naming an executable other than this one.
	Superseded         []string `json:"superseded_events,omitempty"`
	ReplacedExecutable string   `json:"replaced_executable,omitempty"`
}

func runInit(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("init", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "repository to initialize")
	scaffold := set.String("scaffold", "", "codex or claude-code; omit to detect")
	confirmSchema := set.Bool("confirm-schema-switch", false,
		"switch an OpenSpec configuration that names another custom schema")
	fixIgnore := set.Bool("fix-gitignore", false,
		"deprecated; the only ignore rule initialization writes is the one that keeps a scaffold registration out of a commit")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("init accepts no positional arguments")
	}
	if *fixIgnore {
		return fmt.Errorf("--fix-gitignore is not part of v1 project initialization; the only ignore rule it writes is the clone-local one that keeps a scaffold registration out of a commit")
	}
	selected := ""
	if strings.TrimSpace(*scaffold) != "" {
		one, parseErr := parseScaffold(*scaffold)
		if parseErr != nil {
			return parseErr
		}
		selected = string(one)
	}
	report, err := projectstate.Initialize(ctx, *repository, projectstate.InitializeOptions{
		ConfirmForeignSchema: *confirmSchema,
		RequestedScaffold:    selected,
	})
	if err != nil {
		return statedRepositoryCondition(err, *repository)
	}

	response := initReport{InitializeReport: report}
	response.Attachments = registerRepositoryScaffolds(report.Repository, selected, initHome(), initExecutable())
	adoptionReport, adoption := buildAdoptionReport(report.Repository, report.Config)
	response.Adoption = adoptionReport
	if adoption != nil {
		markerTime := time.Now().UTC()
		adoption.AdoptedAt = markerTime
		response.Adoption.AdoptedAt = markerTime
		marker, _, markerErr := ambient.InitializeWithAdoption(
			report.Repository,
			func() time.Time { return markerTime },
			adoption,
		)
		if markerErr != nil && !errors.Is(markerErr, ambient.ErrAdoptionNotRecorded) {
			return markerErr
		}
		if markerErr != nil {
			response.Adoption.Notices = append(
				response.Adoption.Notices,
				"the adoption record could not be written; the existing marker was kept: "+markerErr.Error(),
			)
		} else if marker.Adoption != nil {
			response.Adoption.AdoptedAt = marker.Adoption.AdoptedAt
		}
	}
	return writeJSON(stdout, response)
}

// registerRepositoryScaffolds writes the registration for every supported
// scaffold whose settings layer lives inside the repository.
//
// Initialization is the consented command for that scope, and it is the one a
// health report names as the remedy, so a report that prescribes it and an
// initialization that writes nothing leave the user following correct advice
// with no remaining move.
//
// It acts for a scaffold present on this machine or named explicitly, and for
// nothing else: a machine with no supported scaffold has its configuration left
// alone rather than guessed at.
func registerRepositoryScaffolds(repositoryRoot, requested, home, executable string) []attachmentWrite {
	if executable == "" {
		return nil
	}
	// The home directory answers only "does this machine carry the scaffold".
	// A repository-scope registration never reads it, so an unresolvable home
	// must not silently swallow a scaffold the caller named outright.
	var detected []ambient.Scaffold
	if home != "" {
		detected = ambient.DetectScaffolds(home)
	}
	var writes []attachmentWrite
	for _, scaffold := range ambient.SupportedScaffolds() {
		if scope, scopeErr := ambient.ScopeOf(scaffold); scopeErr != nil || scope != ambient.ScopeRepository {
			continue
		}
		if requested != "" {
			if requested != string(scaffold) {
				continue
			}
		} else if !slices.Contains(detected, scaffold) {
			continue
		}
		writes = append(writes, registerRepositoryScaffold(scaffold, repositoryRoot, executable))
	}
	return writes
}

// initHome and initExecutable are the two facts this needs from the process,
// resolved here so the registration itself can be exercised against a temporary
// home and a chosen executable.
func initHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func initExecutable() string {
	executable, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		return resolved
	}
	return executable
}

func registerRepositoryScaffold(scaffold ambient.Scaffold, repositoryRoot, executable string) attachmentWrite {
	write := attachmentWrite{Scaffold: string(scaffold), Scope: string(ambient.ScopeRepository)}
	target, err := ambient.RegistrationTarget(scaffold, "", repositoryRoot)
	if err != nil {
		write.Action, write.Reason = "refused", err.Error()
		return write
	}
	write.Path = target.Path

	// Made unshareable first, and through this clone's own rule rather than one a
	// commit could carry: a registration a teammate could receive would install
	// this command in their sessions on the strength of one person's decision.
	relative, relErr := filepath.Rel(repositoryRoot, target.Path)
	if relErr != nil {
		write.Action, write.Reason = "refused", relErr.Error()
		return write
	}
	entry := filepath.ToSlash(relative)
	if added, ignoreErr := ambient.AddCloneIgnoreEntries(repositoryRoot, []string{entry}); ignoreErr == nil {
		write.IgnoreEntries = added
	} else {
		write.Reason = ignoreErr.Error()
	}
	if ignored, ignoreErr := ambient.IgnoreState(repositoryRoot, entry); ignoreErr != nil || !ignored {
		write.Action = "refused"
		write.Reason = "this path is not ignored by version control, so a commit could hand the registration to a teammate"
		if source := ambient.IgnoreSource(repositoryRoot, entry); source != "" {
			write.Reason += "; the deciding rule is in " + source
		}
		return write
	}

	plan, err := ambient.PlanRegistration(target, executable)
	if err != nil {
		write.Action, write.Reason = "refused", err.Error()
		return write
	}
	write.Events = plan.Events
	write.Superseded = plan.SupersededPresent
	changed, err := ambient.Connect(plan)
	if err != nil {
		// The notice says the hooks are registered and apply from the next
		// session, which is a success claim. A refusal must not carry it.
		write.Action, write.Reason = "refused", err.Error()
		return write
	}
	write.Trust = ambient.ConnectionNotice(scaffold)
	switch {
	case changed && plan.Repair:
		// A repair changed the hook definition, so whatever review the scaffold
		// applies to it applies again. Reporting it as an ordinary registration
		// would hide that existing behaviour was replaced.
		write.Action, write.ReplacedExecutable = "repaired", plan.RegisteredExecutable
	case changed:
		write.Action = "registered"
	default:
		write.Action = "unchanged"
	}
	return write
}

func runMigrate(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("migrate", flag.ContinueOnError)
	set.SetOutput(stderr)
	repository := set.String("repo", ".", "v0.1.8 repository to migrate")
	scaffold := set.String("scaffold", "", "codex or claude-code; records the requested later setup only")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("migrate accepts no positional arguments")
	}
	selected := ""
	if strings.TrimSpace(*scaffold) != "" {
		one, err := parseScaffold(*scaffold)
		if err != nil {
			return err
		}
		selected = string(one)
	}
	report, err := projectstate.Migrate(ctx, *repository, projectstate.InitializeOptions{RequestedScaffold: selected})
	if err != nil {
		return statedRepositoryCondition(err, *repository)
	}
	return writeJSON(stdout, report)
}

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

	inspection, inspectErr := projectstate.Inspect(context.Background(), root)
	if inspectErr != nil && !errors.Is(inspectErr, projectstate.ErrNotRepository) {
		return inspectErr
	}
	if inspectErr != nil || inspection.State != projectstate.ClaimManaged {
		claimState := string(projectstate.ClaimUnmanaged)
		claimReason, claimDetail := "", "the directory is not a declared Goalrail project"
		nextAction := ""
		if inspectErr == nil && inspection.State == projectstate.ClaimDeclaredInvalid {
			claimState = string(inspection.State)
			claimReason, claimDetail = string(inspection.Reason), inspection.Detail
			nextAction = "restore `.goalrail/project.json` from a trusted Git revision or perform an explicit supported migration"
		}
		states := make([]ambient.AttachmentState, 0, len(selected))
		for _, candidate := range selected {
			states = append(states, ambient.AttachmentState{
				ClaimState: claimState, ClaimReason: claimReason, ClaimDetail: claimDetail,
				Scaffold: candidate, Trust: ambient.TrustUnknown,
				EnforcementScope: "local_advisory_only", Repository: root, NextAction: nextAction,
			})
		}
		return writeAttachmentHealth(stdout, states)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	states := make([]ambient.AttachmentState, 0, len(selected))
	for _, candidate := range selected {
		state, inspectErr := ambient.InspectForProject(candidate, home, inspection.WorktreeRoot, string(inspection.Declaration.ProjectID))
		if inspectErr != nil {
			return inspectErr
		}
		states = append(states, state)
	}
	return writeAttachmentHealth(stdout, states)
}

func writeAttachmentHealth(stdout io.Writer, states []ambient.AttachmentState) error {
	if len(states) == 1 {
		return writeJSON(stdout, states[0])
	}
	anyWorking := false
	for _, state := range states {
		anyWorking = anyWorking || state.Working
	}
	claimState, projectID := "", ""
	if len(states) > 0 {
		claimState, projectID = states[0].ClaimState, states[0].ProjectID
	}
	return writeJSON(stdout, map[string]any{
		"claim_state": claimState, "project_id": projectID,
		"any_working": anyWorking, "scaffolds": states,
	})
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
	// The committed declaration check comes before the payload is read, not merely before it
	// is acted on. A hook payload can carry prompts, transcript paths, and
	// authorization fields; reading one from a repository the user never
	// declared would observe unrelated work regardless of what happened next.
	root, err := os.Getwd()
	if err != nil {
		return nil
	}
	inspection, err := projectstate.Inspect(context.Background(), root)
	if err != nil || inspection.State != projectstate.ClaimManaged {
		return nil
	}
	root = inspection.WorktreeRoot
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
		announcement, _, startErr := ambient.StartDeclaredProjectSession(store, root, time.Now)
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
		if _, stopErr := ambient.StopDeclaredProjectSession(
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
		return "", fmt.Errorf("--scaffold must be codex or claude-code")
	}
}

// nowUTC is the ambient clock. It exists so tests can construct markers
// through the same call the commands use.
func nowUTC() time.Time { return time.Now().UTC() }
