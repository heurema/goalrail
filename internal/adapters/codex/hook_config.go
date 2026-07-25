package codex

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RenderSessionStartHookOverride renders the exact invocation-local hook
// definition supported by the pinned Codex adapter contract.
//
// The caller supplies an absolute path that is already normalized. The
// renderer rejects a path that lexical normalization would rewrite instead of
// normalizing it itself: a ".." segment following a symlinked component
// resolves differently for the kernel than for filepath.Clean, so normalizing
// could emit a hook that silently runs a different executable. The accepted
// path is embedded verbatim and the kernel performs the only resolution.
func RenderSessionStartHookOverride(capsuleExecutable string) (string, error) {
	if !filepath.IsAbs(capsuleExecutable) {
		return "", errors.New("codex: capsule executable must be absolute")
	}
	if !utf8.ValidString(capsuleExecutable) {
		return "", errors.New("codex: capsule executable is not valid UTF-8")
	}
	if containsControlCharacter(capsuleExecutable) {
		return "", errors.New("codex: capsule executable contains a control character")
	}
	if filepath.Clean(capsuleExecutable) != capsuleExecutable {
		return "", errors.New("codex: capsule executable must already be normalized")
	}

	command := quoteShellArgument(capsuleExecutable) + " hook"
	return `hooks.SessionStart=[{hooks=[{type="command",command=` +
		quoteTOMLBasicString(command) +
		`}]}]`, nil
}

func containsControlCharacter(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func quoteShellArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func quoteTOMLBasicString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
