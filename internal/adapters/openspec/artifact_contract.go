package openspec

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

const (
	ArtifactContractID      = "goalrail-context-intent"
	ArtifactContractVersion = uint32(1)
	ContractModeV1          = "goalrail-context-intent/v1"
	ContractModeLegacyV0    = "goalrail-context-intent/legacy-v0"
	ContractModeUnselected  = "unselected"
	MaxPairArtifactBytes    = 1 << 20
)

type ContractSelection struct {
	Identifier string `json:"identifier"`
	Version    uint32 `json:"version"`
	Mode       string `json:"mode"`
}

type ConformedPair struct {
	Selection ContractSelection
	Context   domain.ContextPack
	Intent    domain.IntentSnapshot
}

type pairProfile struct {
	selection             ContractSelection
	contextBindingVersion bool
	contextBindingV       bool
	contextItemsSeven     bool
	contextItemsSix       bool
	currentHeadings       bool
	legacyHeadings        bool
}

var (
	contractV1Profile = pairProfile{
		selection:             ContractSelection{Identifier: ArtifactContractID, Version: ArtifactContractVersion, Mode: ContractModeV1},
		contextBindingVersion: true,
		contextItemsSeven:     true,
		currentHeadings:       true,
	}
	legacyV0Profile = pairProfile{
		selection:             ContractSelection{Identifier: ArtifactContractID, Version: 0, Mode: ContractModeLegacyV0},
		contextBindingVersion: true,
		contextBindingV:       true,
		contextItemsSeven:     true,
		contextItemsSix:       true,
		currentHeadings:       true,
		legacyHeadings:        true,
	}
)

type artifactEnvelope struct {
	document markdownDocument
	metadata map[string]string
	rawKeys  map[string]string
	path     string
	kind     ArtifactKind
}

var contextMetadataKeys = map[string]struct{}{
	"context pack id":           {},
	"version":                   {},
	"previous version":          {},
	"started at":                {},
	"completed at":              {},
	"outcome":                   {},
	"artifact contract":         {},
	"artifact contract version": {},
}

var intentMetadataKeys = map[string]struct{}{
	"intent id":                 {},
	"version":                   {},
	"previous version":          {},
	"status":                    {},
	"owner":                     {},
	"context pack":              {},
	"run references":            {},
	"resolves":                  {},
	"disposition":               {},
	"artifact contract":         {},
	"artifact contract version": {},
}

// ConformPair selects exactly one profile before parsing pair semantics.
func ConformPair(
	contextBytes []byte,
	intentBytes []byte,
	contextPath string,
	intentPath string,
) (ConformedPair, error) {
	safeContextPath, ok := normalizeLogicalPath(contextPath)
	if !ok {
		return ConformedPair{}, newArtifactDiagnostic(
			ArtifactInputUnavailable, "context.md", ArtifactKindContext, ContractModeUnselected,
			"path", "unsafe logical path", "a relative path contained by the caller's root",
			"pass a safe repository-relative Context path", ErrMalformedArtifact,
		)
	}
	safeIntentPath, ok := normalizeLogicalPath(intentPath)
	if !ok {
		return ConformedPair{}, newArtifactDiagnostic(
			ArtifactInputUnavailable, "intent.md", ArtifactKindIntent, ContractModeUnselected,
			"path", "unsafe logical path", "a relative path contained by the caller's root",
			"pass a safe repository-relative Intent path", ErrMalformedArtifact,
		)
	}
	if len(contextBytes) == 0 || len(contextBytes) > MaxPairArtifactBytes {
		return ConformedPair{}, inputSizeDiagnostic(safeContextPath, ArtifactKindContext, len(contextBytes))
	}
	if len(intentBytes) == 0 || len(intentBytes) > MaxPairArtifactBytes {
		return ConformedPair{}, inputSizeDiagnostic(safeIntentPath, ArtifactKindIntent, len(intentBytes))
	}

	contextEnvelope, err := readArtifactEnvelope(contextBytes, safeContextPath, ArtifactKindContext)
	if err != nil {
		return ConformedPair{}, err
	}
	intentEnvelope, err := readArtifactEnvelope(intentBytes, safeIntentPath, ArtifactKindIntent)
	if err != nil {
		return ConformedPair{}, err
	}
	profile, err := selectPairProfile(contextEnvelope, intentEnvelope)
	if err != nil {
		return ConformedPair{}, err
	}
	if err := validateEnvelopeKeys(contextEnvelope, contextMetadataKeys, profile.selection.Mode); err != nil {
		return ConformedPair{}, err
	}
	if err := validateEnvelopeKeys(intentEnvelope, intentMetadataKeys, profile.selection.Mode); err != nil {
		return ConformedPair{}, err
	}

	contextPack, err := parseContextEnvelope(contextEnvelope, profile)
	if err != nil {
		return ConformedPair{}, err
	}
	intent, err := parseIntentEnvelope(intentEnvelope, &contextPack, profile)
	if err != nil {
		return ConformedPair{}, err
	}
	if err := missingContextReference(intent, contextPack, intentEnvelope.path, profile.selection.Mode); err != nil {
		return ConformedPair{}, err
	}
	if err := domain.ValidateIntentSnapshot(intent); err != nil {
		return ConformedPair{}, formatDiagnostic(
			intentEnvelope, profile.selection.Mode, "intent semantics", "invalid domain semantics",
			"a domain-valid Context-backed Intent Snapshot", "repair the named Intent field or section", err,
		)
	}
	if err := validatePairFlow(intent, contextEnvelope, intentEnvelope, profile.selection.Mode); err != nil {
		return ConformedPair{}, err
	}

	return ConformedPair{Selection: profile.selection, Context: contextPack, Intent: intent}, nil
}

func validatePairFlow(
	intent domain.IntentSnapshot,
	contextEnvelope artifactEnvelope,
	intentEnvelope artifactEnvelope,
	mode string,
) error {
	if intent.ContextPack == nil {
		return newArtifactDiagnostic(
			ArtifactContextBindingInvalid, intentEnvelope.path, ArtifactKindIntent, mode,
			"Context Pack", "missing parsed Context Pack", "a bound sibling Context Pack",
			"restore and bind context.md", ErrContextRequired,
		)
	}
	if intent.ContextPack.Outcome != domain.ContextSufficient {
		return formatDiagnostic(
			contextEnvelope, mode, "Outcome (intent.context.not_sufficient)", string(intent.ContextPack.Outcome),
			"sufficient", "resolve material unknowns before confirming the Intent", ErrMalformedArtifact,
		)
	}
	if intent.Status == domain.IntentConfirmed && intent.Confirmation != nil &&
		!intent.ContextPack.CompletedAt.Before(intent.Confirmation.ConfirmedAt) {
		return formatDiagnostic(
			contextEnvelope, mode, "Completed at (intent.context.completed_after_confirmation)",
			"context did not complete before confirmation", "Context completion before Intent confirmation",
			"correct the lifecycle timestamps or reconfirm a later Intent version", ErrMalformedArtifact,
		)
	}
	if err := domain.ValidateFlowIntentSnapshot(intent); err != nil {
		return formatDiagnostic(
			intentEnvelope, mode, "intent flow semantics", "invalid flow semantics",
			"a domain-valid Context-backed Intent", "repair the named Intent field or section", err,
		)
	}
	return nil
}

func inputSizeDiagnostic(logicalPath string, kind ArtifactKind, size int) error {
	observation := "empty input"
	if size > MaxPairArtifactBytes {
		observation = "input exceeds 1048576 bytes"
	}
	return newArtifactDiagnostic(
		ArtifactInputUnavailable, logicalPath, kind, ContractModeUnselected, "input", observation,
		"a non-empty artifact no larger than 1048576 bytes", "provide the bounded regular artifact", ErrMalformedArtifact,
	)
}

func readPairArtifactFile(path string, logicalPath string, kind ArtifactKind) ([]byte, error) {
	raw, err := boundedio.ReadRegularFile(path, string(kind)+" artifact", MaxPairArtifactBytes)
	if err != nil {
		return nil, inputUnavailableDiagnostic(logicalPath, kind, "unavailable or non-regular input", err)
	}
	if len(raw) == 0 {
		return nil, inputSizeDiagnostic(logicalPath, kind, 0)
	}
	return raw, nil
}

func inputUnavailableDiagnostic(logicalPath string, kind ArtifactKind, observation string, cause error) error {
	if observation == "unavailable or non-regular input" {
		observation = "input is unavailable, unreadable, or not a regular file"
	}
	repairHint := "restore the required artifact inside the change directory"
	if errors.Is(cause, ErrContextRequired) {
		repairHint = "restore context.md because OpenSpec context is required"
	}
	return newArtifactDiagnostic(
		ArtifactInputUnavailable, logicalPath, kind, ContractModeUnselected, "input", observation,
		"a readable bounded regular artifact", repairHint, cause,
	)
}

func readArtifactEnvelope(raw []byte, logicalPath string, kind ArtifactKind) (artifactEnvelope, error) {
	document, err := readMarkdownDocument(bytes.NewReader(raw))
	if err != nil {
		return artifactEnvelope{}, formatDiagnostic(
			artifactEnvelope{path: logicalPath, kind: kind}, ContractModeUnselected,
			"document", "malformed Markdown envelope", "unique top-level sections and readable Markdown",
			"repair the artifact envelope", err,
		)
	}
	metadata, err := parseBoldMetadata(document.preamble)
	if err != nil {
		return artifactEnvelope{}, newArtifactDiagnostic(
			ArtifactContractInvalid, logicalPath, kind, ContractModeUnselected, "metadata",
			"duplicate metadata field", "each metadata field exactly once",
			"remove the duplicate metadata row", err,
		)
	}
	rawKeys := make(map[string]string)
	for _, line := range document.preamble {
		key, _, found := parseBoldBullet(line)
		if !found {
			continue
		}
		rawKeys[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(key)
	}
	return artifactEnvelope{document: document, metadata: metadata, rawKeys: rawKeys, path: logicalPath, kind: kind}, nil
}

func selectPairProfile(context artifactEnvelope, intent artifactEnvelope) (pairProfile, error) {
	contextName, contextHasName := context.metadata["artifact contract"]
	contextVersion, contextHasVersion := context.metadata["artifact contract version"]
	intentName, intentHasName := intent.metadata["artifact contract"]
	intentVersion, intentHasVersion := intent.metadata["artifact contract version"]
	anyDeclaration := contextHasName || contextHasVersion || intentHasName || intentHasVersion
	if !anyDeclaration {
		return legacyV0Profile, nil
	}
	if !contextHasName || !contextHasVersion || !intentHasName || !intentHasVersion {
		return pairProfile{}, contractDiagnostic(
			ArtifactContractInvalid, ArtifactKindPair, "context.md + intent.md", "partial declaration",
			"both artifacts declare Artifact Contract and Artifact Contract Version",
			"add the complete equal declaration to both artifacts", ErrMalformedArtifact,
		)
	}
	if context.rawKeys["artifact contract"] != "Artifact Contract" ||
		context.rawKeys["artifact contract version"] != "Artifact Contract Version" ||
		intent.rawKeys["artifact contract"] != "Artifact Contract" ||
		intent.rawKeys["artifact contract version"] != "Artifact Contract Version" {
		return pairProfile{}, contractDiagnostic(
			ArtifactContractInvalid, ArtifactKindPair, "context.md + intent.md", "non-canonical field name",
			"exact Artifact Contract and Artifact Contract Version field names",
			"use the exact contract metadata rows in both artifacts", ErrMalformedArtifact,
		)
	}
	contextVersionNumber, contextVersionErr := parseContractVersion(contextVersion)
	intentVersionNumber, intentVersionErr := parseContractVersion(intentVersion)
	if contextVersionErr != nil || intentVersionErr != nil {
		return pairProfile{}, contractDiagnostic(
			ArtifactContractInvalid, ArtifactKindPair, "context.md + intent.md", "malformed version",
			"a positive decimal Artifact Contract Version in both artifacts",
			"replace both version values with the supported integer", ErrMalformedArtifact,
		)
	}
	contextName = strings.TrimSpace(contextName)
	intentName = strings.TrimSpace(intentName)
	if contextName == "" || intentName == "" {
		return pairProfile{}, contractDiagnostic(
			ArtifactContractInvalid, ArtifactKindPair, "context.md + intent.md", "empty identifier",
			"a non-empty Artifact Contract identifier in both artifacts",
			"replace both identifier values with the supported identifier", ErrMalformedArtifact,
		)
	}
	if contextName != intentName || contextVersionNumber != intentVersionNumber {
		return pairProfile{}, contractDiagnostic(
			ArtifactContractMismatch, ArtifactKindPair, "context.md + intent.md", "declarations differ",
			"identical contract identifier and version in both artifacts",
			"make the two declarations byte-equivalent", ErrMalformedArtifact,
		)
	}
	if contextName != ArtifactContractID || contextVersionNumber != ArtifactContractVersion {
		return pairProfile{}, contractDiagnostic(
			ArtifactContractUnsupported, ArtifactKindPair, "context.md + intent.md",
			fmt.Sprintf("%s/%d", sanitizeDiagnosticObservation(contextName), contextVersionNumber),
			ArtifactContractID+" version 1", "use a supported declared contract or remove both declarations only for pinned legacy input",
			ErrMalformedArtifact,
		)
	}
	return contractV1Profile, nil
}

func parseContractVersion(value string) (uint32, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil || parsed == 0 {
		return 0, ErrMalformedArtifact
	}
	return uint32(parsed), nil
}

func contractDiagnostic(
	code ArtifactDiagnosticCode,
	kind ArtifactKind,
	logicalPath string,
	observation string,
	expectation string,
	repairHint string,
	cause error,
) error {
	return newArtifactDiagnostic(
		code, logicalPath, kind, ContractModeUnselected, "artifact contract", observation,
		expectation, repairHint, cause,
	)
}

func validateEnvelopeKeys(envelope artifactEnvelope, allowed map[string]struct{}, mode string) error {
	for key := range envelope.metadata {
		if _, ok := allowed[key]; ok {
			continue
		}
		return formatDiagnostic(
			envelope, mode, "metadata", key, "only profile-owned metadata fields",
			"remove the unlisted metadata field or version the artifact contract", ErrMalformedArtifact,
		)
	}
	return nil
}

func formatDiagnostic(
	envelope artifactEnvelope,
	mode string,
	field string,
	observation string,
	expectation string,
	repairHint string,
	cause error,
) error {
	return newArtifactDiagnostic(
		ArtifactFormatInvalid, envelope.path, envelope.kind, mode, field, observation,
		expectation, repairHint, errors.Join(ErrMalformedArtifact, cause),
	)
}

func missingContextReference(intent domain.IntentSnapshot, context domain.ContextPack, logicalPath string, mode string) error {
	available := make(map[domain.ContextItemID]struct{}, len(context.Items))
	for _, item := range context.Items {
		available[item.ID] = struct{}{}
	}
	groups := [][]domain.IntentItem{intent.DesiredOutcomes, intent.NonGoals, intent.SuccessSignals}
	for _, group := range groups {
		for _, item := range group {
			for _, reference := range item.ContextRefs {
				if _, ok := available[reference]; ok {
					continue
				}
				return newArtifactDiagnostic(
					ArtifactContextReferenceMissing, logicalPath, ArtifactKindIntent, mode,
					"intent context reference", string(reference), "a Context Item ID present in context.md",
					"add the referenced Context Item or correct the reference because the context item reference does not exist", ErrMalformedArtifact,
				)
			}
		}
	}
	return nil
}
