package openspec

import (
	"bufio"
	"bytes"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

// IntentEnvelopeInspection is a bounded pre-parse view used to decide whether
// an ambient change requires pair validation and whether malformed input made
// an explicit confirmed-status claim. It does not certify the artifact.
type IntentEnvelopeInspection struct {
	ClaimsConfirmed          bool
	DeclaresContext          bool
	DeclaresArtifactContract bool
}

func InspectIntentEnvelope(raw []byte) (IntentEnvelopeInspection, error) {
	if len(raw) == 0 || len(raw) > MaxPairArtifactBytes {
		return IntentEnvelopeInspection{}, ErrMalformedArtifact
	}
	var inspection IntentEnvelopeInspection
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 4096), MaxPairArtifactBytes)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if strings.HasPrefix(line, "## ") {
			break
		}
		key, value, ok := parseBoldBullet(line)
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "status":
			if strings.TrimSpace(value) == "confirmed" {
				inspection.ClaimsConfirmed = true
			}
		case "context pack":
			inspection.DeclaresContext = cleanInline(value) != ""
		case "artifact contract", "artifact contract version":
			inspection.DeclaresArtifactContract = true
		}
	}
	if err := scanner.Err(); err != nil {
		return inspection, err
	}
	return inspection, nil
}

func PairRequired(
	changeDir string,
	inspection IntentEnvelopeInspection,
	siblingContextPresent bool,
) (bool, error) {
	if siblingContextPresent || inspection.DeclaresContext || inspection.DeclaresArtifactContract {
		return true, nil
	}
	return pairRequiredBySchema(changeDir)
}

func pairRequiredBySchema(changeDir string) (bool, error) {
	schema, err := readChangeSchema(filepath.Join(changeDir, ".openspec.yaml"))
	if err != nil {
		return false, err
	}
	return schema == goalrailIntentSchema, nil
}

// ReadIntentArtifact preserves the legitimate intent-only lifecycle while
// giving ambient discovery the same safe typed format failure surface.
func ReadIntentArtifact(raw []byte, logicalPath string) (domain.IntentSnapshot, error) {
	safePath, ok := normalizeLogicalPath(logicalPath)
	if !ok {
		return domain.IntentSnapshot{}, inputUnavailableDiagnostic(
			"intent.md", ArtifactKindIntent, "unsafe logical path", ErrMalformedArtifact,
		)
	}
	if len(raw) == 0 || len(raw) > MaxPairArtifactBytes {
		return domain.IntentSnapshot{}, inputSizeDiagnostic(safePath, ArtifactKindIntent, len(raw))
	}
	snapshot, err := ReadIntent(bytes.NewReader(raw))
	if err != nil {
		return domain.IntentSnapshot{}, newArtifactDiagnostic(
			ArtifactFormatInvalid, safePath, ArtifactKindIntent, ContractModeUnselected,
			"intent artifact", "invalid intent-only format", "a domain-valid Intent Snapshot",
			"repair the named Intent field or section", err,
		)
	}
	return snapshot, nil
}
