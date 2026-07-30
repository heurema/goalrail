package ambient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// TrustState is what Goalrail can observe about the scaffold's own review step.
//
// A registered hook does not run until the user reviews and trusts its exact
// definition through the scaffold's surface. Goalrail only reads this: writing,
// computing, or reproducing a trust record would manufacture a standing consent
// to run a command in every session the user starts, which belongs to the user
// alone — and would depend on private implementation details besides.
type TrustState string

const (
	// TrustRecorded means a trust record exists for every registered hook.
	// It deliberately is not called "granted": the scaffold records trust
	// against the definition's current form, and confirming that the stored
	// value still matches would require reproducing a private hash, which this
	// capability forbids. So the record's presence is reported as what it is —
	// evidence, not proof.
	TrustRecorded TrustState = "recorded"
	TrustPending  TrustState = "pending"
	TrustUnknown  TrustState = "unknown"
)

// AttachmentState answers "is my attachment working, and if not, why".
//
// Three different causes produce "not working", and each has a different next
// action, so the report distinguishes them rather than returning a verdict.
// Pending trust matters most: it is the state that otherwise looks exactly like
// a broken installation.
type AttachmentState struct {
	Scaffold    Scaffold   `json:"scaffold"`
	Connected   bool       `json:"connected"`
	Initialized bool       `json:"initialized"`
	Trust       TrustState `json:"trust"`
	Working     bool       `json:"working"`
	NextAction  string     `json:"next_action,omitempty"`

	// Unverifiable names what this report could not establish. A health command
	// that hides its own blind spots is worse than none, because a green result
	// then means "nothing detected" while reading as "everything works".
	Unverifiable []string `json:"unverifiable,omitempty"`

	// ConfigError reports a scaffold configuration that cannot be read or
	// parsed. Without it the report would say "not connected" and recommend
	// connecting, which fails on the same unreadable file.
	ConfigError string `json:"config_error,omitempty"`

	ConfigPath string `json:"config_path"`
	Repository string `json:"repository"`
}

// TrustSurface names where the user performs the scaffold's review step. It is
// part of the disclosure: telling someone a step is required without saying
// where to do it leaves them exactly as stuck.
func TrustSurface(scaffold Scaffold) string {
	switch scaffold {
	case ScaffoldCodex:
		return "run /hooks inside Codex and trust the Goalrail hooks"
	case ScaffoldClaudeCode:
		return "review the Goalrail hooks in Claude Code's hook settings if it asks for approval"
	default:
		return "review and trust the Goalrail hooks in your scaffold"
	}
}

// ConnectionNotice is the disclosure connection prints after registering.
//
// The failure it prevents is specific: connect, work, observe nothing, conclude
// the product is broken. Documentation does not reach a user in that moment;
// this line does, because they are looking at it already.
//
// The wording differs per scaffold because the certainty differs. For a
// scaffold whose trust gate was observed live, the requirement is stated. For
// one where it was not, claiming a mandatory approval step would send the user
// hunting for a screen that may not exist — inventing an obstacle is its own
// kind of misinformation.
func ConnectionNotice(scaffold Scaffold) string {
	const check = " Run `gr health` to check."
	switch scaffold {
	case ScaffoldCodex:
		return "Hooks are registered but not yet active: Codex requires you to review " +
			"and trust them first — " + TrustSurface(scaffold) +
			". Until then Goalrail does nothing in your sessions, and trust applies " +
			"from the next session onward." + check
	default:
		return "Hooks are registered. Some scaffolds require you to review and trust " +
			"registered commands before they run; whether " + string(scaffold) +
			" does has not been verified here — " + TrustSurface(scaffold) +
			", and if Goalrail stays silent in your sessions that is the first thing " +
			"to check." + check
	}
}

// RepairNotice is the disclosure a replaced registration requires.
//
// The scaffold records trust against the hook definition's current form, so
// replacing a command discards whatever review the previous one was given. The
// moment of repair is the only place this can be stated with certainty: Goalrail
// made the change itself, whereas reading the stored record can never establish
// it, because reproducing the form the record is kept against is prohibited.
// Left unsaid here it is unsayable anywhere, and the silent stale path would
// simply become a silent untrusted hook — the same symptom one layer down.
//
// The wording follows the same per-scaffold discipline as ConnectionNotice:
// asserted where the trust gate was observed live, marked unverified where it was
// not, because inventing a mandatory approval step sends the user hunting for a
// screen that may not exist.
func RepairNotice(scaffold Scaffold, previous string) string {
	const check = " Run `gr health` to check."
	replaced := "The registration named a different Goalrail executable"
	if previous != "" {
		replaced += " (" + previous + ")"
	}
	replaced += " and was replaced. "
	switch scaffold {
	case ScaffoldCodex:
		return replaced + "Changing a hook command changes its definition, so the " +
			"review you gave the previous one no longer applies — " + TrustSurface(scaffold) +
			". Until then Goalrail does nothing in your sessions, and trust applies " +
			"from the next session onward." + check
	default:
		return replaced + "Some scaffolds ask you to review a changed command before " +
			"it runs; whether " + string(scaffold) + " does has not been verified here — " +
			TrustSurface(scaffold) + ", and if Goalrail stays silent in your sessions " +
			"that is the first thing to check." + check
	}
}

// Inspect reports attachment health for one scaffold and one repository.
// It never writes anything.
func Inspect(scaffold Scaffold, home, repositoryRoot string) (AttachmentState, error) {
	configPath, err := ConfigPath(scaffold, home)
	if err != nil {
		return AttachmentState{}, err
	}
	state := AttachmentState{
		Scaffold:    scaffold,
		Initialized: IsInitialized(repositoryRoot),
		Trust:       TrustUnknown,
		ConfigPath:  configPath,
		Repository:  repositoryRoot,
	}

	connected, connectErr := isConnected(scaffold, configPath)
	if connectErr != nil {
		// An unreadable or malformed configuration is its own failure. Reporting
		// "not connected" here would recommend a connection that reads the same
		// file and fails identically.
		state.ConfigError = connectErr.Error()
		state.NextAction = "repair the scaffold configuration at " + configPath +
			"; Goalrail cannot read it"
		return state, nil
	}
	state.Connected = connected
	if connected {
		state.Trust = inspectTrust(scaffold, configPath, &state)
	}

	switch {
	case !state.Connected:
		state.NextAction = "run `gr connect --scaffold " + string(scaffold) + " --yes`"
	case !state.Initialized:
		state.NextAction = "run `gr init` in this repository"
	case state.Trust == TrustPending:
		state.NextAction = TrustSurface(scaffold)
	case state.Trust == TrustUnknown:
		// Reporting what cannot be determined beats guessing that it is fine.
		state.NextAction = "trust state could not be determined; " + TrustSurface(scaffold) +
			" if Goalrail stays silent in your sessions"
	default:
		if executableErr := checkRegisteredExecutable(scaffold, configPath); executableErr != nil {
			// A registration pointing at a missing binary satisfies every
			// configuration check and still cannot run — common after a locally
			// built binary is moved or an old install is removed.
			state.NextAction = executableErr.Error() +
				"; re-run `gr connect --scaffold " + string(scaffold) + " --yes` from the current binary"
			return state, nil
		}
		state.Working = true
	}
	return state, nil
}

// inspectTrust reads the scaffold's own record of which hooks the user has
// reviewed. Read-only and best-effort: an unreadable or unfamiliar shape yields
// TrustUnknown rather than an optimistic answer.
func inspectTrust(scaffold Scaffold, configPath string, state *AttachmentState) TrustState {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return TrustPending
		}
		return TrustUnknown
	}
	switch scaffold {
	case ScaffoldCodex:
		content := string(raw)
		if !strings.Contains(content, "[hooks.state") {
			return TrustPending
		}
		// Connection registers both events, and an untrusted stop hook means
		// questions are never retained while everything else looks healthy.
		// Every registered event must therefore carry a record.
		for _, event := range []string{"session_start", "stop"} {
			if !strings.Contains(content, configPath+":"+event) {
				return TrustPending
			}
		}
		state.Unverifiable = append(state.Unverifiable,
			"whether each trust record still matches the current hook definition: "+
				"the scaffold records trust against a private hash, and reproducing it is prohibited")
		return TrustRecorded
	case ScaffoldClaudeCode:
		var settings map[string]any
		if json.Unmarshal(raw, &settings) != nil {
			return TrustUnknown
		}
		// No trust record has been observed for this scaffold, so claiming
		// either answer would be invention.
		state.Unverifiable = append(state.Unverifiable,
			"whether "+string(scaffold)+" gates hooks behind review at all: not observed live")
		return TrustUnknown
	default:
		return TrustUnknown
	}
}

// checkRegisteredExecutable confirms the command a registration points at still
// exists and is executable.
func checkRegisteredExecutable(scaffold Scaffold, configPath string) error {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("scaffold configuration is unreadable")
	}
	path := registeredExecutable(string(raw))
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("the registered Goalrail executable is missing at %s", path)
	}
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("the registered Goalrail executable at %s is not runnable", path)
	}
	return nil
}

// registeredExecutable extracts the binary path from the first managed command
// in the given content.
//
// It inspects only the first deliberately. Widening it to every registered
// handler would change what health reports — a configuration hand-edited so one
// event names a live binary and another a dead one would flip from working to
// broken — and this change makes connection able to act on health's existing
// diagnosis rather than altering the diagnosis.
func registeredExecutable(content string) string {
	index := strings.Index(content, managedMarker)
	if index < 0 {
		return ""
	}
	return executableBefore(content[:index])
}

// Describe renders one line per state for a human reading terminal output.
func Describe(state AttachmentState) string {
	if state.Working {
		return fmt.Sprintf("%s: attached and active in %s", state.Scaffold, state.Repository)
	}
	return fmt.Sprintf(
		"%s: not active — connected=%t initialized=%t trust=%s\nnext: %s",
		state.Scaffold, state.Connected, state.Initialized, state.Trust, state.NextAction,
	)
}
