package openspec

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/internal/domain"
)

func parseContextEnvelope(envelope artifactEnvelope, profile pairProfile) (domain.ContextPack, error) {
	metadata := envelope.metadata
	version, err := parseUint32Metadata(metadata, "version")
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Version", "missing or invalid value",
			"a positive Context Pack version", "set Version to a positive integer", err,
		)
	}
	previousVersion, err := parseOptionalUint32Metadata(metadata, "previous version")
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Previous version", "invalid value",
			"none or a positive earlier version", "correct Previous version", err,
		)
	}
	startedAt, err := parseArtifactTime(cleanInline(metadata["started at"]))
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Started at", "invalid timestamp",
			"an ISO-8601 timestamp", "correct Started at", err,
		)
	}
	completedAt, err := parseArtifactTime(cleanInline(metadata["completed at"]))
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Completed at", "invalid timestamp",
			"an ISO-8601 timestamp", "correct Completed at", err,
		)
	}
	itemLines, err := envelope.document.requiredSection("Context Items")
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Context Items", "missing section",
			"one Context Items section", "add the required section", err,
		)
	}
	unknownLines, err := envelope.document.requiredSection("Material Unknowns")
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Material Unknowns", "missing section",
			"one Material Unknowns section", "add the required section", err,
		)
	}
	items, err := parseContextItemsForProfile(itemLines, profile)
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Context Items", "unsupported table shape",
			contextTableExpectation(profile), "use one table shape admitted by the selected profile", err,
		)
	}
	unknowns, err := parseContextUnknowns(unknownLines)
	if err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Material Unknowns", "invalid section",
			"explicit None or a structured unknowns table", "repair Material Unknowns", err,
		)
	}
	pack := domain.ContextPack{
		ID:              domain.ContextPackID(cleanInline(metadata["context pack id"])),
		Version:         version,
		PreviousVersion: previousVersion,
		StartedAt:       startedAt,
		CompletedAt:     completedAt,
		Outcome:         domain.ContextCollectionOutcome(strings.ToLower(cleanInline(metadata["outcome"]))),
		Items:           items,
		Unknowns:        unknowns,
	}
	if err := domain.ValidateContextPack(pack); err != nil {
		return domain.ContextPack{}, formatDiagnostic(
			envelope, profile.selection.Mode, "context semantics", "invalid domain semantics",
			"a domain-valid Context Pack", "repair the named Context field or section", err,
		)
	}
	return pack, nil
}

func parseIntentEnvelope(
	envelope artifactEnvelope,
	contextPack *domain.ContextPack,
	profile pairProfile,
) (domain.IntentSnapshot, error) {
	metadata := envelope.metadata
	version, err := parseUint32Metadata(metadata, "version")
	if err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Version", "missing or invalid value",
			"a positive Intent version", "set Version to a positive integer", err,
		)
	}
	previousVersion, err := parseOptionalUint32Metadata(metadata, "previous version")
	if err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Previous version", "invalid value",
			"none or a positive earlier version", "correct Previous version", err,
		)
	}
	declaredID, declaredVersion, err := parseContextPackDeclarationForProfile(metadata["context pack"], profile)
	if err != nil {
		return domain.IntentSnapshot{}, newArtifactDiagnostic(
			ArtifactContextBindingInvalid, envelope.path, ArtifactKindIntent, profile.selection.Mode,
			"Context Pack", "unsupported binding syntax", contextBindingExpectation(profile),
			"use a supported Context Pack ID and version declaration", err,
		)
	}
	if contextPack == nil || declaredID != contextPack.ID || declaredVersion != contextPack.Version {
		return domain.IntentSnapshot{}, newArtifactDiagnostic(
			ArtifactContextBindingInvalid, envelope.path, ArtifactKindIntent, profile.selection.Mode,
			"Context Pack", fmt.Sprintf("%s version %d", declaredID, declaredVersion),
			"the exact Context Pack ID and version from context.md",
			"correct the declaration because it does not match context.md", ErrMalformedArtifact,
		)
	}

	sourceLines, err := requiredPairSection(envelope, profile, "Source Evidence")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	desiredLines, err := requiredPairSection(envelope, profile, "Desired Outcomes")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	nonGoalLines, err := requiredPairSection(envelope, profile, "Non-Goals")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	signalLines, err := requiredPairSection(envelope, profile, "Observable Success Signals")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	ambiguityLines, err := requiredPairSection(envelope, profile, "Ambiguities and Unknowns")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	confirmationLines, err := requiredPairSection(envelope, profile, "Confirmation")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	resolvedEscalation, err := parseEscalationResolution(metadata)
	if err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Resolves and Disposition", "invalid lifecycle provenance",
			"both fields together in the supported form", "repair or remove both lifecycle fields", err,
		)
	}
	snapshot := domain.IntentSnapshot{
		ID:                 domain.IntentID(cleanInline(metadata["intent id"])),
		Version:            version,
		PreviousVersion:    previousVersion,
		Status:             domain.IntentStatus(strings.ToLower(cleanInline(metadata["status"]))),
		ContextPack:        contextPack,
		ResolvedEscalation: resolvedEscalation,
	}
	if snapshot.SourceEvidence, err = parseSourceEvidence(sourceLines); err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Source Evidence", "invalid evidence rows",
			"structured SE-* evidence bullets", "repair Source Evidence", err,
		)
	}
	if snapshot.DesiredOutcomes, err = parseDesiredOutcomesForProfile(desiredLines, profile); err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Desired Outcomes", "unsupported table heading",
			intentHeadingExpectation(profile, true), "use a heading admitted by the selected profile", err,
		)
	}
	if snapshot.NonGoals, err = parseNonGoalsForProfile(nonGoalLines, profile); err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Non-Goals", "unsupported table heading",
			intentHeadingExpectation(profile, false), "use a heading admitted by the selected profile", err,
		)
	}
	if snapshot.SuccessSignals, err = parseIntentItems(
		signalLines,
		[]string{"ID", "Signal", "Measurement", "Evidence"},
		1,
		3,
	); err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Observable Success Signals", "invalid table",
			"ID, Signal, Measurement, Evidence", "repair the success-signal table", err,
		)
	}
	if snapshot.Ambiguities, err = parseAmbiguities(ambiguityLines); err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Ambiguities and Unknowns", "invalid section",
			"explicit None or a structured ambiguity table", "repair the ambiguity section", err,
		)
	}
	if snapshot.Confirmation, err = parseConfirmation(confirmationLines); err != nil {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Confirmation", "invalid confirmation",
			"complete confirmation metadata or an empty candidate confirmation", "repair Confirmation", err,
		)
	}
	if owner := cleanInline(metadata["owner"]); owner != "" && snapshot.Confirmation != nil && owner != snapshot.Confirmation.Owner {
		return domain.IntentSnapshot{}, formatDiagnostic(
			envelope, profile.selection.Mode, "Owner", "owner mismatch",
			"the same owner in metadata and Confirmation", "make the two owner values equal", ErrMalformedArtifact,
		)
	}
	return snapshot, nil
}

func requiredPairSection(envelope artifactEnvelope, profile pairProfile, section string) ([]string, error) {
	lines, err := envelope.document.requiredSection(section)
	if err != nil {
		return nil, formatDiagnostic(
			envelope, profile.selection.Mode, section, "missing section",
			"one "+section+" section", "add the required section", err,
		)
	}
	return lines, nil
}

func parseContextPackDeclarationForProfile(
	value string,
	profile pairProfile,
) (domain.ContextPackID, uint32, error) {
	fields := strings.Fields(cleanInline(value))
	var rawID string
	var rawVersion string
	switch {
	case profile.contextBindingVersion && len(fields) == 3 && strings.EqualFold(fields[1], "version"):
		rawID = fields[0]
		rawVersion = fields[2]
	case profile.contextBindingV && len(fields) == 2 && len(fields[1]) > 1 && (fields[1][0] == 'v' || fields[1][0] == 'V'):
		rawID = fields[0]
		rawVersion = fields[1][1:]
	default:
		return "", 0, ErrMalformedArtifact
	}
	id := domain.ContextPackID(strings.Trim(rawID, "`"))
	version, err := strconv.ParseUint(strings.Trim(rawVersion, "`"), 10, 32)
	if err != nil || version == 0 || !domain.IsCanonicalID(string(id)) {
		return "", 0, ErrMalformedArtifact
	}
	return id, uint32(version), nil
}

func parseContextItemsForProfile(lines []string, profile pairProfile) ([]domain.ContextItem, error) {
	seven := []string{"ID", "Kind", "Claim", "Source", "Verification recipe", "Observed at", "Relevance"}
	six := []string{"ID", "Kind", "Claim", "Source", "Observed at", "Relevance"}
	header, err := firstMarkdownTableHeader(lines)
	if err != nil {
		return nil, err
	}
	var expected []string
	recipeIndex, observedIndex, relevanceIndex := -1, 0, 0
	switch {
	case profile.contextItemsSeven && equalHeaders(header, seven):
		expected = seven
		recipeIndex, observedIndex, relevanceIndex = 4, 5, 6
	case profile.contextItemsSix && equalHeaders(header, six):
		expected = six
		observedIndex, relevanceIndex = 4, 5
	default:
		return nil, ErrMalformedArtifact
	}
	rows, err := parseMarkdownTable(lines, expected)
	if err != nil {
		return nil, err
	}
	items := make([]domain.ContextItem, 0, len(rows))
	for _, row := range rows {
		recipe := ""
		if recipeIndex >= 0 {
			recipe = cleanInline(row[recipeIndex])
			if recipe == "" {
				return nil, ErrMalformedArtifact
			}
		}
		observedAt, err := parseArtifactTime(cleanInline(row[observedIndex]))
		if err != nil {
			return nil, err
		}
		items = append(items, domain.ContextItem{
			ID:                 domain.ContextItemID(cleanInline(row[0])),
			Kind:               domain.ContextItemKind(strings.ToLower(cleanInline(row[1]))),
			Claim:              cleanInline(row[2]),
			SourceRef:          domain.EvidenceReference(cleanInline(row[3])),
			VerificationRecipe: recipe,
			ObservedAt:         observedAt,
			Relevance:          cleanInline(row[relevanceIndex]),
		})
	}
	return items, nil
}

func parseDesiredOutcomesForProfile(lines []string, profile pairProfile) ([]domain.IntentItem, error) {
	current := []string{"ID", "Outcome", "Verification action", "Evidence"}
	legacy := []string{"ID", "Confirmed wording", "Verification action", "Evidence"}
	return parseIntentItemsBySelectedHeader(lines, profile, current, legacy, 1, 3)
}

func parseNonGoalsForProfile(lines []string, profile pairProfile) ([]domain.IntentItem, error) {
	current := []string{"ID", "Boundary", "Evidence"}
	legacy := []string{"ID", "Confirmed boundary", "Evidence"}
	return parseIntentItemsBySelectedHeader(lines, profile, current, legacy, 1, 2)
}

func parseIntentItemsBySelectedHeader(
	lines []string,
	profile pairProfile,
	current []string,
	legacy []string,
	statementColumn int,
	evidenceColumn int,
) ([]domain.IntentItem, error) {
	header, err := firstMarkdownTableHeader(lines)
	if err != nil {
		return nil, err
	}
	switch {
	case profile.currentHeadings && equalHeaders(header, current):
		return parseIntentItems(lines, current, statementColumn, evidenceColumn)
	case profile.legacyHeadings && equalHeaders(header, legacy):
		return parseIntentItems(lines, legacy, statementColumn, evidenceColumn)
	default:
		return nil, ErrMalformedArtifact
	}
}

func firstMarkdownTableHeader(lines []string) ([]string, error) {
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		return splitMarkdownRow(strings.TrimSpace(line))
	}
	return nil, ErrMalformedArtifact
}

func equalHeaders(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if normalizeHeader(actual[index]) != normalizeHeader(expected[index]) {
			return false
		}
	}
	return true
}

func contextTableExpectation(profile pairProfile) string {
	if profile.contextItemsSix {
		return "the pinned six- or seven-column Context Items table"
	}
	return "the seven-column Context Items table with Verification recipe"
}

func contextBindingExpectation(profile pairProfile) string {
	if profile.contextBindingV {
		return "<id> version <number> or <id> v<number>"
	}
	return "<id> version <number>"
}

func intentHeadingExpectation(profile pairProfile, desired bool) string {
	if desired {
		if profile.legacyHeadings {
			return "Outcome or Confirmed wording in the Desired Outcomes table"
		}
		return "Outcome in the Desired Outcomes table"
	}
	if profile.legacyHeadings {
		return "Boundary or Confirmed boundary in the Non-Goals table"
	}
	return "Boundary in the Non-Goals table"
}
