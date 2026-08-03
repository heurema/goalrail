package openspec

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ArtifactDiagnosticCode is a stable machine-readable conformance failure.
type ArtifactDiagnosticCode string

const (
	ArtifactInputUnavailable        ArtifactDiagnosticCode = "ARTIFACT_INPUT_UNAVAILABLE"
	ArtifactContractInvalid         ArtifactDiagnosticCode = "ARTIFACT_CONTRACT_INVALID"
	ArtifactContractUnsupported     ArtifactDiagnosticCode = "ARTIFACT_CONTRACT_UNSUPPORTED"
	ArtifactContractMismatch        ArtifactDiagnosticCode = "ARTIFACT_CONTRACT_MISMATCH"
	ArtifactFormatInvalid           ArtifactDiagnosticCode = "ARTIFACT_FORMAT_INVALID"
	ArtifactContextBindingInvalid   ArtifactDiagnosticCode = "ARTIFACT_CONTEXT_BINDING_INVALID"
	ArtifactContextReferenceMissing ArtifactDiagnosticCode = "ARTIFACT_CONTEXT_REFERENCE_MISSING"
)

// ArtifactKind identifies the member of the Context/Intent pair that failed.
type ArtifactKind string

const (
	ArtifactKindContext ArtifactKind = "context"
	ArtifactKindIntent  ArtifactKind = "intent"
	ArtifactKindPair    ArtifactKind = "pair"
)

// ArtifactDiagnostic contains only bounded, non-sensitive repair information.
// The underlying cause is deliberately excluded from serialization and
// rendering, but remains available to errors.Is through Unwrap.
type ArtifactDiagnostic struct {
	Code           ArtifactDiagnosticCode `json:"code"`
	Path           string                 `json:"path"`
	ArtifactKind   ArtifactKind           `json:"artifact_kind"`
	ContractMode   string                 `json:"contract_mode"`
	FieldOrSection string                 `json:"field_or_section"`
	Observation    string                 `json:"observation"`
	Expectation    string                 `json:"expectation"`
	RepairHint     string                 `json:"repair_hint"`

	cause error
}

func (diagnostic *ArtifactDiagnostic) Error() string {
	if diagnostic == nil {
		return ""
	}
	return fmt.Sprintf(
		"%s: %s %s at %s [%s]: observed %q; expected %s; repair: %s",
		diagnostic.Code,
		diagnostic.ArtifactKind,
		diagnostic.Path,
		diagnostic.FieldOrSection,
		diagnostic.ContractMode,
		diagnostic.Observation,
		diagnostic.Expectation,
		diagnostic.RepairHint,
	)
}

func (diagnostic *ArtifactDiagnostic) Unwrap() error {
	if diagnostic == nil {
		return nil
	}
	return diagnostic.cause
}

func newArtifactDiagnostic(
	code ArtifactDiagnosticCode,
	logicalPath string,
	kind ArtifactKind,
	mode string,
	fieldOrSection string,
	observation string,
	expectation string,
	repairHint string,
	cause error,
) *ArtifactDiagnostic {
	return &ArtifactDiagnostic{
		Code:           code,
		Path:           logicalPath,
		ArtifactKind:   kind,
		ContractMode:   mode,
		FieldOrSection: fieldOrSection,
		Observation:    sanitizeDiagnosticObservation(observation),
		Expectation:    expectation,
		RepairHint:     repairHint,
		cause:          cause,
	}
}

func normalizeLogicalPath(value string) (string, bool) {
	if value == "" || filepath.IsAbs(value) || strings.ContainsRune(value, '\x00') {
		return "", false
	}
	normalized := path.Clean(strings.ReplaceAll(filepath.ToSlash(value), `\`, "/"))
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", false
	}
	return normalized, true
}

func sanitizeDiagnosticObservation(value string) string {
	if secretShaped(value) {
		return "[REDACTED]"
	}
	var builder strings.Builder
	for _, current := range value {
		switch current {
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			if unicode.IsControl(current) {
				fmt.Fprintf(&builder, `\u%04X`, current)
			} else if current == utf8.RuneError {
				builder.WriteRune('\uFFFD')
			} else {
				builder.WriteRune(current)
			}
		}
	}
	return truncateRunes(builder.String(), 80)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func secretShaped(value string) bool {
	lower := strings.ToLower(value)
	markers := []string{
		"authorization:", "bearer ", "password=", "secret=", "token=",
		"api_key=", "apikey=", "github_pat_", "ghp_", "sk-", "akia",
	}
	for _, marker := range markers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
