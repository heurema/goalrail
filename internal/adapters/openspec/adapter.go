// Package openspec translates the repository's intent-first OpenSpec artifacts
// into provider-neutral domain values. Markdown remains an adapter concern.
package openspec

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/heurema/goalrail/internal/domain"
)

var (
	ErrMalformedArtifact  = errors.New("malformed OpenSpec artifact")
	ErrIntentNotConfirmed = errors.New("OpenSpec intent is not confirmed")
	ErrContextRequired    = errors.New("OpenSpec context is required")
)

const goalrailIntentSchema = "goalrail-intent"

type CompiledChange struct {
	Intent   domain.IntentSnapshot
	Proposal domain.Proposal
	Pair     *ConformedPair
}

// ReadContext parses the bounded evidence artifact introduced by schema v2.
// It retains claims and source references, never raw source bodies.
func ReadContext(reader io.Reader) (domain.ContextPack, error) {
	document, err := readMarkdownDocument(reader)
	if err != nil {
		return domain.ContextPack{}, err
	}
	metadata, err := parseBoldMetadata(document.preamble)
	if err != nil {
		return domain.ContextPack{}, err
	}
	version, err := parseUint32Metadata(metadata, "version")
	if err != nil {
		return domain.ContextPack{}, err
	}
	previousVersion, err := parseOptionalUint32Metadata(metadata, "previous version")
	if err != nil {
		return domain.ContextPack{}, err
	}
	startedAt, err := parseArtifactTime(cleanInline(metadata["started at"]))
	if err != nil {
		return domain.ContextPack{}, err
	}
	completedAt, err := parseArtifactTime(cleanInline(metadata["completed at"]))
	if err != nil {
		return domain.ContextPack{}, err
	}

	itemLines, err := document.requiredSection("Context Items")
	if err != nil {
		return domain.ContextPack{}, err
	}
	unknownLines, err := document.requiredSection("Material Unknowns")
	if err != nil {
		return domain.ContextPack{}, err
	}
	items, err := parseContextItems(itemLines)
	if err != nil {
		return domain.ContextPack{}, err
	}
	unknowns, err := parseContextUnknowns(unknownLines)
	if err != nil {
		return domain.ContextPack{}, err
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
		return domain.ContextPack{}, fmt.Errorf("validate OpenSpec context: %w", err)
	}
	return pack, nil
}

// LoadChange reads the optional v2 context.md before intent.md, stops before
// proposal.md when intent is not confirmed, then validates the proposal
// exclusively through canonical domain rules. A missing context.md is accepted
// only for legacy schema-v1 changes.
func LoadChange(changeDir string) (CompiledChange, error) {
	contextPath := filepath.Join(changeDir, "context.md")
	var rawContext []byte
	if _, err := os.Lstat(contextPath); err == nil {
		rawContext, err = readPairArtifactFile(contextPath, "context.md", ArtifactKindContext)
		if err != nil {
			return CompiledChange{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return CompiledChange{}, inputUnavailableDiagnostic("context.md", ArtifactKindContext, "unreadable input", err)
	} else {
		required, requirementErr := changeRequiresContext(changeDir)
		if requirementErr != nil {
			return CompiledChange{}, requirementErr
		}
		if required {
			return CompiledChange{}, inputUnavailableDiagnostic(
				"context.md", ArtifactKindContext, "missing required input", ErrContextRequired,
			)
		}
	}

	intentPath := filepath.Join(changeDir, "intent.md")
	rawIntent, err := readPairArtifactFile(intentPath, "intent.md", ArtifactKindIntent)
	if err != nil {
		if rawContext != nil {
			return CompiledChange{}, err
		}
		return CompiledChange{}, fmt.Errorf("open OpenSpec intent: %w", err)
	}
	var pair *ConformedPair
	var intent domain.IntentSnapshot
	if rawContext != nil {
		conformed, conformErr := ConformPair(rawContext, rawIntent, "context.md", "intent.md")
		if conformErr != nil {
			return CompiledChange{}, conformErr
		}
		pair = &conformed
		intent = conformed.Intent
	} else {
		intent, err = ReadIntent(bytes.NewReader(rawIntent))
		if err != nil {
			return CompiledChange{}, err
		}
	}
	if intent.Status != domain.IntentConfirmed {
		return CompiledChange{}, fmt.Errorf("%w: status is %q", ErrIntentNotConfirmed, intent.Status)
	}

	proposalFile, err := os.Open(filepath.Join(changeDir, "proposal.md"))
	if err != nil {
		return CompiledChange{}, fmt.Errorf("open OpenSpec proposal: %w", err)
	}
	proposal, readErr := ReadProposal(proposalFile)
	closeErr := proposalFile.Close()
	if readErr != nil {
		return CompiledChange{}, readErr
	}
	if closeErr != nil {
		return CompiledChange{}, fmt.Errorf("close OpenSpec proposal: %w", closeErr)
	}
	if err := domain.ValidateProposalCoverage(intent, proposal); err != nil {
		return CompiledChange{}, fmt.Errorf("validate OpenSpec proposal coverage: %w", err)
	}
	return CompiledChange{Intent: intent, Proposal: proposal, Pair: pair}, nil
}

func changeRequiresContext(changeDir string) (bool, error) {
	schema, err := readChangeSchema(filepath.Join(changeDir, ".openspec.yaml"))
	if err != nil || schema != goalrailIntentSchema {
		return false, err
	}
	if !isArchivedChange(changeDir) {
		return true, nil
	}

	intentFile, err := os.Open(filepath.Join(changeDir, "intent.md"))
	if err != nil {
		return false, fmt.Errorf("open archived OpenSpec intent metadata: %w", err)
	}
	document, readErr := readMarkdownDocument(intentFile)
	closeErr := intentFile.Close()
	if readErr != nil {
		return false, readErr
	}
	if closeErr != nil {
		return false, fmt.Errorf("close archived OpenSpec intent metadata: %w", closeErr)
	}
	metadata, err := parseBoldMetadata(document.preamble)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(metadata["context pack"]) != "", nil
}

func readChangeSchema(path string) (string, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("open OpenSpec change metadata: %w", err)
	}
	defer file.Close()

	var schema string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(key) != "schema" {
			continue
		}
		if schema != "" {
			return "", fmt.Errorf("%w: duplicate schema in .openspec.yaml", ErrMalformedArtifact)
		}
		parsed, parseErr := changeSchemaScalar(value)
		if parseErr != nil {
			return "", fmt.Errorf("%w: %v", ErrMalformedArtifact, parseErr)
		}
		schema = parsed
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read OpenSpec change metadata: %w", err)
	}
	if schema == "" {
		return "", fmt.Errorf("%w: .openspec.yaml has no schema", ErrMalformedArtifact)
	}
	return schema, nil
}

// changeSchemaScalar applies the same YAML subset as the harness configuration
// reader: a quoted value ends at its matching quote, while an unquoted comment
// begins only at the value or after whitespace. Keeping this treatment here
// prevents a commented pin from disappearing from adoption counts.
func changeSchemaScalar(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return "", nil
	}
	if value[0] == '\'' || value[0] == '"' {
		quote := value[0]
		end := strings.IndexByte(value[1:], quote)
		if end < 0 {
			return "", fmt.Errorf("unterminated quoted schema")
		}
		rest := strings.TrimSpace(value[2+end:])
		if rest != "" && !strings.HasPrefix(rest, "#") {
			return "", fmt.Errorf("content follows quoted schema")
		}
		return value[1 : 1+end], nil
	}
	for index := 0; index < len(value); index++ {
		if value[index] != '#' || (index > 0 && value[index-1] != ' ' && value[index-1] != '\t') {
			continue
		}
		return strings.TrimSpace(value[:index]), nil
	}
	return value, nil
}

func isArchivedChange(changeDir string) bool {
	archiveDir := filepath.Dir(filepath.Clean(changeDir))
	changesDir := filepath.Dir(archiveDir)
	openspecDir := filepath.Dir(changesDir)
	return filepath.Base(archiveDir) == "archive" &&
		filepath.Base(changesDir) == "changes" &&
		filepath.Base(openspecDir) == "openspec"
}

// ReadIntent parses only the structured fields owned by the custom schema.
func ReadIntent(reader io.Reader) (domain.IntentSnapshot, error) {
	return readIntent(reader, nil)
}

func readIntent(reader io.Reader, contextPack *domain.ContextPack) (domain.IntentSnapshot, error) {
	document, err := readMarkdownDocument(reader)
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	metadata, err := parseBoldMetadata(document.preamble)
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	version, err := parseUint32Metadata(metadata, "version")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	previousVersion, err := parseOptionalUint32Metadata(metadata, "previous version")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	status := domain.IntentStatus(strings.ToLower(cleanInline(metadata["status"])))
	if contextPack != nil {
		declaredID, declaredVersion, declarationErr := parseContextPackDeclaration(metadata["context pack"])
		if declarationErr != nil {
			return domain.IntentSnapshot{}, declarationErr
		}
		if declaredID != contextPack.ID || declaredVersion != contextPack.Version {
			return domain.IntentSnapshot{}, fmt.Errorf(
				"%w: intent Context Pack %q version %d does not match context.md %q version %d",
				ErrMalformedArtifact,
				declaredID,
				declaredVersion,
				contextPack.ID,
				contextPack.Version,
			)
		}
	}

	sourceLines, err := document.requiredSection("Source Evidence")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	desiredLines, err := document.requiredSection("Desired Outcomes")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	nonGoalLines, err := document.requiredSection("Non-Goals")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	signalLines, err := document.requiredSection("Observable Success Signals")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	ambiguityLines, err := document.requiredSection("Ambiguities and Unknowns")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	confirmationLines, err := document.requiredSection("Confirmation")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}

	resolvedEscalation, err := parseEscalationResolution(metadata)
	if err != nil {
		return domain.IntentSnapshot{}, err
	}

	snapshot := domain.IntentSnapshot{
		ID:                 domain.IntentID(cleanInline(metadata["intent id"])),
		Version:            version,
		PreviousVersion:    previousVersion,
		Status:             status,
		ContextPack:        contextPack,
		ResolvedEscalation: resolvedEscalation,
	}
	if snapshot.SourceEvidence, err = parseSourceEvidence(sourceLines); err != nil {
		return domain.IntentSnapshot{}, err
	}
	// Both heading generations are legal: the canon of 2026-08-02 neutralized
	// "Confirmed wording" and "Confirmed boundary" because a heading must not
	// assert what the status may deny, and artifacts written earlier keep the
	// old headings forever.
	if snapshot.DesiredOutcomes, err = parseIntentItems(
		desiredLines,
		[]string{"ID", "Outcome", "Verification action", "Evidence"},
		1,
		3,
	); err != nil {
		if snapshot.DesiredOutcomes, err = parseIntentItems(
			desiredLines,
			[]string{"ID", "Confirmed wording", "Verification action", "Evidence"},
			1,
			3,
		); err != nil {
			return domain.IntentSnapshot{}, fmt.Errorf("parse desired outcomes: %w", err)
		}
	}
	if snapshot.NonGoals, err = parseIntentItems(
		nonGoalLines,
		[]string{"ID", "Boundary", "Evidence"},
		1,
		2,
	); err != nil {
		if snapshot.NonGoals, err = parseIntentItems(
			nonGoalLines,
			[]string{"ID", "Confirmed boundary", "Evidence"},
			1,
			2,
		); err != nil {
			return domain.IntentSnapshot{}, fmt.Errorf("parse non-goals: %w", err)
		}
	}
	if snapshot.SuccessSignals, err = parseIntentItems(
		signalLines,
		[]string{"ID", "Signal", "Measurement", "Evidence"},
		1,
		3,
	); err != nil {
		return domain.IntentSnapshot{}, fmt.Errorf("parse success signals: %w", err)
	}
	if snapshot.Ambiguities, err = parseAmbiguities(ambiguityLines); err != nil {
		return domain.IntentSnapshot{}, err
	}
	if snapshot.Confirmation, err = parseConfirmation(confirmationLines); err != nil {
		return domain.IntentSnapshot{}, err
	}
	if owner := cleanInline(metadata["owner"]); owner != "" && snapshot.Confirmation != nil && owner != snapshot.Confirmation.Owner {
		return domain.IntentSnapshot{}, fmt.Errorf("%w: intent owner and confirming owner differ", ErrMalformedArtifact)
	}
	if err := domain.ValidateIntentSnapshot(snapshot); err != nil {
		return domain.IntentSnapshot{}, fmt.Errorf("validate OpenSpec intent: %w", err)
	}
	return snapshot, nil
}

func parseContextPackDeclaration(value string) (domain.ContextPackID, uint32, error) {
	fields := strings.Fields(cleanInline(value))
	if len(fields) != 3 || !strings.EqualFold(fields[1], "version") {
		return "", 0, fmt.Errorf("%w: Context Pack metadata must be '<id> version <number>'", ErrMalformedArtifact)
	}
	id := domain.ContextPackID(strings.Trim(fields[0], "`"))
	version, err := strconv.ParseUint(strings.Trim(fields[2], "`"), 10, 32)
	if err != nil || version == 0 || !domain.IsCanonicalID(string(id)) {
		return "", 0, fmt.Errorf("%w: Context Pack metadata has an invalid ID or version", ErrMalformedArtifact)
	}
	return id, uint32(version), nil
}

// parseEscalationResolution reads the optional record naming the blocked run an
// intent version answers. Both fields travel together: a resolution without a
// disposition would say a question was answered without saying how, and a
// disposition without a resolution would name no question at all.
//
// Duplicate records need no check here — parseBoldMetadata already rejects a
// repeated metadata field.
func parseEscalationResolution(
	metadata map[string]string,
) (*domain.IntentEscalationResolution, error) {
	rawResolves := cleanInline(metadata["resolves"])
	rawDisposition := cleanInline(metadata["disposition"])
	if rawResolves == "" && rawDisposition == "" {
		return nil, nil
	}
	if rawResolves == "" || rawDisposition == "" {
		return nil, fmt.Errorf(
			"%w: Resolves and Disposition must be recorded together",
			ErrMalformedArtifact,
		)
	}
	fields := strings.Fields(rawResolves)
	if len(fields) != 3 || !strings.EqualFold(fields[1], "escalation") {
		return nil, fmt.Errorf(
			"%w: Resolves metadata must be '<reference> escalation <digest>', where the reference names a blocked run or a background question record",
			ErrMalformedArtifact,
		)
	}
	resolution := &domain.IntentEscalationResolution{
		ResolvedID:       strings.Trim(fields[0], "`"),
		EscalationDigest: strings.ToLower(strings.Trim(fields[2], "`")),
		Disposition:      domain.IntentDisposition(strings.ToLower(rawDisposition)),
	}
	return resolution, nil
}

// ReadProposal treats Intent Coverage rows as the compiled change list. Other
// proposal prose remains human-facing and cannot add canonical work silently.
func ReadProposal(reader io.Reader) (domain.Proposal, error) {
	document, err := readMarkdownDocument(reader)
	if err != nil {
		return domain.Proposal{}, err
	}
	coverageLines, err := document.requiredSection("Intent Coverage")
	if err != nil {
		return domain.Proposal{}, err
	}
	rows, err := parseMarkdownTable(
		coverageLines,
		[]string{"Proposed change", "Intent IDs", "Non-goal preserved"},
	)
	if err != nil {
		return domain.Proposal{}, fmt.Errorf("parse intent coverage: %w", err)
	}

	proposal := domain.Proposal{Changes: make([]domain.ProposalChange, 0, len(rows))}
	preserved := make(map[domain.IntentItemID]struct{})
	for _, row := range rows {
		summary := cleanInline(row[0])
		change := domain.ProposalChange{
			ID:         proposalChangeID(summary),
			Summary:    summary,
			IntentRefs: parseIntentRefs(row[1]),
		}
		proposal.Changes = append(proposal.Changes, change)
		for _, reference := range parseIntentRefs(row[2]) {
			if _, exists := preserved[reference]; exists {
				continue
			}
			preserved[reference] = struct{}{}
			proposal.PreservedNonGoalRefs = append(proposal.PreservedNonGoalRefs, reference)
		}
	}
	return proposal, nil
}

type markdownDocument struct {
	preamble []string
	sections map[string][]string
}

func readMarkdownDocument(reader io.Reader) (markdownDocument, error) {
	document := markdownDocument{sections: make(map[string][]string)}
	currentSection := ""
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if currentSection == "" {
				return markdownDocument{}, fmt.Errorf("%w: empty section heading", ErrMalformedArtifact)
			}
			if _, exists := document.sections[currentSection]; exists {
				return markdownDocument{}, fmt.Errorf("%w: duplicate section %q", ErrMalformedArtifact, currentSection)
			}
			document.sections[currentSection] = nil
			continue
		}
		if currentSection == "" {
			document.preamble = append(document.preamble, line)
		} else {
			document.sections[currentSection] = append(document.sections[currentSection], line)
		}
	}
	if err := scanner.Err(); err != nil {
		return markdownDocument{}, fmt.Errorf("read OpenSpec artifact: %w", err)
	}
	return document, nil
}

func (document markdownDocument) requiredSection(name string) ([]string, error) {
	lines, exists := document.sections[name]
	if !exists {
		return nil, fmt.Errorf("%w: required section %q is missing", ErrMalformedArtifact, name)
	}
	return lines, nil
}

func parseBoldMetadata(lines []string) (map[string]string, error) {
	metadata := make(map[string]string)
	for _, line := range lines {
		key, value, ok := parseBoldBullet(line)
		if !ok {
			continue
		}
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if _, exists := metadata[normalizedKey]; exists {
			return nil, fmt.Errorf("%w: duplicate metadata field %q", ErrMalformedArtifact, key)
		}
		metadata[normalizedKey] = strings.TrimSpace(value)
	}
	return metadata, nil
}

func parseBoldBullet(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "- **") {
		return "", "", false
	}
	body := strings.TrimPrefix(trimmed, "- **")
	separator := strings.Index(body, ":**")
	if separator < 0 {
		return "", "", false
	}
	return strings.TrimSpace(body[:separator]), strings.TrimSpace(body[separator+3:]), true
}

func parseUint32Metadata(metadata map[string]string, key string) (uint32, error) {
	value := cleanInline(metadata[key])
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("%w: metadata %q must be a positive integer", ErrMalformedArtifact, key)
	}
	return uint32(parsed), nil
}

func parseOptionalUint32Metadata(metadata map[string]string, key string) (uint32, error) {
	value := cleanPending(metadata[key])
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("%w: metadata %q must be a positive integer", ErrMalformedArtifact, key)
	}
	return uint32(parsed), nil
}

func parseSourceEvidence(lines []string) ([]domain.SourceEvidence, error) {
	var result []domain.SourceEvidence
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		key, statement, ok := parseBoldBullet(line)
		if !ok || !strings.HasPrefix(strings.ToUpper(key), "SE-") {
			return nil, fmt.Errorf("%w: source evidence must use a structured SE-* bullet", ErrMalformedArtifact)
		}
		identifier, label, found := strings.Cut(key, " — ")
		if !found {
			return nil, fmt.Errorf("%w: source evidence %q lacks a kind label", ErrMalformedArtifact, key)
		}
		kind, err := sourceEvidenceKind(label)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.SourceEvidence{
			ID:        domain.SourceEvidenceID(cleanInline(identifier)),
			Kind:      kind,
			Statement: cleanInline(statement),
		})
	}
	return result, nil
}

func parseContextItems(lines []string) ([]domain.ContextItem, error) {
	// Two shapes are legal: the original six columns, and the canon of
	// 2026-08-02 which adds a Verification recipe column. The reader accepts
	// both because artifacts already written do not change shape when the
	// template does — rejecting either side would strand half the repositories.
	rows, err := parseMarkdownTable(
		lines,
		[]string{"ID", "Kind", "Claim", "Source", "Verification recipe", "Observed at", "Relevance"},
	)
	recipeIndex, observedIndex, relevanceIndex := 4, 5, 6
	if err != nil {
		rows, err = parseMarkdownTable(
			lines,
			[]string{"ID", "Kind", "Claim", "Source", "Observed at", "Relevance"},
		)
		recipeIndex, observedIndex, relevanceIndex = -1, 4, 5
	}
	if err != nil {
		return nil, fmt.Errorf("parse context items: %w", err)
	}
	items := make([]domain.ContextItem, 0, len(rows))
	for _, row := range rows {
		recipe := ""
		if recipeIndex >= 0 {
			recipe = cleanInline(row[recipeIndex])
			if recipe == "" {
				return nil, fmt.Errorf("%w: context item verification recipe is required", ErrMalformedArtifact)
			}
		}
		observedAt, parseErr := parseArtifactTime(cleanInline(row[observedIndex]))
		if parseErr != nil {
			return nil, parseErr
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

func parseContextUnknowns(lines []string) ([]domain.ContextUnknown, error) {
	meaningful := meaningfulArtifactLines(lines)
	if len(meaningful) == 1 && strings.EqualFold(strings.TrimSuffix(meaningful[0], "."), "none") {
		return nil, nil
	}
	if len(meaningful) == 0 {
		return nil, fmt.Errorf("%w: material unknowns must explicitly state None or list rows", ErrMalformedArtifact)
	}
	for _, line := range meaningful {
		if !strings.HasPrefix(line, "|") {
			return nil, fmt.Errorf("%w: material unknowns must be a structured table or explicit None", ErrMalformedArtifact)
		}
	}
	rows, err := parseMarkdownTable(meaningful, []string{"ID", "Question", "Sources"})
	if err != nil {
		return nil, fmt.Errorf("parse material unknowns: %w", err)
	}
	unknowns := make([]domain.ContextUnknown, 0, len(rows))
	for _, row := range rows {
		unknowns = append(unknowns, domain.ContextUnknown{
			ID:         domain.ContextUnknownID(cleanInline(row[0])),
			Question:   cleanInline(row[1]),
			SourceRefs: parseSourceRefs(row[2]),
		})
	}
	return unknowns, nil
}

func sourceEvidenceKind(label string) (domain.SourceEvidenceKind, error) {
	normalized := strings.ToLower(strings.TrimSpace(label))
	if strings.HasPrefix(normalized, "owner ") || normalized == "owner" {
		return domain.EvidenceOwnerStatement, nil
	}
	if strings.Contains(normalized, "repository") || strings.Contains(normalized, "project") {
		return domain.EvidenceRepositoryFact, nil
	}
	return "", fmt.Errorf("%w: unsupported source evidence kind %q", ErrMalformedArtifact, label)
}

func parseIntentItems(lines, headers []string, statementColumn, evidenceColumn int) ([]domain.IntentItem, error) {
	rows, err := parseMarkdownTable(lines, headers)
	if err != nil {
		return nil, err
	}
	items := make([]domain.IntentItem, 0, len(rows))
	for _, row := range rows {
		evidenceRefs, contextRefs := parseProvenanceRefs(row[evidenceColumn])
		items = append(items, domain.IntentItem{
			ID:           domain.IntentItemID(cleanInline(row[0])),
			Statement:    cleanInline(row[statementColumn]),
			EvidenceRefs: evidenceRefs,
			ContextRefs:  contextRefs,
		})
	}
	return items, nil
}

func parseAmbiguities(lines []string) ([]domain.IntentAmbiguity, error) {
	meaningful := meaningfulArtifactLines(lines)
	if len(meaningful) == 1 && strings.EqualFold(strings.TrimSuffix(meaningful[0], "."), "none") {
		return nil, nil
	}
	if len(meaningful) == 0 {
		return nil, fmt.Errorf("%w: ambiguities section must explicitly state None or list rows", ErrMalformedArtifact)
	}
	for _, line := range meaningful {
		if !strings.HasPrefix(line, "|") {
			return nil, fmt.Errorf("%w: ambiguities must be a structured table or explicit None", ErrMalformedArtifact)
		}
	}
	rows, err := parseMarkdownTable(meaningful, []string{"ID", "Question", "Evidence"})
	if err != nil {
		return nil, fmt.Errorf("parse ambiguities: %w", err)
	}
	ambiguities := make([]domain.IntentAmbiguity, 0, len(rows))
	for _, row := range rows {
		ambiguities = append(ambiguities, domain.IntentAmbiguity{
			ID:           domain.AmbiguityID(cleanInline(row[0])),
			Question:     cleanInline(row[1]),
			EvidenceRefs: parseEvidenceRefs(row[2]),
		})
	}
	return ambiguities, nil
}

func meaningfulArtifactLines(lines []string) []string {
	meaningful := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		meaningful = append(meaningful, trimmed)
	}
	return meaningful
}

func parseConfirmation(lines []string) (*domain.IntentConfirmation, error) {
	metadata, err := parseBoldMetadata(lines)
	if err != nil {
		return nil, err
	}
	owner := cleanPending(metadata["confirmed by"])
	confirmedAtText := cleanPending(metadata["confirmed at"])
	verification := cleanPending(metadata["verification action"])
	if owner == "" && confirmedAtText == "" && verification == "" {
		return nil, nil
	}
	confirmedAt, err := parseArtifactTime(confirmedAtText)
	if err != nil {
		return nil, err
	}
	return &domain.IntentConfirmation{
		Owner:              owner,
		ConfirmedAt:        confirmedAt,
		VerificationAction: verification,
	}, nil
}

func parseArtifactTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("%w: artifact timestamp %q is not ISO-8601", ErrMalformedArtifact, value)
}

func parseMarkdownTable(lines, expectedHeaders []string) ([][]string, error) {
	var blocks [][]string
	var current []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") {
			current = append(current, trimmed)
			continue
		}
		if len(current) > 0 {
			blocks = append(blocks, current)
			current = nil
		}
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	if len(blocks) == 0 {
		return nil, fmt.Errorf("%w: required Markdown table is missing", ErrMalformedArtifact)
	}

	var rows [][]string
	for _, block := range blocks {
		blockRows, err := parseMarkdownTableBlock(block, expectedHeaders)
		if err != nil {
			return nil, err
		}
		rows = append(rows, blockRows...)
	}
	return rows, nil
}

func parseMarkdownTableBlock(tableLines, expectedHeaders []string) ([][]string, error) {
	if len(tableLines) < 2 {
		return nil, fmt.Errorf("%w: incomplete Markdown table", ErrMalformedArtifact)
	}
	header, err := splitMarkdownRow(tableLines[0])
	if err != nil {
		return nil, err
	}
	if len(header) != len(expectedHeaders) {
		return nil, fmt.Errorf("%w: table has %d columns, want %d", ErrMalformedArtifact, len(header), len(expectedHeaders))
	}
	for index := range header {
		if normalizeHeader(header[index]) != normalizeHeader(expectedHeaders[index]) {
			return nil, fmt.Errorf("%w: table column %d is %q, want %q", ErrMalformedArtifact, index+1, header[index], expectedHeaders[index])
		}
	}
	separator, err := splitMarkdownRow(tableLines[1])
	if err != nil || len(separator) != len(header) || !isSeparatorRow(separator) {
		return nil, fmt.Errorf("%w: invalid Markdown table separator", ErrMalformedArtifact)
	}

	rows := make([][]string, 0, len(tableLines)-2)
	for _, line := range tableLines[2:] {
		row, err := splitMarkdownRow(line)
		if err != nil {
			return nil, err
		}
		if len(row) != len(header) {
			return nil, fmt.Errorf("%w: table row has %d columns, want %d", ErrMalformedArtifact, len(row), len(header))
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func splitMarkdownRow(line string) ([]string, error) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 2 || trimmed[0] != '|' || trimmed[len(trimmed)-1] != '|' {
		return nil, fmt.Errorf("%w: invalid Markdown table row", ErrMalformedArtifact)
	}
	body := strings.TrimSuffix(strings.TrimPrefix(trimmed, "|"), "|")
	const escapedPipe = "\x00"
	body = strings.ReplaceAll(body, `\|`, escapedPipe)
	parts := strings.Split(body, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(strings.ReplaceAll(parts[index], escapedPipe, "|"))
	}
	return parts, nil
}

func isSeparatorRow(row []string) bool {
	for _, cell := range row {
		trimmed := strings.Trim(strings.TrimSpace(cell), ":")
		if len(trimmed) < 3 || strings.Trim(trimmed, "-") != "" {
			return false
		}
	}
	return true
}

func normalizeHeader(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(cleanInline(value)), " "))
}

func parseEvidenceRefs(value string) []domain.SourceEvidenceID {
	tokens := splitReferenceTokens(value)
	result := make([]domain.SourceEvidenceID, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, domain.SourceEvidenceID(token))
	}
	return result
}

func parseSourceRefs(value string) []domain.EvidenceReference {
	tokens := splitReferenceTokens(value)
	result := make([]domain.EvidenceReference, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, domain.EvidenceReference(token))
	}
	return result
}

func parseProvenanceRefs(value string) ([]domain.SourceEvidenceID, []domain.ContextItemID) {
	tokens := splitReferenceTokens(value)
	evidenceRefs := make([]domain.SourceEvidenceID, 0, len(tokens))
	contextRefs := make([]domain.ContextItemID, 0, len(tokens))
	for _, token := range tokens {
		switch {
		case strings.HasPrefix(strings.ToUpper(token), "SE-"):
			evidenceRefs = append(evidenceRefs, domain.SourceEvidenceID(token))
		case strings.HasPrefix(strings.ToUpper(token), "CTX-"):
			contextRefs = append(contextRefs, domain.ContextItemID(token))
		default:
			evidenceRefs = append(evidenceRefs, domain.SourceEvidenceID(token))
		}
	}
	return evidenceRefs, contextRefs
}

func parseIntentRefs(value string) []domain.IntentItemID {
	tokens := splitReferenceTokens(value)
	result := make([]domain.IntentItemID, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, domain.IntentItemID(token))
	}
	return result
}

func splitReferenceTokens(value string) []string {
	cleaned := cleanInline(value)
	if cleaned == "" || strings.EqualFold(cleaned, "none") || strings.EqualFold(cleaned, "n/a") || cleaned == "—" || cleaned == "-" {
		return nil
	}
	rawTokens := strings.FieldsFunc(cleaned, func(character rune) bool {
		return character == ',' || character == ';' || unicode.IsSpace(character)
	})
	tokens := make([]string, 0, len(rawTokens))
	for _, token := range rawTokens {
		cleanedToken := strings.Trim(strings.TrimSpace(token), "`*_")
		if cleanedToken != "" {
			tokens = append(tokens, cleanedToken)
		}
	}
	return tokens
}

func proposalChangeID(summary string) domain.ProposalChangeID {
	digest := sha256.Sum256([]byte(strings.TrimSpace(summary)))
	return domain.ProposalChangeID("proposal-change-" + hex.EncodeToString(digest[:8]))
}

func cleanInline(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.Contains(trimmed, "<!--") {
		return ""
	}
	return strings.Trim(strings.TrimSpace(trimmed), "`")
}

func cleanPending(value string) string {
	cleaned := cleanInline(value)
	if strings.EqualFold(cleaned, "pending") {
		return ""
	}
	return cleaned
}
