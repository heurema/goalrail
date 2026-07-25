package codex

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"
)

// RenderSessionStartHookOverride renders the exact invocation-local hook
// definition supported by the pinned Codex adapter contract.
func RenderSessionStartHookOverride(capsuleExecutable string) (string, error) {
	if !filepath.IsAbs(capsuleExecutable) {
		return "", errors.New("codex: capsule executable must be absolute")
	}
	if containsControlCharacter(capsuleExecutable) {
		return "", errors.New("codex: capsule executable contains a control character")
	}

	command := quoteShellArgument(filepath.Clean(capsuleExecutable)) + " hook"
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
