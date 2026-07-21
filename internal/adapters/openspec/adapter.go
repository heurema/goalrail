// Package openspec translates the repository's intent-first OpenSpec artifacts
// into provider-neutral domain values. Markdown remains an adapter concern.
package openspec

import (
	"bufio"
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
)

type CompiledChange struct {
	Intent   domain.IntentSnapshot
	Proposal domain.Proposal
}

// LoadChange reads intent.md first, stops before proposal.md when intent is not
// confirmed, then validates the proposal exclusively through canonical domain
// rules.
func LoadChange(changeDir string) (CompiledChange, error) {
	intentFile, err := os.Open(filepath.Join(changeDir, "intent.md"))
	if err != nil {
		return CompiledChange{}, fmt.Errorf("open OpenSpec intent: %w", err)
	}
	intent, readErr := ReadIntent(intentFile)
	closeErr := intentFile.Close()
	if readErr != nil {
		return CompiledChange{}, readErr
	}
	if closeErr != nil {
		return CompiledChange{}, fmt.Errorf("close OpenSpec intent: %w", closeErr)
	}
	if intent.Status != domain.IntentConfirmed {
		return CompiledChange{}, fmt.Errorf("%w: status is %q", ErrIntentNotConfirmed, intent.Status)
	}

	proposalFile, err := os.Open(filepath.Join(changeDir, "proposal.md"))
	if err != nil {
		return CompiledChange{}, fmt.Errorf("open OpenSpec proposal: %w", err)
	}
	proposal, readErr := ReadProposal(proposalFile)
	closeErr = proposalFile.Close()
	if readErr != nil {
		return CompiledChange{}, readErr
	}
	if closeErr != nil {
		return CompiledChange{}, fmt.Errorf("close OpenSpec proposal: %w", closeErr)
	}
	if err := domain.ValidateProposalCoverage(intent, proposal); err != nil {
		return CompiledChange{}, fmt.Errorf("validate OpenSpec proposal coverage: %w", err)
	}
	return CompiledChange{Intent: intent, Proposal: proposal}, nil
}

// ReadIntent parses only the structured fields owned by the custom schema.
func ReadIntent(reader io.Reader) (domain.IntentSnapshot, error) {
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

	snapshot := domain.IntentSnapshot{
		ID:              domain.IntentID(cleanInline(metadata["intent id"])),
		Version:         version,
		PreviousVersion: previousVersion,
		Status:          status,
	}
	if snapshot.SourceEvidence, err = parseSourceEvidence(sourceLines); err != nil {
		return domain.IntentSnapshot{}, err
	}
	if snapshot.DesiredOutcomes, err = parseIntentItems(
		desiredLines,
		[]string{"ID", "Confirmed wording", "Verification action", "Evidence"},
		1,
		3,
	); err != nil {
		return domain.IntentSnapshot{}, fmt.Errorf("parse desired outcomes: %w", err)
	}
	if snapshot.NonGoals, err = parseIntentItems(
		nonGoalLines,
		[]string{"ID", "Confirmed boundary", "Evidence"},
		1,
		2,
	); err != nil {
		return domain.IntentSnapshot{}, fmt.Errorf("parse non-goals: %w", err)
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
	value := cleanInline(metadata[key])
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
		items = append(items, domain.IntentItem{
			ID:           domain.IntentItemID(cleanInline(row[0])),
			Statement:    cleanInline(row[statementColumn]),
			EvidenceRefs: parseEvidenceRefs(row[evidenceColumn]),
		})
	}
	return items, nil
}

func parseAmbiguities(lines []string) ([]domain.IntentAmbiguity, error) {
	meaningful := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		meaningful = append(meaningful, trimmed)
	}
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
	return time.Time{}, fmt.Errorf("%w: confirmation timestamp %q is not ISO-8601", ErrMalformedArtifact, value)
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
