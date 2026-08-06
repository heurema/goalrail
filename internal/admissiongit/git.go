package admissiongit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	openspecadapter "github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/admission"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/lineage"
	"github.com/heurema/goalrail/internal/localrun"
)

const (
	maxGitOutput           = 16 << 20
	maxGitPathList         = 4 << 20
	maxFrozenPaths         = domain.MaxAdmissionPaths
	maxReplicaBytes        = 1 << 20
	maxBootstrapBytes      = 64 << 10
	maxChangeMetadataBytes = 8 << 10
)

var errGitObjectMissing = errors.New("Git object path is missing")
var errDeclarationMissing = errors.New("project declaration is missing")

// CollectFrozenRange resolves only explicit immutable commit IDs and reads all
// governance and lineage bytes from Git objects. It never reads governed bytes
// from the checkout.
func CollectFrozenRange(ctx context.Context, repository, base, head string, packet domain.AdmissionPacket) (admission.FrozenRange, error) {
	if !fullGitOID(base) || !fullGitOID(head) || base == head {
		return admission.FrozenRange{}, fmt.Errorf("base and head must be distinct full lowercase Git commit IDs")
	}
	root, err := filepath.Abs(repository)
	if err != nil {
		return admission.FrozenRange{}, fmt.Errorf("resolve repository path: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return admission.FrozenRange{}, fmt.Errorf("resolve repository path: %w", err)
	}
	for _, revision := range []string{base, head} {
		if _, err := runGit(ctx, root, 1024, "cat-file", "-e", revision+"^{commit}"); err != nil {
			return admission.FrozenRange{}, fmt.Errorf("resolve immutable commit %s: %w", revision, err)
		}
	}

	baseDeclaration, baseDeclarationDigest, basePolicy, basePolicyDigest, baseErr := readGovernance(ctx, root, base)
	baseInvalid := baseErr != nil && !errors.Is(baseErr, errDeclarationMissing)
	headDeclaration, headDeclarationDigest, headPolicy, headPolicyDigest, err := readGovernance(ctx, root, head)
	headInvalid := err != nil
	changes, err := collectChanges(ctx, root, base, head)
	if err != nil {
		return admission.FrozenRange{}, err
	}
	commits, err := collectCommits(ctx, root, base, head)
	if err != nil {
		return admission.FrozenRange{}, err
	}
	graph, err := collectGraph(ctx, root, head, packet.WorkUnitRef)
	if err != nil {
		return admission.FrozenRange{}, fmt.Errorf("load frozen lineage graph: %w", err)
	}
	projections, err := collectProjections(ctx, root, head, graph)
	if err != nil {
		return admission.FrozenRange{}, err
	}
	frozen := admission.FrozenRange{
		BaseRevision: base, HeadRevision: head,
		BaseDeclaration: baseDeclaration, BaseDeclarationDigest: baseDeclarationDigest,
		BasePolicy: basePolicy, BasePolicyDigest: basePolicyDigest, BaseGovernanceInvalid: baseInvalid,
		HeadDeclaration: headDeclaration, HeadDeclarationDigest: headDeclarationDigest,
		HeadPolicy: headPolicy, HeadPolicyDigest: headPolicyDigest, HeadGovernanceInvalid: headInvalid,
		Changes: changes, Commits: commits, Graph: graph, Projections: projections,
	}
	return frozen, nil
}

// collectProjections parses the exact committed artifacts whose semantic state
// decides admission. It binds every projection to the raw bytes behind the
// lineage target digest, so a relation that merely exists cannot stand in for a
// confirmed intent or a passed run.
func collectProjections(ctx context.Context, repository, revision string, graph admission.WorkUnitGraph) ([]admission.SemanticProjection, error) {
	var projections []admission.SemanticProjection
	for _, event := range graph.Events {
		switch event.Relation {
		case domain.LineageConfirmedIntent:
			for _, target := range event.Targets {
				projection, err := projectIntent(ctx, repository, revision, target)
				if err != nil {
					return nil, err
				}
				projections = append(projections, projection)
			}
		case domain.LineageTerminalReceipt:
			for _, target := range event.Targets {
				projections = append(projections, projectTerminalReceipt(graph.Replicas[target.Digest], target))
			}
		}
	}
	return projections, nil
}

func projectIntent(ctx context.Context, repository, revision string, target domain.ContentAddressedEvidenceReference) (admission.SemanticProjection, error) {
	projection := admission.SemanticProjection{
		Relation: domain.LineageConfirmedIntent, Kind: admission.ProjectionKindIntent,
		Identity: target.Identity, Version: target.Version, Digest: target.Digest,
		CodecID: admission.IntentSnapshotCodecV1, State: admission.ProjectionStateInvalid,
	}
	if target.ArtifactKind != admission.ProjectionKindIntent || !strings.HasPrefix(target.SourceRef, "repo:root/") {
		return projection, nil
	}
	relative := strings.TrimPrefix(target.SourceRef, "repo:root/")
	raw, err := readGitObject(ctx, repository, revision, relative, openspecadapter.MaxResolvedIntentBytes)
	if errors.Is(err, errGitObjectMissing) {
		return projection, nil
	}
	if err != nil {
		return admission.SemanticProjection{}, err
	}
	if domain.DigestCanonicalJSON(raw) != target.Digest {
		return projection, nil
	}
	snapshot, err := readIntentArtifact(ctx, repository, revision, relative, raw)
	if err != nil {
		return projection, nil
	}
	if err := domain.ValidateIntentSnapshot(snapshot); err != nil {
		return projection, nil
	}
	if "intent:"+string(snapshot.ID) != target.Identity ||
		strconv.FormatUint(uint64(snapshot.Version), 10) != target.Version {
		return projection, nil
	}
	if snapshot.Status != domain.IntentConfirmed {
		projection.State = string(snapshot.Status)
		return projection, nil
	}
	if snapshot.Confirmation == nil || len(snapshot.Ambiguities) > 0 {
		return projection, nil
	}
	projection.State = admission.ProjectionStateConfirmed
	projection.Valid = true
	return projection, nil
}

// readIntentArtifact parses the exact intent bytes under the adapter's own
// contract. A change that carries a context pack is only readable as a pair, so
// the projection uses the same pair conformance the lineage anchor did.
func readIntentArtifact(ctx context.Context, repository, revision, intentPath string, raw []byte) (domain.IntentSnapshot, error) {
	changeDir := path.Dir(intentPath)
	contextPath := path.Join(changeDir, "context.md")
	contextRaw, err := readGitObject(ctx, repository, revision, contextPath, openspecadapter.MaxPairArtifactBytes)
	if errors.Is(err, errGitObjectMissing) {
		// An intent whose change requires context is only readable as a pair.
		// The single-artifact reader ignores the requirement and would accept
		// the document as a legacy snapshot, so deleting the required context at
		// head would satisfy confirmed intent — admission passing precisely
		// because evidence was removed.
		//
		// The requirement comes from the change's own schema, not from a line in
		// the intent: deleting the declaration row along with the context would
		// otherwise remove the evidence and the reason to expect it in one move.
		required, requirementErr := contextRequired(ctx, repository, revision, changeDir, raw)
		if requirementErr != nil {
			return domain.IntentSnapshot{}, requirementErr
		}
		if required {
			return domain.IntentSnapshot{}, fmt.Errorf(
				"the change at %s requires a context pack, but %s is absent at this revision", changeDir, contextPath)
		}
		return openspecadapter.ReadIntent(bytes.NewReader(raw))
	}
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	conformed, err := openspecadapter.ConformPair(contextRaw, raw, "context.md", "intent.md")
	if err != nil {
		return domain.IntentSnapshot{}, err
	}
	return conformed.Intent, nil
}

// contextRequired applies the planning adapter's own schema-aware rule to the
// immutable revision: a live change on the project schema always requires its
// context, and an archived one requires it when its intent still names a pack.
//
// Read from Git rather than from the checkout, like everything else on this
// path, and mirroring the adapter rather than inventing a second rule.
func contextRequired(ctx context.Context, repository, revision, changeDir string, intentRaw []byte) (bool, error) {
	schemaRaw, err := readGitObject(ctx, repository, revision, path.Join(changeDir, ".openspec.yaml"), maxChangeMetadataBytes)
	if errors.Is(err, errGitObjectMissing) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if changeSchema(schemaRaw) != openspecadapter.GoalrailIntentSchema {
		return false, nil
	}
	if !archivedChangeDir(changeDir) {
		return true, nil
	}
	return declaresContextPack(intentRaw), nil
}

// changeSchema reads the one scalar this path needs from the change metadata.
func changeSchema(raw []byte) string {
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, found := strings.Cut(trimmed, ":")
		if !found || strings.TrimSpace(key) != "schema" {
			continue
		}
		value = strings.TrimSpace(value)
		if index := strings.IndexAny(value, "#"); index >= 0 {
			value = strings.TrimSpace(value[:index])
		}
		return strings.Trim(value, "'\"")
	}
	return ""
}

func archivedChangeDir(changeDir string) bool {
	return path.Base(path.Dir(changeDir)) == "archive"
}

// declaresContextPack reports whether an intent artifact names a context pack
// in its own preamble.
//
// Read from the exact bytes rather than from the parsed snapshot: the
// single-artifact reader drops the declaration, so asking the parse whether a
// pack was declared always answers no — which is the gap this closes. Only the
// preamble is inspected, so a mention inside the body cannot make an intent
// unreadable.
func declaresContextPack(raw []byte) bool {
	const marker = "- **context pack:**"
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			return false
		}
		// Case-insensitive, because the metadata reader that decides the same
		// question elsewhere folds case and a spelling difference must not
		// change whether evidence is required.
		if !strings.HasPrefix(strings.ToLower(trimmed), marker) {
			continue
		}
		value := strings.TrimSpace(trimmed[len(marker):])
		value = strings.TrimSpace(strings.Trim(value, "*`"))
		return value != "" && !strings.EqualFold(value, "pending")
	}
	return false
}

func projectTerminalReceipt(replica []byte, target domain.ContentAddressedEvidenceReference) admission.SemanticProjection {
	projection := admission.SemanticProjection{
		Relation: domain.LineageTerminalReceipt, Kind: admission.ProjectionKindTerminalReceipt,
		Identity: target.Identity, Version: target.Version, Digest: target.Digest,
		CodecID: admission.TerminalReceiptCodecV1, State: admission.ProjectionStateInvalid,
	}
	if target.ArtifactKind != admission.ProjectionKindTerminalReceipt || len(replica) == 0 ||
		domain.DigestCanonicalJSON(replica) != target.Digest {
		return projection
	}
	receipt, err := localrun.DecodeTerminalReceipt(bytes.NewReader(replica))
	if err != nil || receipt.Schema != localrun.TerminalReceiptSchemaV1 {
		return projection
	}
	switch receipt.Status {
	case localrun.StatePassed:
		projection.State, projection.Valid = admission.ProjectionStatePassed, true
	case localrun.StateFailed, localrun.StateBlocked, localrun.StateUnlinked,
		localrun.StateLaunchFailed, localrun.StateLaunchAttemptedUnknown,
		localrun.StateVerificationIncomplete, localrun.StatePrepared,
		localrun.StateLaunchAttempted, localrun.StateAwaitingVerification:
		projection.State = string(receipt.Status)
	}
	return projection
}

func collectCommits(ctx context.Context, repository, base, head string) ([]string, error) {
	if _, err := runGit(ctx, repository, 1024, "merge-base", "--is-ancestor", base, head); err != nil {
		return nil, fmt.Errorf("base commit is not an ancestor of head")
	}
	raw, err := runGit(ctx, repository, 4096*65, "rev-list", "--reverse", "--topo-order", base+".."+head)
	if err != nil {
		return nil, fmt.Errorf("collect frozen commit set: %w", err)
	}
	lines := strings.Fields(string(raw))
	if len(lines) == 0 || len(lines) > 4096 {
		return nil, fmt.Errorf("frozen commit set is empty or exceeds the 4096-commit bound")
	}
	for _, revision := range lines {
		if !fullGitOID(revision) {
			return nil, fmt.Errorf("Git returned a non-canonical commit ID")
		}
	}
	return lines, nil
}

func readGovernance(ctx context.Context, repository, revision string) (*domain.ProjectDeclaration, domain.SHA256Digest, *domain.ProjectPolicy, domain.SHA256Digest, error) {
	declarationRaw, err := readGitObject(ctx, repository, revision, domain.ProjectDeclarationPath, domain.MaxProjectDeclarationBytes)
	if err != nil {
		if errors.Is(err, errGitObjectMissing) {
			return nil, "", nil, "", fmt.Errorf("%w: %v", errDeclarationMissing, err)
		}
		return nil, "", nil, "", err
	}
	declaration, err := domain.DecodeProjectDeclaration(bytes.NewReader(declarationRaw))
	if err != nil {
		return nil, "", nil, "", fmt.Errorf("decode project declaration: %w", err)
	}
	frozenDeclaration, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil || !bytes.Equal(declarationRaw, frozenDeclaration.CanonicalJSON()) {
		return nil, "", nil, "", fmt.Errorf("project declaration is not canonical JSON")
	}
	policyRaw, err := readGitObject(ctx, repository, revision, declaration.Policy.Path, domain.MaxPolicyBytes)
	if err != nil {
		return nil, "", nil, "", err
	}
	policy, err := domain.DecodeProjectPolicy(bytes.NewReader(policyRaw))
	if err != nil {
		return nil, "", nil, "", fmt.Errorf("decode project policy: %w", err)
	}
	frozenPolicy, err := domain.FreezeProjectPolicy(policy)
	if err != nil || !bytes.Equal(policyRaw, frozenPolicy.CanonicalJSON()) {
		return nil, "", nil, "", fmt.Errorf("project policy is not canonical JSON")
	}
	if declaration.Policy.Digest != frozenPolicy.Digest() || policy.ProjectID != declaration.ProjectID {
		return nil, "", nil, "", fmt.Errorf("project policy does not match its declaration")
	}
	if err := verifyDeclarationBoundArtifacts(ctx, repository, revision, declaration); err != nil {
		return nil, "", nil, "", err
	}
	return &declaration, frozenDeclaration.Digest(), &policy, frozenPolicy.Digest(), nil
}

// verifyDeclarationBoundArtifacts reads every remaining artifact the
// declaration binds. Policy alone is not the governance set: a drifted
// bootstrap or setup profile is a governance failure, never a silent fallback
// to unmanaged or head-defined authority.
func verifyDeclarationBoundArtifacts(ctx context.Context, repository, revision string, declaration domain.ProjectDeclaration) error {
	bootstrapRaw, err := readDeclaredArtifact(ctx, repository, revision, declaration.Bootstrap, domain.BootstrapSchemaV1, maxBootstrapBytes)
	if err != nil {
		return err
	}
	if len(bootstrapRaw) == 0 {
		return fmt.Errorf("project bootstrap guide is empty")
	}
	setupRaw, err := readDeclaredArtifact(ctx, repository, revision, declaration.SetupProfile, domain.SetupProfileSchemaV1, domain.MaxSetupProfileBytes)
	if err != nil {
		return err
	}
	profile, err := domain.DecodeSetupProfile(bytes.NewReader(setupRaw))
	if err != nil {
		return fmt.Errorf("decode setup profile: %w", err)
	}
	frozenProfile, err := domain.FreezeSetupProfile(profile)
	if err != nil || !bytes.Equal(setupRaw, frozenProfile.CanonicalJSON()) {
		return fmt.Errorf("setup profile is not canonical JSON")
	}
	if profile.ProjectID != declaration.ProjectID {
		return fmt.Errorf("setup profile project ID differs from the declaration")
	}
	return nil
}

func readDeclaredArtifact(ctx context.Context, repository, revision string, reference domain.CommittedArtifactReference, schema string, limit int) ([]byte, error) {
	if reference.Schema != schema {
		return nil, fmt.Errorf("declared %s artifact has schema %q", schema, reference.Schema)
	}
	if !validRepositoryPath(reference.Path) || len(reference.Path) > domain.MaxProjectArtifactPathBytes {
		return nil, fmt.Errorf("declared %s artifact path is unsafe", schema)
	}
	raw, err := readGitObject(ctx, repository, revision, reference.Path, limit)
	if err != nil {
		return nil, err
	}
	if domain.DigestCanonicalJSON(raw) != reference.Digest {
		return nil, fmt.Errorf("declared %s artifact digest mismatch at %s", schema, reference.Path)
	}
	return raw, nil
}

func collectChanges(ctx context.Context, repository, base, head string) ([]admission.ChangedPath, error) {
	raw, err := runGit(ctx, repository, maxGitOutput,
		"-c", "diff.algorithm=myers", "-c", "core.quotepath=false",
		"diff", "--raw", "-z", "--no-abbrev", "--find-renames=50%", "--no-ext-diff", base, head, "--")
	if err != nil {
		return nil, fmt.Errorf("collect frozen Git diff: %w", err)
	}
	tokens := bytes.Split(raw, []byte{0})
	if len(tokens) > 0 && len(tokens[len(tokens)-1]) == 0 {
		tokens = tokens[:len(tokens)-1]
	}
	changes := make([]admission.ChangedPath, 0, len(tokens)/2)
	for index := 0; index < len(tokens); {
		header := strings.Fields(string(tokens[index]))
		index++
		if len(header) != 5 || !strings.HasPrefix(header[0], ":") || index >= len(tokens) {
			return nil, fmt.Errorf("Git produced malformed raw diff metadata")
		}
		oldMode := strings.TrimPrefix(header[0], ":")
		newMode, oldObject, newObject, status := header[1], header[2], header[3], header[4]
		changed := admission.ChangedPath{
			Path: string(tokens[index]), OldMode: oldMode, NewMode: newMode,
			OldObject: oldObject, NewObject: newObject,
		}
		index++
		if strings.HasPrefix(status, "R") {
			if index >= len(tokens) {
				return nil, fmt.Errorf("Git rename omits its destination path")
			}
			changed.PreviousPath = changed.Path
			changed.Path = string(tokens[index])
			changed.Kind = domain.ChangeRename
			index++
		} else {
			changed.Kind = classifyGitChange(status, oldMode, newMode)
		}
		if !validRepositoryPath(changed.Path) || (changed.PreviousPath != "" && !validRepositoryPath(changed.PreviousPath)) || changed.Kind == "" {
			return nil, fmt.Errorf("Git diff contains an unsupported path or change kind")
		}
		changes = append(changes, changed)
		if len(changes) > maxFrozenPaths {
			return nil, fmt.Errorf("Git diff exceeds the %d-path admission bound", maxFrozenPaths)
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Path != changes[j].Path {
			return changes[i].Path < changes[j].Path
		}
		return changes[i].PreviousPath < changes[j].PreviousPath
	})
	for index := range changes {
		arguments := []string{"rev-list", "--reverse", "--topo-order", base + ".." + head, "--", changes[index].Path}
		if changes[index].PreviousPath != "" {
			arguments = append(arguments, changes[index].PreviousPath)
		}
		rawCommits, err := runGit(ctx, repository, 4096*65, arguments...)
		if err != nil {
			return nil, fmt.Errorf("collect path commit set for %s: %w", changes[index].Path, err)
		}
		changes[index].Commits = strings.Fields(string(rawCommits))
		if len(changes[index].Commits) == 0 || len(changes[index].Commits) > 4096 {
			return nil, fmt.Errorf("path commit set for %s is empty or exceeds the bound", changes[index].Path)
		}
		for _, revision := range changes[index].Commits {
			if !fullGitOID(revision) {
				return nil, fmt.Errorf("Git returned a non-canonical path commit ID")
			}
		}
	}
	return changes, nil
}

func classifyGitChange(status, oldMode, newMode string) domain.ChangeKind {
	if oldMode == "160000" || newMode == "160000" {
		return domain.ChangeSubmodule
	}
	switch status {
	case "A":
		return domain.ChangeAdd
	case "D":
		return domain.ChangeDelete
	case "M":
		if oldMode != newMode {
			return domain.ChangeMode
		}
		return domain.ChangeModify
	case "T":
		return domain.ChangeMode
	default:
		return ""
	}
}

func collectGraph(ctx context.Context, repository, revision string, reference domain.ContentAddressedEvidenceReference) (admission.WorkUnitGraph, error) {
	const identityPrefix = "work-unit:"
	if reference.ArtifactKind != "work_unit" || !strings.HasPrefix(reference.Identity, identityPrefix) {
		return admission.WorkUnitGraph{}, fmt.Errorf("packet does not identify one work unit")
	}
	id := domain.WorkUnitID(strings.TrimPrefix(reference.Identity, identityPrefix))
	if !domain.IsCanonicalID(string(id)) || !strings.HasPrefix(string(id), "wu_") {
		return admission.WorkUnitGraph{}, fmt.Errorf("packet work-unit ID is not canonical")
	}
	unitPath := path.Join(".goalrail", "work-units", string(id), "unit.json")
	if reference.SourceRef != "repo:root/"+unitPath {
		return admission.WorkUnitGraph{}, fmt.Errorf("packet work-unit source does not name the canonical repository path")
	}
	unitRaw, err := readGitObject(ctx, repository, revision, unitPath, domain.MaxWorkUnitBytes)
	if err != nil {
		return admission.WorkUnitGraph{}, err
	}
	unit, err := domain.DecodeWorkUnit(bytes.NewReader(unitRaw))
	if err != nil {
		return admission.WorkUnitGraph{}, err
	}
	unitArtifact, err := domain.FreezeWorkUnit(unit)
	if err != nil || !bytes.Equal(unitRaw, unitArtifact.CanonicalJSON()) || unitArtifact.Digest() != reference.Digest || unit.ID != id {
		return admission.WorkUnitGraph{}, fmt.Errorf("frozen work-unit anchor does not match the packet")
	}

	eventDirectory := path.Join(".goalrail", "work-units", string(id), "events")
	listing, err := runGit(ctx, repository, maxGitPathList, "ls-tree", "-r", "-z", "--name-only", revision, "--", eventDirectory)
	if err != nil {
		return admission.WorkUnitGraph{}, err
	}
	paths := bytes.Split(listing, []byte{0})
	if len(paths) > 0 && len(paths[len(paths)-1]) == 0 {
		paths = paths[:len(paths)-1]
	}
	if len(paths) > domain.MaxLineageRelations*domain.MaxLineageReferences {
		return admission.WorkUnitGraph{}, fmt.Errorf("lineage event count exceeds the bounded store limit")
	}
	events := make([]domain.LineageEvent, 0, len(paths))
	for _, rawPath := range paths {
		eventPath := string(rawPath)
		if path.Dir(eventPath) != eventDirectory || path.Ext(eventPath) != ".json" || !validRepositoryPath(eventPath) {
			return admission.WorkUnitGraph{}, fmt.Errorf("lineage event tree contains an unsupported path")
		}
		raw, err := readGitObject(ctx, repository, revision, eventPath, domain.MaxLineageEventBytes)
		if err != nil {
			return admission.WorkUnitGraph{}, err
		}
		event, err := domain.DecodeLineageEvent(bytes.NewReader(raw))
		if err != nil {
			return admission.WorkUnitGraph{}, err
		}
		artifact, freezeErr := domain.FreezeLineageEvent(event)
		name := strings.TrimSuffix(path.Base(eventPath), ".json")
		if freezeErr != nil || !bytes.Equal(raw, artifact.CanonicalJSON()) || strings.TrimPrefix(string(event.SemanticDigest), "sha256:") != name || event.WorkUnitID != id {
			return admission.WorkUnitGraph{}, fmt.Errorf("lineage event %s is not canonical", eventPath)
		}
		if !eventCardinalityMatches(unit, event) {
			return admission.WorkUnitGraph{}, fmt.Errorf("lineage event cardinality differs from the work-unit contract")
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool { return events[i].SemanticDigest < events[j].SemanticDigest })
	return assembleGraph(ctx, repository, revision, unit, events)
}

func assembleGraph(ctx context.Context, repository, revision string, unit domain.WorkUnit, events []domain.LineageEvent) (admission.WorkUnitGraph, error) {
	graph := admission.WorkUnitGraph{Unit: unit, Events: events, Replicas: make(map[domain.SHA256Digest][]byte)}
	claims := make(map[domain.LineageRelation]map[string][]domain.SHA256Digest)
	unavailable := make(map[domain.LineageRelation]struct{})
	available := make(map[domain.LineageRelation]struct{})
	for _, event := range events {
		if event.Cardinality == domain.RelationSingle {
			canonicalTargets, err := json.Marshal(event.Targets)
			if err != nil {
				return admission.WorkUnitGraph{}, fmt.Errorf("canonicalize lineage targets: %w", err)
			}
			key := string(canonicalTargets)
			if claims[event.Relation] == nil {
				claims[event.Relation] = make(map[string][]domain.SHA256Digest)
			}
			claims[event.Relation][key] = append(claims[event.Relation][key], event.SemanticDigest)
		}
		for _, reference := range event.Targets {
			if reference.ArtifactKind == "provider_unavailable" {
				unavailable[event.Relation] = struct{}{}
				continue
			}
			if !strings.HasPrefix(reference.SourceRef, "repo:root/") {
				available[event.Relation] = struct{}{}
				continue
			}
			replicaPath, ok := repositoryReplicaPath(reference)
			if strings.HasPrefix(reference.SourceRef, "repo:root/.goalrail/evidence/sha256/") && !ok {
				return admission.WorkUnitGraph{}, fmt.Errorf("repository evidence reference does not match its digest")
			}
			if !ok {
				if err := verifyRepositoryReference(ctx, repository, revision, reference); errors.Is(err, errGitObjectMissing) {
					unavailable[event.Relation] = struct{}{}
					continue
				} else if err != nil {
					return admission.WorkUnitGraph{}, err
				}
				available[event.Relation] = struct{}{}
				continue
			}
			raw, err := readGitObject(ctx, repository, revision, replicaPath, maxReplicaBytes)
			if errors.Is(err, errGitObjectMissing) {
				unavailable[event.Relation] = struct{}{}
				continue
			}
			if err != nil {
				return admission.WorkUnitGraph{}, err
			}
			if domain.DigestCanonicalJSON(raw) != reference.Digest {
				return admission.WorkUnitGraph{}, fmt.Errorf("evidence replica digest mismatch at %s", replicaPath)
			}
			if err := validatePortableReplica(raw); err != nil {
				return admission.WorkUnitGraph{}, fmt.Errorf("validate evidence replica %s: %w", replicaPath, err)
			}
			graph.Replicas[reference.Digest] = raw
			available[event.Relation] = struct{}{}
		}
	}
	for relation, variants := range claims {
		if len(variants) < 2 {
			continue
		}
		var digests []domain.SHA256Digest
		for _, values := range variants {
			digests = append(digests, values...)
		}
		sort.Slice(digests, func(i, j int) bool { return digests[i] < digests[j] })
		graph.Conflicts = append(graph.Conflicts, admission.RelationConflict{Relation: relation, Digests: digests})
	}
	for relation := range unavailable {
		if _, ok := available[relation]; !ok {
			graph.Unavailable = append(graph.Unavailable, relation)
		}
	}
	sort.Slice(graph.Conflicts, func(i, j int) bool { return graph.Conflicts[i].Relation < graph.Conflicts[j].Relation })
	sort.Slice(graph.Unavailable, func(i, j int) bool { return graph.Unavailable[i] < graph.Unavailable[j] })
	return graph, nil
}

func verifyRepositoryReference(ctx context.Context, repository, revision string, reference domain.ContentAddressedEvidenceReference) error {
	relative := strings.TrimPrefix(reference.SourceRef, "repo:root/")
	if !validRepositoryPath(relative) {
		return fmt.Errorf("repository evidence path is invalid")
	}
	if reference.ArtifactKind == "change" {
		digest, err := digestGitChangeSnapshot(ctx, repository, revision, relative)
		if err != nil {
			return err
		}
		if digest != reference.Digest {
			return fmt.Errorf("change snapshot digest mismatch at %s", relative)
		}
		return nil
	}
	raw, err := readGitObject(ctx, repository, revision, relative, maxReplicaBytes)
	if err != nil {
		return err
	}
	if domain.DigestCanonicalJSON(raw) != reference.Digest {
		return fmt.Errorf("repository evidence digest mismatch at %s", relative)
	}
	return nil
}

type gitChangeSnapshotEntry struct {
	Path   string              `json:"path"`
	Size   int64               `json:"size"`
	Digest domain.SHA256Digest `json:"digest"`
}

type gitChangeSnapshotManifest struct {
	Schema  string                   `json:"schema"`
	Entries []gitChangeSnapshotEntry `json:"entries"`
}

func digestGitChangeSnapshot(ctx context.Context, repository, revision, prefix string) (domain.SHA256Digest, error) {
	raw, err := runGit(ctx, repository, maxGitPathList, "ls-tree", "-r", "-z", "--long", revision, "--", prefix)
	if err != nil {
		return "", err
	}
	records := bytes.Split(raw, []byte{0})
	if len(records) > 0 && len(records[len(records)-1]) == 0 {
		records = records[:len(records)-1]
	}
	if len(records) == 0 {
		return "", fmt.Errorf("%w: %s", errGitObjectMissing, prefix)
	}
	if len(records) > 512 {
		return "", fmt.Errorf("change snapshot exceeds 512 files")
	}
	manifest := gitChangeSnapshotManifest{Schema: "goalrail.change-snapshot/v1"}
	total := 0
	for _, record := range records {
		tab := bytes.IndexByte(record, '\t')
		if tab <= 0 {
			return "", fmt.Errorf("Git change snapshot entry is malformed")
		}
		metadata := strings.Fields(string(record[:tab]))
		objectPath := string(record[tab+1:])
		if len(metadata) != 4 || metadata[1] != "blob" || metadata[0] == "120000" || !validRepositoryPath(objectPath) || !strings.HasPrefix(objectPath, strings.TrimSuffix(prefix, "/")+"/") {
			return "", fmt.Errorf("change snapshot contains an unsupported Git entry")
		}
		content, err := readGitObject(ctx, repository, revision, objectPath, 1<<20)
		if err != nil {
			return "", err
		}
		total += len(content)
		if total > 8<<20 {
			return "", fmt.Errorf("change snapshot exceeds 8388608 bytes")
		}
		manifest.Entries = append(manifest.Entries, gitChangeSnapshotEntry{
			Path: strings.TrimPrefix(objectPath, strings.TrimSuffix(prefix, "/")+"/"),
			Size: int64(len(content)), Digest: domain.DigestCanonicalJSON(content),
		})
	}
	sort.Slice(manifest.Entries, func(i, j int) bool { return manifest.Entries[i].Path < manifest.Entries[j].Path })
	canonical, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return domain.DigestCanonicalJSON(canonical), nil
}

func validatePortableReplica(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("replica contains trailing JSON")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("replica must be one JSON object")
	}
	schema, ok := object["schema"].(string)
	if !ok {
		return fmt.Errorf("replica schema must be one string")
	}
	_, err := lineage.PrepareReplica(bytes.NewReader(raw), domain.DigestCanonicalJSON(raw), schema)
	return err
}

func repositoryReplicaPath(reference domain.ContentAddressedEvidenceReference) (string, bool) {
	prefix := "repo:root/.goalrail/evidence/sha256/"
	if !strings.HasPrefix(reference.SourceRef, prefix) {
		return "", false
	}
	want := strings.TrimPrefix(string(reference.Digest), "sha256:")
	if strings.TrimPrefix(reference.SourceRef, prefix) != want {
		return "", false
	}
	return strings.TrimPrefix(reference.SourceRef, "repo:root/"), true
}

func eventCardinalityMatches(unit domain.WorkUnit, event domain.LineageEvent) bool {
	for _, requirement := range unit.RequiredRelations {
		if requirement.Relation == event.Relation {
			return requirement.Cardinality == event.Cardinality
		}
	}
	return true
}

func readGitObject(ctx context.Context, repository, revision, objectPath string, limit int) ([]byte, error) {
	if !validRepositoryPath(objectPath) {
		return nil, fmt.Errorf("Git object path is not repository-relative")
	}
	spec := revision + ":" + objectPath
	if _, err := runGit(ctx, repository, 1024, "cat-file", "-e", spec); err != nil {
		return nil, fmt.Errorf("%w: %s", errGitObjectMissing, objectPath)
	}
	raw, err := runGit(ctx, repository, limit, "show", spec)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func runGit(ctx context.Context, repository string, limit int, arguments ...string) ([]byte, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("find git: %w", err)
	}
	base := []string{"-C", repository, "-c", "core.hooksPath=/dev/null", "-c", "filter.lfs.smudge=", "-c", "filter.lfs.required=false"}
	command := exec.CommandContext(ctx, gitPath, append(base, arguments...)...)
	command.Env = []string{
		"LC_ALL=C", "LANG=C", "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_TERMINAL_PROMPT=0", "GIT_NO_LAZY_FETCH=1", "GIT_OPTIONAL_LOCKS=0",
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	command.Stderr = &limitedWriter{writer: &stderr, remaining: 4096}
	if err := command.Start(); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	outputWriter := &limitedWriter{writer: &output, remaining: limit}
	_, readErr := io.Copy(outputWriter, stdout)
	waitErr := command.Wait()
	if readErr != nil {
		return nil, readErr
	}
	if outputWriter.overflow {
		return nil, fmt.Errorf("Git output exceeds the %d-byte bound", limit)
	}
	if waitErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = waitErr.Error()
		}
		return nil, fmt.Errorf("git command failed: %s", message)
	}
	return output.Bytes(), nil
}

type limitedWriter struct {
	writer    io.Writer
	remaining int
	overflow  bool
}

func (writer *limitedWriter) Write(value []byte) (int, error) {
	original := len(value)
	keep := 0
	if writer.remaining > 0 {
		keep = min(writer.remaining, len(value))
		_, _ = writer.writer.Write(value[:keep])
		writer.remaining -= keep
	}
	if keep < original {
		writer.overflow = true
	}
	return original, nil
}

func fullGitOID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func validRepositoryPath(value string) bool {
	if value == "" || value == "." || strings.HasPrefix(value, "/") || strings.ContainsRune(value, '\\') || path.Clean(value) != value {
		return false
	}
	for _, component := range strings.Split(value, "/") {
		if component == "" || component == "." || component == ".." {
			return false
		}
	}
	for _, character := range value {
		if character < ' ' || character == 0x7f {
			return false
		}
	}
	return true
}
