package codex

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/localrun"
)

func decodeHookResponse(t *testing.T, rendered string) struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
} {
	t.Helper()
	var decoded struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(rendered), &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func TestSessionStartHookResponseCarriesTheAnnouncementVerbatim(t *testing.T) {
	rendered, err := RenderSessionStartHookResponse(
		localrun.EscalationAnnouncement,
		StartupHookSource,
	)
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeHookResponse(t, rendered)
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

func TestSessionStartHookResponseAnnouncesOnlyAtStartup(t *testing.T) {
	// Codex re-emits SessionStart on resume, clear, and compaction. Announcing
	// there would turn one launch-time statement into a recurring instruction
	// inside a single run.
	for _, source := range []string{"resume", "clear", "compact", "", "unknown"} {
		t.Run(source, func(t *testing.T) {
			rendered, err := RenderSessionStartHookResponse(
				localrun.EscalationAnnouncement,
				source,
			)
			if err != nil {
				t.Fatal(err)
			}
			decoded := decodeHookResponse(t, rendered)
			if decoded.HookSpecificOutput.HookEventName != "SessionStart" {
				t.Fatalf("hook event = %q", decoded.HookSpecificOutput.HookEventName)
			}
			if decoded.HookSpecificOutput.AdditionalContext != "" {
				t.Fatalf("source %q repeated the announcement inside a run", source)
			}
		})
	}
}

func TestSessionStartHookResponseRejectsWhatItCannotCarry(t *testing.T) {
	for name, announcement := range map[string]string{
		"empty":         "",
		"invalid utf-8": string([]byte{0xff, 0xfe}),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := RenderSessionStartHookResponse(announcement, StartupHookSource); err == nil {
				t.Fatal("an announcement that cannot be carried was accepted")
			}
		})
	}
}

func TestSessionStartHookResponseEscapesTheAnnouncementSafely(t *testing.T) {
	// The announcement is multi-line and contains quotes-worthy punctuation; the
	// response must remain parseable rather than relying on the text's shape.
	rendered, err := RenderSessionStartHookResponse(
		"line one\n\"quoted\"\tand\\escaped",
		StartupHookSource,
	)
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

func TestAdapterDelegatesAnnouncementDeliveryToTheLauncher(t *testing.T) {
	// The adapter places the announcement in the environment, but only the
	// launcher knows whether a capsule reads it and answers the hook. An adapter
	// that claimed delivery it does not perform would defeat the fail-closed
	// check in Start.
	silent := &recordingLauncher{announcementErr: errors.New("no capsule installed")}
	adapter, err := NewLocalRunAdapter(silent, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.VerifyAnnouncementDelivery(); err == nil {
		t.Fatal("the adapter claimed delivery its launcher does not perform")
	}

	capable := &recordingLauncher{}
	adapter, err = NewLocalRunAdapter(capable, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.VerifyAnnouncementDelivery(); err != nil {
		t.Fatalf("a capable launcher was reported as unable: %v", err)
	}
}
