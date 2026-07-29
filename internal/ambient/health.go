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
	TrustGranted TrustState = "granted"
	TrustPending TrustState = "pending"
	TrustUnknown TrustState = "unknown"
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
	ConfigPath  string     `json:"config_path"`
	Repository  string     `json:"repository"`
}

// TrustSurface names where the user performs the scaffold's review step. It is
// part of the disclosure: telling someone a step is required without saying
// where to do it leaves them exactly as stuck.
func TrustSurface(scaffold Scaffold) string {
	switch scaffold {
	case ScaffoldCodex:
		return "run /hooks inside Codex and trust the Goalrail hooks"
	case ScaffoldClaudeCode:
		return "review the Goalrail hooks in Claude Code's hook settings"
	default:
		return "review and trust the Goalrail hooks in your scaffold"
	}
}

// ConnectionNotice is the disclosure connection prints after registering.
//
// The failure it prevents is specific: connect, work, observe nothing, conclude
// the product is broken. Documentation does not reach a user in that moment;
// this line does, because they are looking at it already.
func ConnectionNotice(scaffold Scaffold) string {
	return "Hooks are registered but not yet active: " + string(scaffold) +
		" requires you to review and trust them first — " + TrustSurface(scaffold) +
		". Until then Goalrail does nothing in your sessions. " +
		"Trust applies from the next session onward; run `gr health` to check."
}

// Inspect reports attachment health for one scaffold and one repository.
// It never writes anything.
func Inspect(scaffold Scaffold, home, repositoryRoot string) (AttachmentState, error) {
	configPath, err := ConfigPath(scaffold, home)
	if err != nil {
		return AttachmentState{}, err
	}
	connected, err := isConnected(scaffold, configPath)
	if err != nil {
		// A configuration we cannot parse is not a connection we can claim.
		connected = false
	}
	state := AttachmentState{
		Scaffold:    scaffold,
		Connected:   connected,
		Initialized: IsInitialized(repositoryRoot),
		Trust:       TrustUnknown,
		ConfigPath:  configPath,
		Repository:  repositoryRoot,
	}
	if connected {
		state.Trust = inspectTrust(scaffold, configPath)
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
		state.Working = true
	}
	return state, nil
}

// inspectTrust reads the scaffold's own record of which hooks the user has
// reviewed. Read-only and best-effort: an unreadable or unfamiliar shape yields
// TrustUnknown rather than an optimistic answer.
func inspectTrust(scaffold Scaffold, configPath string) TrustState {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return TrustPending
		}
		return TrustUnknown
	}
	switch scaffold {
	case ScaffoldCodex:
		// Codex records trust per hook under a state table keyed by the
		// configuration that declares it.
		if !strings.Contains(string(raw), "[hooks.state") {
			return TrustPending
		}
		if strings.Contains(string(raw), configPath+":session_start") {
			return TrustGranted
		}
		return TrustPending
	case ScaffoldClaudeCode:
		var settings map[string]any
		if json.Unmarshal(raw, &settings) != nil {
			return TrustUnknown
		}
		// No documented trust record has been observed for this scaffold, so
		// claiming either answer would be invention.
		return TrustUnknown
	default:
		return TrustUnknown
	}
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
