package lineage

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/localrun"
)

func TestAttachIsIdempotentAndRetainsSingleRelationConflict(t *testing.T) {
	store, receipt, observedAt := begunLineageFixture(t, "wu_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", nil)
	first := testLineageEvent(receipt, domain.LineagePullRequest, domain.RelationSingle,
		testLineageReference("pull_request", "github-pr:owner/repo/1", "provider:github/pr/1", []byte("pr-1")), observedAt)
	attached, err := store.Attach(first, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !attached.Created {
		t.Fatal("first event attachment was not reported as created")
	}
	repeated, err := store.Attach(first, nil)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Created || repeated.EventRef != attached.EventRef {
		t.Fatalf("identical attachment = %+v", repeated)
	}
	conflicting := testLineageEvent(receipt, domain.LineagePullRequest, domain.RelationSingle,
		testLineageReference("pull_request", "github-pr:owner/repo/2", "provider:github/pr/2", []byte("pr-2")), observedAt.Add(time.Minute))
	if _, err := store.Attach(conflicting, nil); err != nil {
		t.Fatal(err)
	}
	graph, err := store.Graph(receipt.WorkUnitID)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Conflicts) != 1 || graph.Conflicts[0].Relation != domain.LineagePullRequest || len(graph.Conflicts[0].Digests) != 2 {
		t.Fatalf("single-valued conflict was not retained: %+v", graph.Conflicts)
	}
}

func TestReplicaRequiresExactCanonicalSafeBytes(t *testing.T) {
	canonical := []byte(`{"schema":"goalrail.test-evidence/v1","state":"verified"}`)
	digest := domain.DigestCanonicalJSON(canonical)
	replica, err := PrepareReplica(bytes.NewReader(canonical), digest, "goalrail.test-evidence/v1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(replica.Canonical, canonical) || replica.Digest != digest {
		t.Fatalf("replica = %+v", replica)
	}
	tests := []struct {
		name   string
		raw    []byte
		digest domain.SHA256Digest
		schema string
	}{
		{name: "digest mismatch", raw: canonical, digest: domain.DigestCanonicalJSON([]byte("other")), schema: "goalrail.test-evidence/v1"},
		{name: "schema mismatch", raw: canonical, digest: digest, schema: "goalrail.other/v1"},
		{name: "noncanonical", raw: []byte("{ \"schema\":\"goalrail.test-evidence/v1\",\"state\":\"verified\"}"), schema: "goalrail.test-evidence/v1"},
		{name: "raw payload", raw: []byte(`{"schema":"goalrail.test-evidence/v1","prompt":"copied body"}`), schema: "goalrail.test-evidence/v1"},
		{name: "invalid known schema", raw: []byte(`{"schema":"goalrail.work-spec/v1"}`), schema: domain.WorkSpecSchemaV1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.digest == "" {
				test.digest = domain.DigestCanonicalJSON(test.raw)
			}
			if _, err := PrepareReplica(bytes.NewReader(test.raw), test.digest, test.schema); err == nil {
				t.Fatal("unsafe replica was accepted")
			}
		})
	}
}

func TestEventOrderLateReferencesReplicaAndReverseResolution(t *testing.T) {
	repository := managedRepositoryFixture(t)
	copyCurrentChange(t, repository)
	observedAt := time.Date(2026, 8, 5, 14, 0, 0, 0, time.UTC)
	firstReceipt := beginWithID(t, repository, "wu_cccccccccccccccccccccccccccccccc", observedAt, nil)
	secondReceipt := beginWithID(t, repository, "wu_dddddddddddddddddddddddddddddddd", observedAt, nil)
	store, err := NewStore(repository)
	if err != nil {
		t.Fatal(err)
	}
	receiptRaw, err := localrun.CanonicalTerminalReceipt(localrun.TerminalReceipt{
		Schema:     localrun.TerminalReceiptSchemaV1,
		WorkSpecID: "work-spec-test", WorkSpecVersion: 1,
		WorkSpecDigest: domain.WorkSpecDigest("sha256:" + strings.Repeat("a", 64)),
		Intent: &localrun.ReceiptIntentReference{
			ID: "intent-test", Version: 1, Digest: "sha256:" + strings.Repeat("b", 64),
		},
		RunID: "run-test", Adapter: "test", AdapterVersion: "1",
		BaseRevision: "base", TerminalHead: "head", Status: localrun.StatePassed,
		PreparedAt: observedAt, LaunchAttemptedAt: observedAt, ProviderObservedAt: observedAt, TerminalAt: observedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	receiptDigest := domain.DigestCanonicalJSON(receiptRaw)
	replica, err := PrepareReplica(bytes.NewReader(receiptRaw), receiptDigest, localrun.TerminalReceiptSchemaV1)
	if err != nil {
		t.Fatal(err)
	}
	replicaRef := testLineageReference("terminal_receipt", "receipt:shared-terminal", repositorySourceRef(replica.Reference), receiptRaw)
	if replicaRef.Digest != replica.Digest {
		t.Fatal("replica fixture digest mismatch")
	}
	latePR := testLineageEvent(firstReceipt, domain.LineagePullRequest, domain.RelationSingle,
		testLineageReference("pull_request", "github-pr:owner/repo/9", "provider:github/pr/9", []byte("pr-9")), observedAt.Add(2*time.Minute))
	run := testLineageEvent(firstReceipt, domain.LineageRunSession, domain.RelationSingle,
		testLineageReference("run_session", "run:run-9", "local-run:run-9", []byte("run-9")), observedAt.Add(time.Minute))
	terminal := testLineageEvent(firstReceipt, domain.LineageTerminalReceipt, domain.RelationSingle, replicaRef, observedAt.Add(3*time.Minute))
	for _, event := range []domain.LineageEvent{latePR, run} {
		if _, err := store.Attach(event, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Attach(terminal, &replica); err != nil {
		t.Fatal(err)
	}
	secondTerminal := testLineageEvent(secondReceipt, domain.LineageTerminalReceipt, domain.RelationSingle, replicaRef, observedAt.Add(4*time.Minute))
	if _, err := store.Attach(secondTerminal, nil); err != nil {
		t.Fatal(err)
	}
	graph, err := store.Graph(firstReceipt.WorkUnitID)
	if err != nil {
		t.Fatal(err)
	}
	for index := 1; index < len(graph.Events); index++ {
		if graph.Events[index-1].SemanticDigest > graph.Events[index].SemanticDigest {
			t.Fatal("events are not resolved in digest order")
		}
	}
	if !bytes.Equal(graph.Replicas[receiptDigest], receiptRaw) {
		t.Fatal("exact content-addressed replica was not resolved")
	}
	resolved, err := store.Resolve("run:run-9")
	if err != nil || resolved.Status != ResolutionFound || !reflect.DeepEqual(resolved.WorkUnitIDs, []domain.WorkUnitID{firstReceipt.WorkUnitID}) {
		t.Fatalf("run reverse resolution = %+v, %v", resolved, err)
	}
	ambiguous, err := store.Resolve("receipt:shared-terminal")
	if err != nil || ambiguous.Status != ResolutionAmbiguous || len(ambiguous.WorkUnitIDs) != 2 {
		t.Fatalf("ambiguous reverse resolution = %+v, %v", ambiguous, err)
	}
}

func TestCompletenessReportsEmptyUnavailableAndExpiredException(t *testing.T) {
	requirements := []domain.LineageRequirement{
		{Relation: domain.LineageWorkSpec, Cardinality: domain.RelationSingle},
		{Relation: domain.LineageReviewIndex, Cardinality: domain.RelationSet},
		{Relation: domain.LineageException, Cardinality: domain.RelationSet},
		{Relation: domain.LineageClosure, Cardinality: domain.RelationSingle},
	}
	store, receipt, observedAt := begunLineageFixture(t, "wu_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", requirements)
	workSpec := testLineageEvent(receipt, domain.LineageWorkSpec, domain.RelationSingle,
		testLineageReference("work_spec", "work-spec:test/v1", "local-run:work-spec", []byte("work-spec")), observedAt)
	emptyReviews := testLineageEvent(receipt, domain.LineageReviewIndex, domain.RelationSet,
		testLineageReference("empty_set", "review-index:none", "provider:github/reviews", []byte("empty")), observedAt)
	exceptionRaw := []byte(`{"expires_at":"2026-08-05T15:00:00Z","schema":"goalrail.lineage-exception/v1"}`)
	exceptionReplica, err := PrepareReplica(bytes.NewReader(exceptionRaw), domain.DigestCanonicalJSON(exceptionRaw), LineageExceptionSchemaV1)
	if err != nil {
		t.Fatal(err)
	}
	exceptionRef := testLineageReference("exception", "exception:bounded-1", repositorySourceRef(exceptionReplica.Reference), exceptionRaw)
	exception := testLineageEvent(receipt, domain.LineageException, domain.RelationSet, exceptionRef, observedAt)
	unavailableClosure := testLineageEvent(receipt, domain.LineageClosure, domain.RelationSingle,
		testLineageReference("provider_unavailable", "closure:unavailable", "provider:github/closure", []byte("unavailable")), observedAt)
	for _, input := range []struct {
		event   domain.LineageEvent
		replica *Replica
	}{{workSpec, nil}, {emptyReviews, nil}, {exception, &exceptionReplica}, {unavailableClosure, nil}} {
		if _, err := store.Attach(input.event, input.replica); err != nil {
			t.Fatal(err)
		}
	}
	graph, err := store.Graph(receipt.WorkUnitID)
	if err != nil {
		t.Fatal(err)
	}
	complete, err := EvaluateCompleteness(graph, observedAt.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if complete.Lifecycle != domain.WorkUnitOpen ||
		!reflect.DeepEqual(complete.ExplicitEmpty, []domain.LineageRelation{domain.LineageReviewIndex}) ||
		!reflect.DeepEqual(complete.Unavailable, []domain.LineageRelation{domain.LineageClosure}) ||
		len(complete.ExpiredExceptions) != 1 {
		t.Fatalf("completeness = %+v", complete)
	}
}

func begunLineageFixture(t *testing.T, id domain.WorkUnitID, requirements []domain.LineageRequirement) (*Store, BeginReceipt, time.Time) {
	t.Helper()
	repository := managedRepositoryFixture(t)
	copyCurrentChange(t, repository)
	observedAt := time.Date(2026, 8, 5, 13, 0, 0, 0, time.UTC)
	receipt := beginWithID(t, repository, id, observedAt, requirements)
	store, err := NewStore(repository)
	if err != nil {
		t.Fatal(err)
	}
	return store, receipt, observedAt
}

func copyCurrentChange(t *testing.T, repository string) {
	t.Helper()
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(
		workingDirectory,
		"..", "..", "openspec", "changes", "archive", "2026-08-05-project-lineage-admission-v0",
	)
	if _, err := os.Stat(source); err != nil {
		// Tests normally run with package cwd internal/lineage.
		source = filepath.Join(
			workingDirectory,
			"openspec", "changes", "archive", "2026-08-05-project-lineage-admission-v0",
		)
	}
	copyTree(t, source, filepath.Join(repository, "openspec", "changes", "project-lineage-admission-v0"))
}

func beginWithID(t *testing.T, repository string, id domain.WorkUnitID, observedAt time.Time, requirements []domain.LineageRequirement) BeginReceipt {
	t.Helper()
	receipt, err := Begin(context.Background(), BeginOptions{
		Repository: repository, ChangeID: "project-lineage-admission-v0", ActorRef: "user:test-owner",
		RequiredRelations: requirements, Now: func() time.Time { return observedAt },
		NewWorkUnitID: func() (domain.WorkUnitID, error) { return id, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func testLineageEvent(receipt BeginReceipt, relation domain.LineageRelation, cardinality domain.RelationCardinality, target domain.ContentAddressedEvidenceReference, observedAt time.Time) domain.LineageEvent {
	source := testLineageReference("work_unit", "work-unit:"+string(receipt.WorkUnitID), repositorySourceRef(receipt.AnchorRef), []byte(receipt.AnchorDigest))
	source.Digest = receipt.AnchorDigest
	return domain.LineageEvent{
		Schema: domain.LineageEventSchemaV1, WorkUnitID: receipt.WorkUnitID,
		Relation: relation, Cardinality: cardinality,
		Sources: []domain.ContentAddressedEvidenceReference{source}, Targets: []domain.ContentAddressedEvidenceReference{target},
		ActorRef: "user:test-owner", AdapterID: lineageAdapterID, ObservedAt: observedAt,
	}
}

func testLineageReference(kind, identity, source string, content []byte) domain.ContentAddressedEvidenceReference {
	return domain.ContentAddressedEvidenceReference{
		ArtifactKind: kind, Identity: identity, Version: "1", Digest: domain.DigestCanonicalJSON(content),
		SourceRef: source, AdapterID: lineageAdapterID,
	}
}
