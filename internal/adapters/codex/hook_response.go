package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"
)

// EscalationAnnouncementEnvironment carries the provider-neutral announcement
// from the adapter to the capsule that answers the SessionStart hook. The
// adapter and the capsule are separate processes, so the text travels the
// invocation environment rather than a shared value.
const EscalationAnnouncementEnvironment = "GOALRAIL_ESCALATION_ANNOUNCEMENT"

// sessionStartHookEvent is the event name the pinned Codex hook contract
// requires on hook-specific output.
const sessionStartHookEvent = "SessionStart"

type sessionStartHookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

type sessionStartHookResponse struct {
	HookSpecificOutput sessionStartHookSpecificOutput `json:"hookSpecificOutput"`
}

// StartupHookSource is the only SessionStart source that opens a run. Codex
// also emits SessionStart for `resume`, `clear`, and `compact`, all of which
// occur inside a run that has already started.
const StartupHookSource = "startup"

// RenderSessionStartHookResponse renders the reply a capsule writes when Codex
// invokes the SessionStart hook, carrying the escalation announcement as the
// session's additional context.
//
// The announcement is delivered exactly once, so it is emitted only for the
// startup source. Codex re-emits SessionStart on resume, clear, and compaction
// — all recognised sources in the correlation path — and repeating the
// announcement there would turn one statement into a recurring instruction
// inside a single run.
//
// This is the provider-specific half of the announcement. The content is
// composed by the local-run boundary and is provider-neutral; only this
// transport knows the shape Codex accepts. The renderer therefore never edits
// the text — it rejects an announcement it cannot carry verbatim rather than
// truncating or reformatting one.
func RenderSessionStartHookResponse(announcement string, source string) (string, error) {
	if announcement == "" {
		return "", errors.New("codex: escalation announcement is empty")
	}
	if !utf8.ValidString(announcement) {
		return "", errors.New("codex: escalation announcement is not valid UTF-8")
	}
	response := sessionStartHookResponse{
		HookSpecificOutput: sessionStartHookSpecificOutput{
			HookEventName: sessionStartHookEvent,
		},
	}
	if source == StartupHookSource {
		response.HookSpecificOutput.AdditionalContext = announcement
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("codex: encode SessionStart hook response: %w", err)
	}
	return string(encoded), nil
}
