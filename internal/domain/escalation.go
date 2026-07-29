package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxEscalationBytes bounds an escalation payload. It is sized for a question
// rather than for a sentence, so it does not reuse MaxTextBytes.
const MaxEscalationBytes = 64 << 10

// EscalationSchema names the provider-neutral payload shape that downstream
// tooling validates. Goalrail itself never parses the payload.
const EscalationSchema = "goalrail.escalation/v0"

var ErrInvalidEscalationPayload = errors.New("invalid escalation payload")

// ValidateEscalationPayload applies retention hygiene to an escalation payload
// and nothing else. It deliberately performs no semantic validation: Goalrail
// runs no checks, and the meaning of the payload belongs to EscalationSchema,
// which downstream tooling validates.
//
// The rules are the retained-text rules already applied to WorkSpec text, so
// the two cannot drift into separate families.
func ValidateEscalationPayload(content []byte) error {
	if len(content) == 0 || strings.TrimSpace(string(content)) == "" {
		return fmt.Errorf("%w: payload cannot be empty", ErrInvalidEscalationPayload)
	}
	if len(content) > MaxEscalationBytes {
		return fmt.Errorf(
			"%w: payload exceeds %d bytes",
			ErrInvalidEscalationPayload,
			MaxEscalationBytes,
		)
	}
	if !utf8.Valid(content) {
		return fmt.Errorf("%w: payload must be valid UTF-8", ErrInvalidEscalationPayload)
	}
	value := string(content)
	if hasUnsafeControl(value) {
		return fmt.Errorf(
			"%w: payload contains an unsupported control character",
			ErrInvalidEscalationPayload,
		)
	}
	if hasSecretShapedContent(value) {
		return fmt.Errorf(
			"%w: secret-shaped content cannot be retained",
			ErrInvalidEscalationPayload,
		)
	}
	return nil
}
