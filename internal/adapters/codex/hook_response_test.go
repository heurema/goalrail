package codex

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/localrun"
)

func TestSessionStartHookResponseCarriesTheAnnouncementVerbatim(t *testing.T) {
	rendered, err := RenderSessionStartHookResponse(localrun.EscalationAnnouncement)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Fatalf("hook event = %q", decoded.HookSpecificOutput.HookEventName)
	}
	// The transport is provider-specific; the content is not. Reformatting or
	// truncating the announcement here would let the delivered text drift from
	// the text the local-run boundary pinned.
	if decoded.HookSpecificOutput.AdditionalContext != localrun.EscalationAnnouncement {
		t.Fatal("the rendered response altered the announcement")
	}
}

func TestSessionStartHookResponseRejectsWhatItCannotCarry(t *testing.T) {
	for name, announcement := range map[string]string{
		"empty":         "",
		"invalid utf-8": string([]byte{0xff, 0xfe}),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := RenderSessionStartHookResponse(announcement); err == nil {
				t.Fatal("an announcement that cannot be carried was accepted")
			}
		})
	}
}

func TestSessionStartHookResponseEscapesTheAnnouncementSafely(t *testing.T) {
	// The announcement is multi-line and contains quotes-worthy punctuation; the
	// response must remain parseable rather than relying on the text's shape.
	rendered, err := RenderSessionStartHookResponse("line one\n\"quoted\"\tand\\escaped")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered, "\n") {
		t.Fatal("the rendered response contains a raw newline")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatalf("the rendered response does not parse: %v", err)
	}
}

func TestLocalRunAdapterCanAnnounce(t *testing.T) {
	adapter, err := NewLocalRunAdapter(&recordingLauncher{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.VerifyAnnouncementDelivery(); err != nil {
		t.Fatalf("the Codex adapter reported it cannot announce: %v", err)
	}
}
