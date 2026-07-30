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

	// TrustNotRequired means this scaffold was observed running a registered hook
	// with no approval step, so there is no trust record to wait for. It is a
	// separate state rather than "recorded": nothing was granted, and claiming a
	// record exists would be an invention in the opposite direction.
	TrustNotRequired TrustState = "not-required"
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
		// A live session showed a registered hook running with no approval step,
		// so there is no surface to send anyone to. `/hooks` is named as what it
		// is — a way to look — because pointing at it as a gate would invent one.
		return "nothing to approve; run /hooks inside Claude Code if you want to see what is registered"
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
	const check = " Run `gr doctor` to check."
	switch TrustEvidenceOf(scaffold) {
	case TrustGateObserved:
		return "Hooks are registered but not yet active: " + string(scaffold) +
			" requires you to review and trust them first — " + TrustSurface(scaffold) +
			". Until then Goalrail does nothing in your sessions, and trust applies " +
			"from the next session onward." + check
	case TrustGateObservedAbsent:
		// Observed rather than assumed: a live session ran a registered hook with
		// no approval step. Inventing an approval screen here would send the user
		// hunting for one that does not exist.
		return "Hooks are registered and need no approval step on " + string(scaffold) +
			": a live session confirmed a registered hook runs without one. The " +
			"attachment applies from your next session onward." + check
	case TrustGateDocumentedAbsent:
		return "Hooks are registered. " + string(scaffold) + " documents no approval " +
			"step for hooks configured in a settings file, though that has not been " +
			"observed here, so if Goalrail stays silent in your sessions that is the " +
			"first thing to check." + check
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
	const check = " Run `gr doctor` to check."
	replaced := "The registration named a different Goalrail executable"
	if previous != "" {
		replaced += " (" + previous + ")"
	}
	replaced += " and was replaced. "
	if TrustEvidenceOf(scaffold) == TrustGateObservedAbsent {
		// No gate was observed on this scaffold, so a replaced command needs no
		// second review and saying otherwise would invent an obstacle.
		return replaced + "This scaffold was observed running a registered hook " +
			"without an approval step, so the replacement applies from your next " +
			"session onward." + check
	}
	switch scaffold {
	case ScaffoldCodex:
		// Deliberately does not send the user to `gr health`. The scaffold's trust
		// record survives a repair and still reads as present, so health would
		// report this very attachment as working. Pointing at a surface that
		// contradicts this notice would be worse than saying nothing.
		return replaced + "Changing a hook command changes its definition, so the " +
			"review you gave the previous one no longer applies — " + TrustSurface(scaffold) +
			". Until then Goalrail does nothing in your sessions, and trust applies " +
			"from the next session onward. `gr health` cannot confirm this step: it " +
			"will report the attachment as working, because the record it reads was " +
			"written for the command that has just been replaced."
	default:
		// Here the health pointer is accurate: no trust record has been observed
		// for this scaffold, so health reports the trust state as undetermined
		// rather than claiming the attachment works.
		return replaced + "Some scaffolds ask you to review a changed command before " +
			"it runs; whether " + string(scaffold) + " does has not been verified here — " +
			TrustSurface(scaffold) + ", and if Goalrail stays silent in your sessions " +
			"that is the first thing to check." + check
	}
}

// SupersededEventNotice is the disclosure a registration corrected for its event
// carries.
//
// It is not RepairNotice: that one states what a replaced executable path means,
// and saying "the registration named a different Goalrail executable" when the
// executable never changed would be a false statement about the user's
// configuration. What changed is which event fires, and the consequence is the one
// the user might otherwise have noticed as a growing pile of records.
func SupersededEventNotice(scaffold Scaffold, replaced []string) string {
	if len(replaced) == 0 {
		return ""
	}
	return "The registration was moved off " + strings.Join(replaced, ", ") +
		", which fires once per turn on " + string(scaffold) +
		" rather than once when a session ends. Until now a question left at the " +
		"reserved path was retained again on every turn; one question is now " +
		"retained once. Run `gr doctor` to check."
}

// Inspect reports attachment health for one scaffold and one repository.
// It never writes anything.
//
// It reads the scope that scaffold's registration belongs to. Reading user
// configuration for a scaffold that registers per repository would report a
// working attachment as absent — a confident wrong answer, which is worse than
// none.
func Inspect(scaffold Scaffold, home, repositoryRoot string) (AttachmentState, error) {
	target, err := RegistrationTarget(scaffold, home, repositoryRoot)
	if err != nil {
		return AttachmentState{}, err
	}
	configPath := target.Path
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
		state.NextAction = registrationAction(scaffold)
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
				"; " + registrationAction(scaffold) + " from the current binary"
			return state, nil
		}
		state.Working = true
	}
	return state, nil
}

// registrationAction names the command that registers this scaffold. It follows
// the scope, because a remedy naming the wrong command is advice that cannot be
// followed.
func registrationAction(scaffold Scaffold) string {
	if scope, err := ScopeOf(scaffold); err == nil && scope == ScopeRepository {
		return "run `gr init` in this repository"
	}
	return "run `gr connect --scaffold " + string(scaffold) + " --yes`"
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
		// A live session ran a hook registered in this scaffold's repository-scope
		// settings file with no approval step, so there is no record to wait for.
		// What that session did not capture is recorded as the limit of the claim
		// rather than left to be inferred from silence.
		state.Unverifiable = append(state.Unverifiable,
			"what "+string(scaffold)+" displays at startup for a folder it has not seen before: "+
				"not captured; the hook ran regardless")
		return TrustNotRequired
	default:
		return TrustUnknown
	}
}

// checkRegisteredExecutable confirms the command a registration points at still
// exists and is executable.
//
// It inspects the first registered handler only. Widening it to every handler
// would change what health reports — a configuration hand-edited so one event
// names a live binary and another a dead one would flip from working to broken —
// and this change makes connection able to act on health's existing diagnosis
// rather than altering the diagnosis.
//
// The decoding, by contrast, is shared with connection deliberately. Reading the
// path as raw text mishandled a path containing an apostrophe, which would report
// a present binary as missing; two different answers to the same question in one
// package is not a boundary worth preserving.
func checkRegisteredExecutable(scaffold Scaffold, configPath string) error {
	registered, err := registeredExecutables(scaffold, configPath)
	if err != nil {
		return fmt.Errorf("scaffold configuration is unreadable")
	}
	if len(registered) == 0 {
		return nil
	}
	path := registered[0]
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
