package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type (
	WorkSpecID       string
	WorkSpecPosture  string
	WorkSpecCheckID  string
	StopConditionID  string
	WorkSpecDigest   string
	CheckResultState string
)

const (
	WorkSpecSchemaV0 = "goalrail.work-spec/v0"
	WorkSpecSchemaV1 = "goalrail.work-spec/v1"

	PostureTrustedLocalProviderEnforcedV0 WorkSpecPosture = "trusted-local-provider-enforced-v0"
	PostureTrustedLocalProviderEnforcedV1 WorkSpecPosture = "trusted-local-provider-enforced-v1"

	CheckResultPass        CheckResultState = "pass"
	CheckResultFail        CheckResultState = "fail"
	CheckResultUnavailable CheckResultState = "unavailable"
	CheckResultMissing     CheckResultState = "missing"

	MaxWorkSpecBytes       = 64 << 10
	MaxTaskBytes           = 4 << 10
	MaxTextBytes           = 1 << 10
	MaxReferenceBytes      = 512
	MaxPathBytes           = 512
	MaxWorkSpecPaths       = 64
	MaxWorkSpecChecks      = 32
	MaxStopConditions      = 32
	MaxCheckArguments      = 32
	MaxCheckArgumentBytes  = 512
	MaxEvidenceResultBytes = 512
)

var (
	hexDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	gitCommitPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
	secretPatterns   = []*regexp.Regexp{
		regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`),
		regexp.MustCompile(`(?i)(?:api[_-]?key|access[_-]?token|client[_-]?secret|password)\s*[:=]\s*\S+`),
		regexp.MustCompile(`(?:sk|ghp|github_pat|xox[baprs])[-_][A-Za-z0-9_-]{12,}`),
	}
)

// ClassifyLegacyWorkSpecBootstrap permits legacy authority only for an
// explicitly bounded set of migration/setup path prefixes. It never returns
// VALID and does not mutate or upgrade the legacy WorkSpec.
func ClassifyLegacyWorkSpecBootstrap(spec WorkSpec, allowedPathPrefixes []string) AdmissionClassification {
	if spec.Schema != WorkSpecSchemaV0 || len(allowedPathPrefixes) == 0 || len(allowedPathPrefixes) > MaxWorkSpecPaths {
		return AdmissionInvalid
	}
	for _, prefix := range allowedPathPrefixes {
		if validateRepositoryRelativePath(prefix) != nil {
			return AdmissionInvalid
		}
	}
	for _, scopedPath := range spec.Paths {
		allowed := false
		for _, prefix := range allowedPathPrefixes {
			if scopedPath == prefix || strings.HasPrefix(scopedPath, prefix+"/") {
				allowed = true
				break
			}
		}
		if !allowed {
			return AdmissionInvalid
		}
	}
	return AdmissionBootstrap
}

type WorkSpecRepository struct {
	Root         string `json:"root"`
	BaseRevision string `json:"base_revision"`
}

type WorkSpecIntentReference struct {
	ID          IntentID `json:"id"`
	Version     uint32   `json:"version"`
	ArtifactRef string   `json:"artifact_ref"`
	Digest      string   `json:"digest"`
}

// WorkSpecProjectReference binds a managed run to the committed project
// declaration that controls admission for the repository.
type WorkSpecProjectReference struct {
	ID          ProjectID    `json:"id"`
	ArtifactRef string       `json:"artifact_ref"`
	Digest      SHA256Digest `json:"digest"`
}

// WorkSpecPolicyReference binds a managed run to the exact committed policy
// revision used for admission.
type WorkSpecPolicyReference struct {
	ArtifactRef string       `json:"artifact_ref"`
	Digest      SHA256Digest `json:"digest"`
}

// WorkSpecChangeReference identifies the current Goalrail change and records
// the confirmed intent version that the change was compiled from.
type WorkSpecChangeReference struct {
	ID            string       `json:"id"`
	ArtifactRef   string       `json:"artifact_ref"`
	Digest        SHA256Digest `json:"digest"`
	IntentID      IntentID     `json:"intent_id"`
	IntentVersion uint32       `json:"intent_version"`
	IntentDigest  SHA256Digest `json:"intent_digest"`
}

// WorkSpecWorkUnitReference binds the immutable execution authority to one
// open repository-local work unit.
type WorkSpecWorkUnitReference struct {
	ID          WorkUnitID   `json:"id"`
	ArtifactRef string       `json:"artifact_ref"`
	Digest      SHA256Digest `json:"digest"`
}

type WorkSpecCheck struct {
	ID   WorkSpecCheckID `json:"id"`
	Argv []string        `json:"argv"`
}

type WorkSpecStopCondition struct {
	ID          StopConditionID `json:"id"`
	Description string          `json:"description"`
}

// WorkSpec is the complete provider-neutral semantic input for one trusted
// repository-local run. Provider configuration and session identity are
// deliberately absent.
type WorkSpec struct {
	Schema         string                     `json:"schema"`
	ID             WorkSpecID                 `json:"id"`
	Version        uint32                     `json:"version"`
	Project        *WorkSpecProjectReference  `json:"project,omitempty"`
	Policy         *WorkSpecPolicyReference   `json:"policy,omitempty"`
	Change         *WorkSpecChangeReference   `json:"change,omitempty"`
	WorkUnit       *WorkSpecWorkUnitReference `json:"work_unit,omitempty"`
	Repository     WorkSpecRepository         `json:"repository"`
	Intent         WorkSpecIntentReference    `json:"intent"`
	Task           string                     `json:"task"`
	Paths          []string                   `json:"paths"`
	Checks         []WorkSpecCheck            `json:"checks"`
	StopConditions []WorkSpecStopCondition    `json:"stop_conditions"`
	Posture        WorkSpecPosture            `json:"posture"`
}

// FrozenWorkSpec retains canonical bytes and their deterministic identity.
// Accessors return copies so callers cannot mutate the frozen value in place.
type FrozenWorkSpec struct {
	spec      WorkSpec
	canonical []byte
	digest    WorkSpecDigest
}

func DecodeWorkSpec(reader io.Reader) (WorkSpec, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, MaxWorkSpecBytes+1))
	if err != nil {
		return WorkSpec{}, fmt.Errorf("read WorkSpec: %w", err)
	}
	if len(raw) > MaxWorkSpecBytes {
		return WorkSpec{}, fmt.Errorf("decode WorkSpec: payload exceeds %d bytes", MaxWorkSpecBytes)
	}

	var spec WorkSpec
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&spec); err != nil {
		return WorkSpec{}, fmt.Errorf("decode WorkSpec: %w", err)
	}
	if err := requireWorkSpecJSONEOF(decoder); err != nil {
		return WorkSpec{}, fmt.Errorf("decode WorkSpec: %w", err)
	}
	return normalizeWorkSpec(spec), nil
}

func FreezeWorkSpec(spec WorkSpec) (FrozenWorkSpec, error) {
	spec = normalizeWorkSpec(spec)
	if err := ValidateWorkSpec(spec); err != nil {
		return FrozenWorkSpec{}, err
	}
	canonical, err := json.Marshal(spec)
	if err != nil {
		return FrozenWorkSpec{}, fmt.Errorf("encode canonical WorkSpec: %w", err)
	}
	sum := sha256.Sum256(canonical)
	digest := WorkSpecDigest("sha256:" + hex.EncodeToString(sum[:]))
	return FrozenWorkSpec{
		spec:      cloneWorkSpec(spec),
		canonical: append([]byte(nil), canonical...),
		digest:    digest,
	}, nil
}

func OpenFrozenWorkSpec(canonical []byte, expected WorkSpecDigest) (FrozenWorkSpec, error) {
	spec, err := DecodeWorkSpec(bytes.NewReader(canonical))
	if err != nil {
		return FrozenWorkSpec{}, err
	}
	frozen, err := FreezeWorkSpec(spec)
	if err != nil {
		return FrozenWorkSpec{}, err
	}
	if !bytes.Equal(canonical, frozen.canonical) {
		return FrozenWorkSpec{}, fmt.Errorf("stored WorkSpec is not canonical")
	}
	if frozen.digest != expected {
		return FrozenWorkSpec{}, fmt.Errorf("WorkSpec digest mismatch: expected %s, got %s", expected, frozen.digest)
	}
	return frozen, nil
}

func (frozen FrozenWorkSpec) Spec() WorkSpec {
	return cloneWorkSpec(frozen.spec)
}

func (frozen FrozenWorkSpec) CanonicalJSON() []byte {
	return append([]byte(nil), frozen.canonical...)
}

func (frozen FrozenWorkSpec) Digest() WorkSpecDigest {
	return frozen.digest
}

func ValidateWorkSpec(spec WorkSpec) error {
	v := &validator{}

	switch spec.Schema {
	case WorkSpecSchemaV0:
		validateWorkSpecV0Authority(v, spec)
	case WorkSpecSchemaV1:
		validateWorkSpecV1Authority(v, spec)
	default:
		v.add("work_spec.schema.invalid", "schema", "unsupported WorkSpec schema")
	}
	if !IsCanonicalID(string(spec.ID)) {
		v.add("work_spec.id.invalid", "id", "WorkSpec ID must be canonical")
	}
	if spec.Version == 0 {
		v.add("work_spec.version.invalid", "version", "version must be at least 1")
	}
	validateRepository(v, spec.Repository)
	validateIntentReference(v, spec.Intent)
	validateRetainedText(v, "task", spec.Task, MaxTaskBytes)
	validatePaths(v, spec.Paths)
	validateChecks(v, spec.Checks)
	validateStopConditions(v, spec.StopConditions)
	return v.result()
}

func validateWorkSpecV0Authority(v *validator, spec WorkSpec) {
	if spec.Project != nil || spec.Policy != nil || spec.Change != nil || spec.WorkUnit != nil {
		v.add("work_spec.v0.authority_forbidden", "schema", "v0 cannot carry managed-project authority bindings")
	}
	if spec.Posture != PostureTrustedLocalProviderEnforcedV0 {
		v.add("work_spec.posture.unsupported", "posture", "v0 accepts only trusted-local provider-enforced posture")
	}
}

func validateWorkSpecV1Authority(v *validator, spec WorkSpec) {
	if spec.Project == nil {
		v.add("work_spec.project.required", "project", "managed WorkSpec requires a project declaration reference")
	} else {
		validateWorkSpecProjectReference(v, *spec.Project)
	}
	if spec.Policy == nil {
		v.add("work_spec.policy.required", "policy", "managed WorkSpec requires a policy reference")
	} else {
		validateWorkSpecPolicyReference(v, *spec.Policy)
	}
	if spec.Change == nil {
		v.add("work_spec.change.required", "change", "managed WorkSpec requires a Goalrail change reference")
	} else {
		validateWorkSpecChangeReference(v, spec, *spec.Change)
	}
	if spec.WorkUnit == nil {
		v.add("work_spec.work_unit.required", "work_unit", "managed WorkSpec requires a work-unit reference")
	} else {
		validateWorkSpecWorkUnitReference(v, *spec.WorkUnit)
	}
	if spec.Posture != PostureTrustedLocalProviderEnforcedV1 {
		v.add("work_spec.posture.unsupported", "posture", "v1 accepts only trusted-local provider-enforced posture")
	}
}

func validateWorkSpecProjectReference(v *validator, reference WorkSpecProjectReference) {
	if !IsCanonicalID(string(reference.ID)) {
		v.add("work_spec.project.id_invalid", "project.id", "project ID must be canonical")
	}
	validateWorkSpecArtifactReference(v, "project", reference.ArtifactRef, reference.Digest)
}

func validateWorkSpecPolicyReference(v *validator, reference WorkSpecPolicyReference) {
	validateWorkSpecArtifactReference(v, "policy", reference.ArtifactRef, reference.Digest)
}

func validateWorkSpecChangeReference(v *validator, spec WorkSpec, reference WorkSpecChangeReference) {
	if !IsCanonicalID(reference.ID) {
		v.add("work_spec.change.id_invalid", "change.id", "change ID must be canonical")
	}
	validateWorkSpecArtifactReference(v, "change", reference.ArtifactRef, reference.Digest)
	if !IsCanonicalID(string(reference.IntentID)) {
		v.add("work_spec.change.intent_id_invalid", "change.intent_id", "change intent ID must be canonical")
	}
	if reference.IntentVersion == 0 {
		v.add("work_spec.change.intent_version_invalid", "change.intent_version", "change intent version must be at least 1")
	}
	validateDigestField(v, "work_spec.change.intent_digest_invalid", "change.intent_digest", reference.IntentDigest)
	if reference.IntentID != spec.Intent.ID || reference.IntentVersion != spec.Intent.Version || string(reference.IntentDigest) != spec.Intent.Digest {
		v.add("work_spec.change.intent_mismatch", "change", "change must bind the exact confirmed intent carried by the WorkSpec")
	}
}

func validateWorkSpecWorkUnitReference(v *validator, reference WorkSpecWorkUnitReference) {
	if !IsCanonicalID(string(reference.ID)) || !strings.HasPrefix(string(reference.ID), "wu_") {
		v.add("work_spec.work_unit.id_invalid", "work_unit.id", "work-unit ID must be canonical")
	}
	validateWorkSpecArtifactReference(v, "work_unit", reference.ArtifactRef, reference.Digest)
}

func validateWorkSpecArtifactReference(v *validator, prefix, artifactRef string, digest SHA256Digest) {
	if err := validateRepositoryRelativePath(artifactRef); err != nil {
		v.add("work_spec."+prefix+".artifact_ref_invalid", prefix+".artifact_ref", err.Error())
	}
	if len(artifactRef) > MaxReferenceBytes {
		v.add("work_spec."+prefix+".artifact_ref_too_large", prefix+".artifact_ref", "artifact reference exceeds the retention bound")
	}
	validateDigestField(v, "work_spec."+prefix+".digest_invalid", prefix+".digest", digest)
}

func validateRepository(v *validator, repository WorkSpecRepository) {
	if repository.Root == "" || !filepath.IsAbs(repository.Root) || strings.ContainsRune(repository.Root, '\x00') {
		v.add("work_spec.repository.root_invalid", "repository.root", "repository root must be a non-root absolute path")
	} else if filepath.Clean(repository.Root) == string(filepath.Separator) {
		v.add("work_spec.repository.root_invalid", "repository.root", "filesystem root cannot be a repository boundary")
	}
	if !gitCommitPattern.MatchString(repository.BaseRevision) {
		v.add("work_spec.repository.revision_invalid", "repository.base_revision", "base revision must be a full lowercase Git commit ID")
	}
}

func validateIntentReference(v *validator, reference WorkSpecIntentReference) {
	if !IsCanonicalID(string(reference.ID)) {
		v.add("work_spec.intent.id_invalid", "intent.id", "intent ID must be canonical")
	}
	if reference.Version == 0 {
		v.add("work_spec.intent.version_invalid", "intent.version", "intent version must be at least 1")
	}
	if err := validateRepositoryRelativePath(reference.ArtifactRef); err != nil {
		v.add("work_spec.intent.artifact_ref_invalid", "intent.artifact_ref", err.Error())
	}
	if len(reference.ArtifactRef) > MaxReferenceBytes {
		v.add("work_spec.intent.artifact_ref_too_large", "intent.artifact_ref", "artifact reference exceeds the retention bound")
	}
	if !hexDigestPattern.MatchString(reference.Digest) {
		v.add("work_spec.intent.digest_invalid", "intent.digest", "intent digest must be a lowercase SHA-256 reference")
	}
}

func validatePaths(v *validator, paths []string) {
	if len(paths) == 0 {
		v.add("work_spec.paths.required", "paths", "at least one scoped path is required")
	}
	if len(paths) > MaxWorkSpecPaths {
		v.add("work_spec.paths.too_many", "paths", "path count exceeds the v0 bound")
	}
	seen := make(map[string]struct{}, len(paths))
	for index, path := range paths {
		field := fmt.Sprintf("paths[%d]", index)
		if len(path) > MaxPathBytes {
			v.add("work_spec.path.too_large", field, "path exceeds the retention bound")
		}
		if err := validateRepositoryRelativePath(path); err != nil {
			v.add("work_spec.path.invalid", field, err.Error())
		}
		if _, exists := seen[path]; exists {
			v.add("work_spec.path.duplicate", field, "scoped paths must be unique")
		}
		seen[path] = struct{}{}
	}
}

func validateChecks(v *validator, checks []WorkSpecCheck) {
	if len(checks) == 0 {
		v.add("work_spec.checks.required", "checks", "at least one verification check is required")
	}
	if len(checks) > MaxWorkSpecChecks {
		v.add("work_spec.checks.too_many", "checks", "check count exceeds the v0 bound")
	}
	seen := make(map[WorkSpecCheckID]struct{}, len(checks))
	for index, check := range checks {
		path := fmt.Sprintf("checks[%d]", index)
		if !IsCanonicalID(string(check.ID)) {
			v.add("work_spec.check.id_invalid", path+".id", "check ID must be canonical")
		} else if _, exists := seen[check.ID]; exists {
			v.add("work_spec.check.id_duplicate", path+".id", "check IDs must be unique")
		}
		seen[check.ID] = struct{}{}
		if len(check.Argv) == 0 {
			v.add("work_spec.check.argv_required", path+".argv", "check argument vector cannot be empty")
		}
		if len(check.Argv) > MaxCheckArguments {
			v.add("work_spec.check.argv_too_many", path+".argv", "check argument count exceeds the v0 bound")
		}
		for argumentIndex, argument := range check.Argv {
			field := fmt.Sprintf("%s.argv[%d]", path, argumentIndex)
			validateRetainedText(v, field, argument, MaxCheckArgumentBytes)
		}
		if isShellCommand(check.Argv) {
			v.add("work_spec.check.shell_fragment", path+".argv", "checks must be direct argument vectors, not shell fragments")
		}
	}
}

func validateStopConditions(v *validator, conditions []WorkSpecStopCondition) {
	if len(conditions) == 0 {
		v.add("work_spec.stop_conditions.required", "stop_conditions", "at least one explicit stop condition is required")
	}
	if len(conditions) > MaxStopConditions {
		v.add("work_spec.stop_conditions.too_many", "stop_conditions", "stop-condition count exceeds the v0 bound")
	}
	seen := make(map[StopConditionID]struct{}, len(conditions))
	for index, condition := range conditions {
		path := fmt.Sprintf("stop_conditions[%d]", index)
		if !IsCanonicalID(string(condition.ID)) {
			v.add("work_spec.stop_condition.id_invalid", path+".id", "stop-condition ID must be canonical")
		} else if _, exists := seen[condition.ID]; exists {
			v.add("work_spec.stop_condition.id_duplicate", path+".id", "stop-condition IDs must be unique")
		}
		seen[condition.ID] = struct{}{}
		validateRetainedText(v, path+".description", condition.Description, MaxTextBytes)
	}
}

func validateRetainedText(v *validator, path, value string, maximum int) {
	if strings.TrimSpace(value) == "" {
		v.add("work_spec.text.required", path, "value cannot be empty")
		return
	}
	if !utf8.ValidString(value) {
		v.add("work_spec.text.invalid_utf8", path, "value must be valid UTF-8")
	}
	if len(value) > maximum {
		v.add("work_spec.text.too_large", path, "value exceeds the retention bound")
	}
	if hasUnsafeControl(value) {
		v.add("work_spec.text.control_character", path, "value contains an unsupported control character")
	}
	if hasSecretShapedContent(value) {
		v.add("work_spec.text.secret_shaped", path, "secret-shaped content cannot be retained")
	}
}

func hasSecretShapedContent(value string) bool {
	for _, pattern := range secretPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func normalizeWorkSpec(spec WorkSpec) WorkSpec {
	spec.Schema = strings.TrimSpace(spec.Schema)
	spec.ID = WorkSpecID(strings.TrimSpace(string(spec.ID)))
	spec.Repository.Root = filepath.Clean(strings.TrimSpace(spec.Repository.Root))
	spec.Repository.BaseRevision = strings.ToLower(strings.TrimSpace(spec.Repository.BaseRevision))
	spec.Intent.ID = IntentID(strings.TrimSpace(string(spec.Intent.ID)))
	spec.Intent.ArtifactRef = normalizeRelativePath(spec.Intent.ArtifactRef)
	spec.Intent.Digest = strings.ToLower(strings.TrimSpace(spec.Intent.Digest))
	if spec.Project != nil {
		reference := *spec.Project
		reference.ID = ProjectID(strings.TrimSpace(string(reference.ID)))
		reference.ArtifactRef = normalizeRelativePath(reference.ArtifactRef)
		reference.Digest = SHA256Digest(strings.ToLower(strings.TrimSpace(string(reference.Digest))))
		spec.Project = &reference
	}
	if spec.Policy != nil {
		reference := *spec.Policy
		reference.ArtifactRef = normalizeRelativePath(reference.ArtifactRef)
		reference.Digest = SHA256Digest(strings.ToLower(strings.TrimSpace(string(reference.Digest))))
		spec.Policy = &reference
	}
	if spec.Change != nil {
		reference := *spec.Change
		reference.ID = strings.TrimSpace(reference.ID)
		reference.ArtifactRef = normalizeRelativePath(reference.ArtifactRef)
		reference.Digest = SHA256Digest(strings.ToLower(strings.TrimSpace(string(reference.Digest))))
		reference.IntentID = IntentID(strings.TrimSpace(string(reference.IntentID)))
		reference.IntentDigest = SHA256Digest(strings.ToLower(strings.TrimSpace(string(reference.IntentDigest))))
		spec.Change = &reference
	}
	if spec.WorkUnit != nil {
		reference := *spec.WorkUnit
		reference.ID = WorkUnitID(strings.TrimSpace(string(reference.ID)))
		reference.ArtifactRef = normalizeRelativePath(reference.ArtifactRef)
		reference.Digest = SHA256Digest(strings.ToLower(strings.TrimSpace(string(reference.Digest))))
		spec.WorkUnit = &reference
	}
	spec.Task = normalizeText(spec.Task)
	spec.Posture = WorkSpecPosture(strings.TrimSpace(string(spec.Posture)))

	spec.Paths = append([]string(nil), spec.Paths...)
	for index := range spec.Paths {
		spec.Paths[index] = normalizeRelativePath(spec.Paths[index])
	}
	sort.Strings(spec.Paths)

	spec.Checks = append([]WorkSpecCheck(nil), spec.Checks...)
	for index := range spec.Checks {
		spec.Checks[index].ID = WorkSpecCheckID(strings.TrimSpace(string(spec.Checks[index].ID)))
		spec.Checks[index].Argv = append([]string(nil), spec.Checks[index].Argv...)
		for argumentIndex := range spec.Checks[index].Argv {
			spec.Checks[index].Argv[argumentIndex] = normalizeText(spec.Checks[index].Argv[argumentIndex])
		}
	}
	sort.Slice(spec.Checks, func(left, right int) bool {
		return spec.Checks[left].ID < spec.Checks[right].ID
	})

	spec.StopConditions = append([]WorkSpecStopCondition(nil), spec.StopConditions...)
	for index := range spec.StopConditions {
		spec.StopConditions[index].ID = StopConditionID(strings.TrimSpace(string(spec.StopConditions[index].ID)))
		spec.StopConditions[index].Description = normalizeText(spec.StopConditions[index].Description)
	}
	sort.Slice(spec.StopConditions, func(left, right int) bool {
		return spec.StopConditions[left].ID < spec.StopConditions[right].ID
	})
	return spec
}

func cloneWorkSpec(spec WorkSpec) WorkSpec {
	return normalizeWorkSpec(spec)
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
}

func normalizeRelativePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
}

func validateRepositoryRelativePath(value string) error {
	if value == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if strings.ContainsRune(value, '\x00') || strings.Contains(value, `\`) {
		return fmt.Errorf("path contains an unsupported character")
	}
	if filepath.IsAbs(value) {
		return fmt.Errorf("path must be repository-relative")
	}
	cleaned := normalizeRelativePath(value)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("path escapes the repository")
	}
	return nil
}

func hasUnsafeControl(value string) bool {
	for _, character := range value {
		if character < 0x20 && character != '\n' && character != '\t' {
			return true
		}
	}
	return false
}

func isShellCommand(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	switch strings.ToLower(filepath.Base(argv[0])) {
	case "sh", "bash", "zsh", "fish", "cmd", "cmd.exe", "powershell", "pwsh":
		return true
	default:
		return false
	}
}

func requireWorkSpecJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}
