package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateEscalationPayloadAcceptsABoundedQuestion(t *testing.T) {
	payload := []byte("---\nschema: goalrail.escalation/v0\nquestion: Which one governs?\n---\n")
	if err := ValidateEscalationPayload(payload); err != nil {
		t.Fatalf("a bounded question was rejected: %v", err)
	}
}

func TestValidateEscalationPayloadAppliesHygieneOnly(t *testing.T) {
	// Goalrail runs no checks, so the payload's meaning is never validated here.
	// Anything that passes hygiene is retained, however little it says.
	if err := ValidateEscalationPayload([]byte("this says nothing useful\n")); err != nil {
		t.Fatalf("a semantically empty but hygienic payload was rejected: %v", err)
	}
}

func TestValidateEscalationPayloadRejectsUnretainableContent(t *testing.T) {
	for name, payload := range map[string][]byte{
		"empty":            nil,
		"whitespace only":  []byte("   \n\t\n"),
		"oversized":        []byte(strings.Repeat("x", MaxEscalationBytes+1)),
		"invalid utf-8":    {0xff, 0xfe, 0x0a},
		"control byte":     []byte("question\x00terminator"),
		"secret shaped":    []byte("needs api_key: 8f3ca11b9d0e4c72 to proceed"),
		"private key body": []byte("-----BEGIN RSA PRIVATE KEY-----\nabc\n"),
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateEscalationPayload(payload)
			if err == nil {
				t.Fatal("unretainable content was accepted")
			}
			if !errors.Is(err, ErrInvalidEscalationPayload) {
				t.Fatalf("error = %v, want ErrInvalidEscalationPayload", err)
			}
		})
	}
}

func TestValidateEscalationPayloadAcceptsExactlyTheBound(t *testing.T) {
	if err := ValidateEscalationPayload([]byte(strings.Repeat("x", MaxEscalationBytes))); err != nil {
		t.Fatalf("a payload of exactly the bound was rejected: %v", err)
	}
}
