package harness

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SchemaDifference is the structural part of an OpenSpec schema adoption.
// Instructions stay opaque: only the digest of their source span, with Git's
// CRLF transport difference normalized, is compared.
type SchemaDifference struct {
	Comparable          bool     `json:"comparable"`
	Reason              string   `json:"reason,omitempty"`
	OnlyInReplaced      []string `json:"only_in_replaced,omitempty"`
	OnlyInAdopted       []string `json:"only_in_adopted,omitempty"`
	DependenciesChanged []string `json:"dependencies_changed,omitempty"`
	InstructionsChanged []string `json:"instructions_changed,omitempty"`
}

// RulesSnapshot is the exact top-level rules block present at adoption time.
// Counted distinguishes an honest zero from a YAML shape the bounded reader
// cannot count confidently.
type RulesSnapshot struct {
	Present     bool   `json:"present"`
	HasRules    bool   `json:"has_rules"`
	Text        string `json:"text,omitempty"`
	Count       int    `json:"count,omitempty"`
	Counted     bool   `json:"counted"`
	Digest      string `json:"sha256"`
	CountReason string `json:"count_reason,omitempty"`
}

type schemaArtifact struct {
	id              string
	requires        []string
	instructionSpan []byte
}

type sourceLine struct {
	start int
	end   int
	body  string
}

// CompareRepositorySchemas compares repository-owned schema files only. A
// stock schema that exists only in the OpenSpec package is deliberately not
// reached through Node or a package-manager path.
func CompareRepositorySchemas(repositoryRoot, replacedSchema string) SchemaDifference {
	replacedPath, err := repositorySchemaPath(repositoryRoot, replacedSchema)
	if err != nil {
		return SchemaDifference{Reason: err.Error()}
	}
	adoptedPath, err := repositorySchemaPath(repositoryRoot, SchemaName)
	if err != nil {
		return SchemaDifference{Reason: err.Error()}
	}
	replaced, err := os.ReadFile(replacedPath)
	if errors.Is(err, fs.ErrNotExist) {
		return SchemaDifference{Reason: fmt.Sprintf("replaced schema %q is not a file in the repository", replacedSchema)}
	}
	if err != nil {
		return SchemaDifference{Reason: fmt.Sprintf("read replaced schema: %v", err)}
	}
	adopted, err := os.ReadFile(adoptedPath)
	if err != nil {
		return SchemaDifference{Reason: fmt.Sprintf("read adopted schema: %v", err)}
	}
	return compareSchemaBytes(replaced, adopted)
}

func repositorySchemaPath(repositoryRoot, name string) (string, error) {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return "", fmt.Errorf("schema name %q cannot identify a repository schema file", name)
	}
	return filepath.Join(repositoryRoot, "openspec", "schemas", name, "schema.yaml"), nil
}

func compareSchemaBytes(replaced, adopted []byte) SchemaDifference {
	left, err := readSchemaArtifacts(replaced)
	if err != nil {
		return SchemaDifference{Reason: "could not read replaced schema: " + err.Error()}
	}
	right, err := readSchemaArtifacts(adopted)
	if err != nil {
		return SchemaDifference{Reason: "could not read adopted schema: " + err.Error()}
	}

	difference := SchemaDifference{Comparable: true}
	leftByID := make(map[string]schemaArtifact, len(left))
	rightByID := make(map[string]schemaArtifact, len(right))
	for _, artifact := range left {
		leftByID[artifact.id] = artifact
	}
	for _, artifact := range right {
		rightByID[artifact.id] = artifact
	}
	for _, artifact := range left {
		other, present := rightByID[artifact.id]
		if !present {
			difference.OnlyInReplaced = append(difference.OnlyInReplaced, artifact.id)
			continue
		}
		if !sameStringSet(artifact.requires, other.requires) {
			difference.DependenciesChanged = append(difference.DependenciesChanged, artifact.id)
		}
		if instructionDigest(artifact.instructionSpan) != instructionDigest(other.instructionSpan) {
			difference.InstructionsChanged = append(difference.InstructionsChanged, artifact.id)
		}
	}
	for _, artifact := range right {
		if _, present := leftByID[artifact.id]; !present {
			difference.OnlyInAdopted = append(difference.OnlyInAdopted, artifact.id)
		}
	}
	return difference
}

// instructionDigest keeps the instruction opaque while removing the one
// transport-level difference Git may introduce across clones. CRLF and LF are
// the same YAML line break; treating them as prose changes would make a Windows
// checkout report every shared instruction as changed.
func instructionDigest(span []byte) string {
	return digestBytes(bytes.ReplaceAll(span, []byte("\r\n"), []byte("\n")))
}

func sameStringSet(left, right []string) bool {
	leftSet := make(map[string]struct{}, len(left))
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range left {
		leftSet[value] = struct{}{}
	}
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	if len(leftSet) != len(rightSet) {
		return false
	}
	for value := range leftSet {
		if _, present := rightSet[value]; !present {
			return false
		}
	}
	return true
}

func readSchemaArtifacts(raw []byte) ([]schemaArtifact, error) {
	lines := splitSourceLines(raw)
	artifactsLine := -1
	for index, line := range lines {
		indent, err := sourceIndent(line.body)
		if err != nil {
			return nil, err
		}
		if indent != 0 || ignorableYAMLLine(line.body) {
			continue
		}
		key, value, ok := yamlMapping(line.body)
		if !ok || key != "artifacts" {
			continue
		}
		if strings.TrimSpace(stripUnquotedComment(value)) != "" {
			return nil, fmt.Errorf("top-level artifacts must be a block sequence")
		}
		if artifactsLine >= 0 {
			return nil, fmt.Errorf("duplicate top-level artifacts key")
		}
		artifactsLine = index
	}
	if artifactsLine < 0 {
		return nil, fmt.Errorf("no top-level artifacts key")
	}

	blockEnd := len(lines)
	for index := artifactsLine + 1; index < len(lines); index++ {
		if ignorableYAMLLine(lines[index].body) {
			continue
		}
		indent, err := sourceIndent(lines[index].body)
		if err != nil {
			return nil, err
		}
		if indent == 0 {
			blockEnd = index
			break
		}
	}

	var starts []int
	for index := artifactsLine + 1; index < blockEnd; index++ {
		line := lines[index]
		if ignorableYAMLLine(line.body) {
			continue
		}
		indent, err := sourceIndent(line.body)
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimSpace(line.body)
		if indent == 2 && strings.HasPrefix(trimmed, "- ") {
			starts = append(starts, index)
			continue
		}
		if indent < 4 {
			return nil, fmt.Errorf("cannot follow artifacts indentation at line %d", index+1)
		}
	}
	if len(starts) == 0 {
		return nil, fmt.Errorf("artifacts contains no entries")
	}

	seen := make(map[string]bool, len(starts))
	artifacts := make([]schemaArtifact, 0, len(starts))
	for offset, start := range starts {
		end := blockEnd
		if offset+1 < len(starts) {
			end = starts[offset+1]
		}
		artifact, err := readSchemaArtifact(raw, lines, start, end)
		if err != nil {
			return nil, err
		}
		if seen[artifact.id] {
			return nil, fmt.Errorf("duplicate artifact id %q", artifact.id)
		}
		seen[artifact.id] = true
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func readSchemaArtifact(raw []byte, lines []sourceLine, start, end int) (schemaArtifact, error) {
	first := strings.TrimSpace(lines[start].body)
	key, value, ok := yamlMapping(strings.TrimSpace(strings.TrimPrefix(first, "- ")))
	if !ok || key != "id" {
		return schemaArtifact{}, fmt.Errorf("artifact at line %d does not start with id", start+1)
	}
	id, err := simpleYAMLScalar(value)
	if err != nil || id == "" {
		return schemaArtifact{}, fmt.Errorf("artifact at line %d has no readable id", start+1)
	}

	artifact := schemaArtifact{id: id}
	fieldStarts := make([]int, 0, 6)
	for index := start + 1; index < end; index++ {
		if ignorableYAMLLine(lines[index].body) {
			continue
		}
		indent, indentErr := sourceIndent(lines[index].body)
		if indentErr != nil {
			return schemaArtifact{}, indentErr
		}
		if indent == 4 {
			if _, _, field := yamlMapping(strings.TrimSpace(lines[index].body)); !field {
				return schemaArtifact{}, fmt.Errorf("cannot follow artifact field at line %d", index+1)
			}
			fieldStarts = append(fieldStarts, index)
			continue
		}
		if indent < 6 {
			return schemaArtifact{}, fmt.Errorf("cannot follow artifact indentation at line %d", index+1)
		}
		if len(fieldStarts) == 0 {
			return schemaArtifact{}, fmt.Errorf("nested artifact value has no field at line %d", index+1)
		}
	}

	instructionSeen := false
	requiresSeen := false
	for offset, fieldStart := range fieldStarts {
		fieldEnd := end
		if offset+1 < len(fieldStarts) {
			fieldEnd = fieldStarts[offset+1]
		}
		key, value, _ := yamlMapping(strings.TrimSpace(lines[fieldStart].body))
		switch key {
		case "requires":
			if requiresSeen {
				return schemaArtifact{}, fmt.Errorf("artifact %q has duplicate requires", id)
			}
			requiresSeen = true
			requires, parseErr := readRequires(lines, fieldStart, fieldEnd, value)
			if parseErr != nil {
				return schemaArtifact{}, fmt.Errorf("artifact %q requires: %w", id, parseErr)
			}
			artifact.requires = requires
		case "instruction":
			if instructionSeen {
				return schemaArtifact{}, fmt.Errorf("artifact %q has duplicate instruction", id)
			}
			instructionSeen = true
			if parseErr := validateInstruction(lines, fieldStart, fieldEnd, value); parseErr != nil {
				return schemaArtifact{}, fmt.Errorf("artifact %q instruction: %w", id, parseErr)
			}
			spanEnd := instructionSpanEnd(lines, fieldStart, fieldEnd, value)
			artifact.instructionSpan = append([]byte(nil), raw[lines[fieldStart].start:spanEnd]...)
		}
	}
	if !instructionSeen {
		return schemaArtifact{}, fmt.Errorf("artifact %q has no instruction", id)
	}
	return artifact, nil
}

func instructionSpanEnd(lines []sourceLine, start, end int, value string) int {
	trimmed := strings.TrimSpace(stripUnquotedComment(value))
	if !strings.HasPrefix(trimmed, ">") && !strings.HasPrefix(trimmed, "|") {
		return lines[start].end
	}
	last := end
	for last > start+1 {
		line := lines[last-1].body
		trimmedLine := strings.TrimSpace(line)
		indent, err := sourceIndent(line)
		if err != nil || indent > 4 || !strings.HasPrefix(trimmedLine, "#") {
			break
		}
		last--
	}
	return lines[last-1].end
}

func readRequires(lines []sourceLine, start, end int, value string) ([]string, error) {
	trimmed := strings.TrimSpace(stripUnquotedComment(value))
	if trimmed != "" {
		if trimmed == "[]" {
			return nil, nil
		}
		if !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
			return nil, fmt.Errorf("unsupported inline value")
		}
		inside := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		if inside == "" {
			return nil, nil
		}
		parts := strings.Split(inside, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			item, err := simpleYAMLScalar(part)
			if err != nil || item == "" {
				return nil, fmt.Errorf("unsupported inline sequence")
			}
			result = append(result, item)
		}
		return result, nil
	}

	var result []string
	for index := start + 1; index < end; index++ {
		if ignorableYAMLLine(lines[index].body) {
			continue
		}
		indent, err := sourceIndent(lines[index].body)
		if err != nil || indent != 6 {
			return nil, fmt.Errorf("cannot follow block sequence at line %d", index+1)
		}
		trimmed := strings.TrimSpace(lines[index].body)
		if !strings.HasPrefix(trimmed, "- ") {
			return nil, fmt.Errorf("cannot follow block sequence at line %d", index+1)
		}
		item, itemErr := simpleYAMLScalar(strings.TrimPrefix(trimmed, "- "))
		if itemErr != nil || item == "" {
			return nil, fmt.Errorf("unreadable dependency at line %d", index+1)
		}
		result = append(result, item)
	}
	return result, nil
}

func validateInstruction(lines []sourceLine, start, end int, value string) error {
	trimmed := strings.TrimSpace(stripUnquotedComment(value))
	if trimmed == "" {
		return fmt.Errorf("has no value")
	}
	block := strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "|")
	if block && !validBlockIndicator(trimmed) {
		return fmt.Errorf("has an unsupported block indicator")
	}
	for index := start + 1; index < end; index++ {
		if ignorableYAMLLine(lines[index].body) {
			continue
		}
		indent, err := sourceIndent(lines[index].body)
		if err != nil {
			return err
		}
		if !block || indent < 6 {
			return fmt.Errorf("cannot follow value at line %d", index+1)
		}
	}
	return nil
}

func validBlockIndicator(value string) bool {
	if len(value) == 0 || (value[0] != '>' && value[0] != '|') {
		return false
	}
	for _, character := range value[1:] {
		if character != '+' && character != '-' && (character < '1' || character > '9') {
			return false
		}
	}
	return true
}

// ReadRules reads the configuration without changing it.
func ReadRules(repositoryRoot string) (RulesSnapshot, error) {
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(ConfigPath)))
	if err != nil {
		return RulesSnapshot{}, err
	}
	return extractRules(raw), nil
}

func extractRules(raw []byte) RulesSnapshot {
	absent := RulesSnapshot{Counted: true, Digest: digestBytes(nil)}
	lines := splitSourceLines(raw)
	var starts []int
	for index, line := range lines {
		indent, err := sourceIndent(line.body)
		if err != nil || indent != 0 || ignorableYAMLLine(line.body) {
			continue
		}
		key, _, ok := yamlMapping(line.body)
		if ok && key == "rules" {
			starts = append(starts, index)
		}
	}
	if len(starts) == 0 {
		return absent
	}
	if len(starts) > 1 {
		var combined strings.Builder
		for _, start := range starts {
			text, _, _ := rulesSourceSpan(raw, lines, start)
			combined.WriteString(text)
		}
		text := combined.String()
		return RulesSnapshot{
			Present:     true,
			HasRules:    true,
			Text:        text,
			Counted:     false,
			Digest:      digestBytes([]byte(text)),
			CountReason: "multiple top-level rules keys cannot be counted confidently",
		}
	}

	start := starts[0]
	text, inline, end := rulesSourceSpan(raw, lines, start)
	snapshot := RulesSnapshot{Present: true, Text: text, Counted: true, Digest: digestBytes([]byte(text))}
	if inline != "" {
		switch inline {
		case "[]", "{}":
			return snapshot
		default:
			snapshot.HasRules = true
			snapshot.Counted = false
			snapshot.CountReason = "inline rules cannot be counted confidently"
			return snapshot
		}
	}

	blockScalarIndent := -1
	for index := start + 1; index < end; index++ {
		line := lines[index]
		if ignorableYAMLLine(line.body) {
			continue
		}
		indent, err := sourceIndent(line.body)
		if err != nil || indent == 0 {
			snapshot.HasRules = true
			snapshot.Counted = false
			snapshot.CountReason = fmt.Sprintf("rules indentation at line %d cannot be counted confidently", index+1)
			return snapshot
		}
		trimmed := strings.TrimSpace(line.body)
		if blockScalarIndent >= 0 {
			if indent > blockScalarIndent {
				continue
			}
			blockScalarIndent = -1
		}
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			snapshot.Count++
			snapshot.HasRules = true
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if strings.HasPrefix(value, ">") || strings.HasPrefix(value, "|") {
				blockScalarIndent = indent
			}
			continue
		}
		key, mapped, ok := yamlMapping(trimmed)
		if !ok || key == "" {
			snapshot.HasRules = true
			snapshot.Counted = false
			snapshot.CountReason = fmt.Sprintf("rules shape at line %d cannot be counted confidently", index+1)
			return snapshot
		}
		mapped = strings.TrimSpace(stripUnquotedComment(mapped))
		if mapped != "" && mapped != "[]" && mapped != "{}" {
			snapshot.HasRules = true
			snapshot.Counted = false
			snapshot.CountReason = fmt.Sprintf("inline rule sequence at line %d cannot be counted confidently", index+1)
			return snapshot
		}
	}
	return snapshot
}

func rulesSourceSpan(raw []byte, lines []sourceLine, start int) (string, string, int) {
	_, value, _ := yamlMapping(lines[start].body)
	inline := strings.TrimSpace(stripUnquotedComment(value))
	end := start + 1
	if inline == "" {
		end = len(lines)
		for index := start + 1; index < len(lines); index++ {
			line := lines[index]
			trimmed := strings.TrimSpace(line.body)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			indent, err := sourceIndent(line.body)
			if err != nil {
				end = index + 1
				continue
			}
			if indent == 0 {
				end = index
				break
			}
		}
		for end > start+1 {
			line := lines[end-1].body
			trimmed := strings.TrimSpace(line)
			indent, err := sourceIndent(line)
			if trimmed == "" || (err == nil && indent == 0 && strings.HasPrefix(trimmed, "#")) {
				end--
				continue
			}
			break
		}
	}
	spanEnd := lines[start].end
	if end > start {
		spanEnd = lines[end-1].end
	}
	text := string(raw[lines[start].start:spanEnd])
	return text, inline, end
}

func splitSourceLines(raw []byte) []sourceLine {
	if len(raw) == 0 {
		return nil
	}
	lines := make([]sourceLine, 0, bytes.Count(raw, []byte{'\n'})+1)
	start := 0
	for index, character := range raw {
		if character != '\n' {
			continue
		}
		body := raw[start:index]
		body = bytes.TrimSuffix(body, []byte{'\r'})
		lines = append(lines, sourceLine{start: start, end: index + 1, body: string(body)})
		start = index + 1
	}
	if start < len(raw) {
		body := bytes.TrimSuffix(raw[start:], []byte{'\r'})
		lines = append(lines, sourceLine{start: start, end: len(raw), body: string(body)})
	}
	return lines
}

func sourceIndent(line string) (int, error) {
	indent := 0
	for _, character := range line {
		switch character {
		case ' ':
			indent++
		case '\t':
			return 0, fmt.Errorf("tab indentation cannot be followed")
		default:
			return indent, nil
		}
	}
	return indent, nil
}

func ignorableYAMLLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

func yamlMapping(line string) (string, string, bool) {
	key, value, found := strings.Cut(line, ":")
	if !found {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	return key, value, true
}

func simpleYAMLScalar(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "#") {
		return "", nil
	}
	if value[0] == '\'' || value[0] == '"' {
		quote := value[0]
		end := strings.IndexByte(value[1:], quote)
		if end < 0 {
			return "", fmt.Errorf("unterminated quoted scalar")
		}
		rest := strings.TrimSpace(value[2+end:])
		if rest != "" && !strings.HasPrefix(rest, "#") {
			return "", fmt.Errorf("content follows quoted scalar")
		}
		return value[1 : 1+end], nil
	}
	return strings.TrimSpace(stripUnquotedComment(value)), nil
}

func stripUnquotedComment(value string) string {
	value = strings.TrimSpace(value)
	for index := 0; index < len(value); index++ {
		if value[index] != '#' || (index > 0 && value[index-1] != ' ' && value[index-1] != '\t') {
			continue
		}
		return strings.TrimSpace(value[:index])
	}
	return value
}

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
