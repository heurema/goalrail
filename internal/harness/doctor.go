package harness

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
)

// nodeExecutable is the runtime the stock OpenSpec CLI needs. Its presence is
// looked up, never executed: `gr` runs no Node, and reporting a version would
// require running one.
const nodeExecutable = "node"

// ToolchainState reports the runtime the stock CLI needs, as a fact rather than a
// fault. The harness stands without it; only validation and archival do not run.
type ToolchainState struct {
	NodePresent bool   `json:"node_present"`
	NodePath    string `json:"node_path,omitempty"`

	// Needed names what the runtime is for, so its absence is legible instead of
	// mysterious.
	Needed string `json:"needed"`

	Note string `json:"note,omitempty"`
}

// AttachmentDiagnosis extends the attachment report with the scope questions this
// arrangement introduced.
type AttachmentDiagnosis struct {
	ambient.AttachmentState

	Scope ambient.Scope `json:"scope"`

	// SupersededScope reports a registration surviving in the scope this
	// arrangement no longer writes. It duplicates the current one, so every hook
	// fires twice.
	SupersededScope string `json:"superseded_scope,omitempty"`

	// SupersededEvents reports handlers of ours on events this arrangement no
	// longer writes. On the scaffold that moved, that event fires once per turn,
	// so one question is retained again on every turn.
	SupersededEvents []string `json:"superseded_events,omitempty"`
}

// Diagnosis is one report covering everything a working harness requires.
type Diagnosis struct {
	Repository  string `json:"repository"`
	Version     string `json:"version"`
	Initialized bool   `json:"initialized"`

	Overlay OverlayState `json:"overlay"`

	// Schema is what the OpenSpec configuration names, and SchemaPresent whether
	// there is a configuration at all.
	Schema        string `json:"schema,omitempty"`
	SchemaPresent bool   `json:"schema_present"`

	Attachments   []AttachmentDiagnosis `json:"attachments"`
	Toolchain     ToolchainState        `json:"toolchain"`
	Observability ObservabilityState    `json:"observability"`

	// Invocation is the exact command the repository is driven by, printed so an
	// agent working here does not have to guess it.
	Invocation string `json:"invocation"`

	// Working is the verdict. Optional absences never affect it.
	Working bool `json:"working"`

	// Problems are the states that are not working, each paired with its next
	// action in NextActions at the same index.
	Problems    []string `json:"problems,omitempty"`
	NextActions []string `json:"next_actions,omitempty"`
}

// DiagnoseInput is what a diagnosis needs from its caller, so nothing here reads
// ambient process state directly and every field can be driven by a test.
type DiagnoseInput struct {
	RepositoryRoot string
	Home           string
	StateRoot      string
	Scaffolds      []ambient.Scaffold

	// LookupEnvironment and LookPath are injectable for the same reason.
	LookupEnvironment func(string) string
	LookPath          func(string) (string, error)
}

// Diagnose assembles the report. It never writes anything.
func Diagnose(input DiagnoseInput) (Diagnosis, error) {
	lookPath := input.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	diagnosis := Diagnosis{
		Repository:  input.RepositoryRoot,
		Version:     Version,
		Initialized: ambient.IsInitialized(input.RepositoryRoot),
		Invocation:  PinnedNewChange,
	}

	overlay, err := InspectOverlay(input.RepositoryRoot)
	if err != nil {
		return Diagnosis{}, err
	}
	diagnosis.Overlay = overlay

	schema, present, err := ConfiguredSchema(input.RepositoryRoot)
	if err != nil {
		return Diagnosis{}, err
	}
	diagnosis.Schema, diagnosis.SchemaPresent = schema, present

	scaffolds := input.Scaffolds
	if len(scaffolds) == 0 {
		// Defaulting to one scaffold would hand a user who attached the other a
		// confident, wrong diagnosis.
		scaffolds = ambient.SupportedScaffolds()
	}
	for _, scaffold := range scaffolds {
		attachment, attachErr := diagnoseAttachment(scaffold, input.Home, input.RepositoryRoot)
		if attachErr != nil {
			return Diagnosis{}, attachErr
		}
		diagnosis.Attachments = append(diagnosis.Attachments, attachment)
	}

	nodePath, lookErr := lookPath(nodeExecutable)
	diagnosis.Toolchain = ToolchainState{
		NodePresent: lookErr == nil,
		NodePath:    nodePath,
		Needed:      "validating and archiving changes with the stock OpenSpec CLI",
	}
	if lookErr != nil {
		diagnosis.Toolchain.Note = "the stock OpenSpec CLI needs a Node runtime, which is not on this machine's path; " +
			"the harness itself is unaffected, and validation and archival are what cannot run"
	}

	diagnosis.Observability = InspectObservability(input.StateRoot, input.LookupEnvironment)

	diagnosis.Problems, diagnosis.NextActions = summarize(diagnosis)
	diagnosis.Working = len(diagnosis.Problems) == 0
	return diagnosis, nil
}

func diagnoseAttachment(scaffold ambient.Scaffold, home, repositoryRoot string) (AttachmentDiagnosis, error) {
	scope, err := ambient.ScopeOf(scaffold)
	if err != nil {
		return AttachmentDiagnosis{}, err
	}
	state, err := ambient.Inspect(scaffold, home, repositoryRoot)
	if err != nil {
		return AttachmentDiagnosis{}, err
	}
	diagnosis := AttachmentDiagnosis{AttachmentState: state, Scope: scope}

	if superseded, present := ambient.SupersededTarget(scaffold, home); present {
		occupied, occupiedErr := ambient.HasAnyRegistration(superseded)
		if occupiedErr == nil && occupied {
			diagnosis.SupersededScope = superseded.Path
		}
	}
	target, targetErr := ambient.RegistrationTarget(scaffold, home, repositoryRoot)
	if targetErr == nil {
		events, eventsErr := ambient.SupersededRegistrations(target)
		if eventsErr == nil && len(events) > 0 {
			diagnosis.SupersededEvents = events
		}
	}
	return diagnosis, nil
}

// summarize turns the report into problems and the action each one needs, in a
// stable order. Every next action names a command this tool accepts: advice that
// cannot be followed is worse than none.
func summarize(diagnosis Diagnosis) (problems []string, actions []string) {
	add := func(problem, action string) {
		problems = append(problems, problem)
		actions = append(actions, action)
	}

	if !diagnosis.Initialized {
		add("this repository is not initialized", "run `gr init`")
	}
	switch {
	case !diagnosis.Overlay.Present:
		add("the harness overlay is not installed", "run `gr init`")
	case !diagnosis.Overlay.Complete:
		add("the harness overlay is incomplete", "run `gr init`")
	}
	for _, finding := range diagnosis.Overlay.Drifted() {
		switch finding.State {
		case FileEdited:
			add(fmt.Sprintf("%s differs from the canon this binary carries", finding.Path),
				"keep the edit, or run `gr update --discard-local-edits` to restore it")
		case FileBehind:
			add(fmt.Sprintf("%s is behind the canon this binary carries", finding.Path),
				"run `gr update`")
		case FileSuperseded:
			add(fmt.Sprintf("%s is overlay content the canon no longer carries", finding.Path),
				"delete it when you are ready; Goalrail does not remove repository content")
		}
	}
	if diagnosis.SchemaPresent && diagnosis.Schema != SchemaName {
		if diagnosis.Schema == "" {
			// A malformed or empty key is not a foreign schema: the stock CLI reads
			// no schema from it at all, and saying it "names" something would be
			// a diagnosis about a claim the file does not make.
			add("the OpenSpec configuration names no schema the stock CLI can read",
				"run `gr init`")
		} else {
			add(fmt.Sprintf("the OpenSpec configuration names %q rather than %q", diagnosis.Schema, SchemaName),
				"run `gr init` and confirm the schema switch")
		}
	}
	if !diagnosis.SchemaPresent {
		add("there is no OpenSpec configuration", "run `gr init`")
	}

	anyAttached := false
	for _, attachment := range diagnosis.Attachments {
		if attachment.Working {
			anyAttached = true
		}
		if attachment.SupersededScope != "" {
			add(fmt.Sprintf("%s carries a registration at %s, which this arrangement replaced; every hook fires twice",
				attachment.Scaffold, attachment.SupersededScope),
				fmt.Sprintf("run `gr disconnect --scaffold %s` to remove it, then `gr init` here", attachment.Scaffold))
		}
		if len(attachment.SupersededEvents) > 0 {
			add(fmt.Sprintf("%s is registered against %s, which fires once per turn rather than once per session, so one question is retained again on every turn",
				attachment.Scaffold, strings.Join(attachment.SupersededEvents, ", ")),
				"run `gr init` here to replace it")
		}
	}
	if !anyAttached {
		var pending []string
		for _, attachment := range diagnosis.Attachments {
			if attachment.NextAction != "" {
				pending = append(pending, fmt.Sprintf("%s: %s", attachment.Scaffold, attachment.NextAction))
			}
		}
		sort.Strings(pending)
		add("no scaffold attachment is active in this repository", strings.Join(pending, "; "))
	}
	return problems, actions
}

// Describe renders the diagnosis for a person. The machine-readable form is the
// struct itself.
func Describe(diagnosis Diagnosis) string {
	var report strings.Builder
	fmt.Fprintf(&report, "goalrail %s — %s\n", diagnosis.Version, diagnosis.Repository)

	verdict := "not working"
	if diagnosis.Working {
		verdict = "working"
	}
	fmt.Fprintf(&report, "harness: %s\n", verdict)

	fmt.Fprintf(&report, "overlay: ")
	switch {
	case !diagnosis.Overlay.Present:
		report.WriteString("not installed\n")
	case diagnosis.Overlay.Current:
		fmt.Fprintf(&report, "current (%s)\n", diagnosis.Overlay.Canon)
	default:
		fmt.Fprintf(&report, "%d file(s) differ from %s\n", len(diagnosis.Overlay.Drifted()), diagnosis.Overlay.Canon)
		for _, finding := range diagnosis.Overlay.Drifted() {
			fmt.Fprintf(&report, "  %-12s %s\n", finding.State, finding.Path)
		}
	}

	for _, attachment := range diagnosis.Attachments {
		state := "not active"
		if attachment.Working {
			state = "active"
		}
		fmt.Fprintf(&report, "%s: %s (%s scope)\n", attachment.Scaffold, state, attachment.Scope)
	}

	if diagnosis.Toolchain.NodePresent {
		fmt.Fprintf(&report, "openspec cli: available (needed for %s)\n", diagnosis.Toolchain.Needed)
	} else {
		fmt.Fprintf(&report, "openspec cli: %s\n", diagnosis.Toolchain.Note)
	}

	if diagnosis.Observability.Configured {
		fmt.Fprintf(&report, "observability: configured at %s\n", diagnosis.Observability.Endpoint)
	} else {
		fmt.Fprintf(&report, "%s\n", diagnosis.Observability.Note)
	}

	fmt.Fprintf(&report, "invocation: %s\n", diagnosis.Invocation)

	for index, problem := range diagnosis.Problems {
		fmt.Fprintf(&report, "\nproblem: %s\n", problem)
		if index < len(diagnosis.NextActions) && diagnosis.NextActions[index] != "" {
			fmt.Fprintf(&report, "next:    %s\n", diagnosis.NextActions[index])
		}
	}
	return report.String()
}
