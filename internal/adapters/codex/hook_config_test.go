package codex

import (
	"path/filepath"
	"testing"
	"unicode/utf8"
)

type pinnedMatcherGroup struct {
	hooks []pinnedHookHandler
}

type pinnedHookHandler struct {
	handlerType string
	command     string
}

func TestPinnedSessionStartHookContract(t *testing.T) {
	t.Parallel()

	legacy := decodePinnedMatcherGroup(map[string]any{
		"command": []string{"/tmp/goalrail-capsule", "hook"},
	})
	if got := len(legacy.hooks); got != 0 {
		t.Fatalf("spent override registered %d handlers, want 0", got)
	}

	const capsuleExecutable = "/tmp/goalrail-capsule"
	override, err := RenderSessionStartHookOverride(capsuleExecutable)
	if err != nil {
		t.Fatal(err)
	}

	repaired := decodePinnedMatcherGroup(map[string]any{
		"hooks": []pinnedHookHandler{
			{
				handlerType: "command",
				command:     "'/tmp/goalrail-capsule' hook",
			},
		},
	})
	if got := len(repaired.hooks); got != 1 {
		t.Fatalf("repaired override registered %d handlers, want 1", got)
	}
	if repaired.hooks[0].handlerType != "command" {
		t.Fatalf("handler type = %q, want command", repaired.hooks[0].handlerType)
	}
	if repaired.hooks[0].command != "'/tmp/goalrail-capsule' hook" {
		t.Fatalf("handler command = %q", repaired.hooks[0].command)
	}

	const expected = `hooks.SessionStart=[{hooks=[{type="command",command="'/tmp/goalrail-capsule' hook"}]}]`
	if override != expected {
		t.Fatalf("override = %q, want %q", override, expected)
	}
}

func decodePinnedMatcherGroup(fields map[string]any) pinnedMatcherGroup {
	hooks := []pinnedHookHandler{}
	if configured, ok := fields["hooks"].([]pinnedHookHandler); ok {
		hooks = configured
	}
	return pinnedMatcherGroup{hooks: hooks}
}

func TestRenderSessionStartHookOverrideQuotesExecutable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		capsuleExecutable string
		expected          string
	}{
		{
			name:              "spaces and shell metacharacters",
			capsuleExecutable: "/tmp/goalrail capsule;$HOME",
			expected:          `hooks.SessionStart=[{hooks=[{type="command",command="'/tmp/goalrail capsule;$HOME' hook"}]}]`,
		},
		{
			name:              "embedded single quote",
			capsuleExecutable: "/tmp/goalrail'capsule",
			expected:          `hooks.SessionStart=[{hooks=[{type="command",command="'/tmp/goalrail'\"'\"'capsule' hook"}]}]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := RenderSessionStartHookOverride(test.capsuleExecutable)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.expected {
				t.Fatalf("override = %q, want %q", got, test.expected)
			}
		})
	}
}

func TestRenderSessionStartHookOverrideRejectsUnsafeExecutable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		capsuleExecutable string
	}{
		{
			name:              "relative path",
			capsuleExecutable: "tmp/goalrail-capsule",
		},
		{
			name:              "nul",
			capsuleExecutable: "/tmp/goalrail\x00capsule",
		},
		{
			name:              "carriage return",
			capsuleExecutable: "/tmp/goalrail\rcapsule",
		},
		{
			name:              "newline",
			capsuleExecutable: "/tmp/goalrail\ncapsule",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := RenderSessionStartHookOverride(test.capsuleExecutable)
			if err == nil {
				t.Fatal("expected error")
			}
			if got != "" {
				t.Fatalf("partial override = %q, want empty", got)
			}
		})
	}
}

func TestRenderSessionStartHookOverrideRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		capsuleExecutable string
	}{
		{
			name:              "invalid continuation byte",
			capsuleExecutable: "/tmp/goalrail\xffcapsule",
		},
		{
			name:              "truncated multi-byte sequence",
			capsuleExecutable: "/tmp/goalrail\xe2\x82capsule",
		},
		{
			name:              "surrogate half encoding",
			capsuleExecutable: "/tmp/goalrail\xed\xa0\x80capsule",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// The control-character guard cannot catch these: ranging over an
			// invalid string yields utf8.RuneError, which is not a control
			// character. Only the UTF-8 guard rejects them.
			if containsControlCharacter(test.capsuleExecutable) {
				t.Fatal("input must exercise the UTF-8 guard, not the control-character guard")
			}

			got, err := RenderSessionStartHookOverride(test.capsuleExecutable)
			if err == nil {
				t.Fatal("expected error")
			}
			if got != "" {
				t.Fatalf("partial override = %q, want empty", got)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("override %q is not valid UTF-8 and cannot be valid TOML", got)
			}
		})
	}
}

func TestRenderSessionStartHookOverrideRejectsUnnormalizedExecutable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		capsuleExecutable string
	}{
		{
			name:              "parent traversal after component",
			capsuleExecutable: "/work/link/../goalrail-capsule",
		},
		{
			name:              "current directory segment",
			capsuleExecutable: "/tmp/./goalrail-capsule",
		},
		{
			name:              "duplicate separator",
			capsuleExecutable: "/tmp//goalrail-capsule",
		},
		{
			name:              "trailing separator",
			capsuleExecutable: "/tmp/goalrail-capsule/",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			normalized := filepath.Clean(test.capsuleExecutable)
			if normalized == test.capsuleExecutable {
				t.Fatal("input must be unnormalized to exercise the guard")
			}

			got, err := RenderSessionStartHookOverride(test.capsuleExecutable)
			if err == nil {
				t.Fatal("expected error")
			}
			if got != "" {
				t.Fatalf("partial override = %q, want empty", got)
			}
		})
	}
}

func TestRenderSessionStartHookOverrideEmbedsExecutableVerbatim(t *testing.T) {
	t.Parallel()

	// A ".." segment after a symlinked component resolves differently for the
	// kernel than for filepath.Clean, so an accepted path must reach the hook
	// exactly as supplied, with no lexical rewriting.
	const capsuleExecutable = "/tmp/goalrail-capsule"

	override, err := RenderSessionStartHookOverride(capsuleExecutable)
	if err != nil {
		t.Fatal(err)
	}

	const expected = `hooks.SessionStart=[{hooks=[{type="command",command="'/tmp/goalrail-capsule' hook"}]}]`
	if override != expected {
		t.Fatalf("override = %q, want %q", override, expected)
	}
	if !utf8.ValidString(override) {
		t.Fatalf("override %q is not valid UTF-8", override)
	}
}
