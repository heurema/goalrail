package githubadmission

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/admission"
	"github.com/heurema/goalrail/internal/domain"
)

type readerFunc func(context.Context, Repository, int64) (Snapshot, error)

func (function readerFunc) Read(ctx context.Context, repository Repository, number int64) (Snapshot, error) {
	return function(ctx, repository, number)
}

func TestCollectorCoversReviewCommentCheckAndClosureFixtures(t *testing.T) {
	seed, event, snapshot := collectorFixture(t)
	first, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: readerFunc(func(context.Context, Repository, int64) (Snapshot, error) {
		return snapshot, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	packet, err := domain.DecodeAdmissionPacket(bytes.NewReader(first.Artifact.CanonicalJSON()))
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Evidence) != 4 { // PR, explicit empty review/comment index, filtered checks, owner decision.
		t.Fatalf("open packet evidence = %d, want 4: %s", len(packet.Evidence), first.Artifact.CanonicalJSON())
	}
	if ContainsProhibitedPacketContent(first.Artifact.CanonicalJSON(), "old approval body", "review body", "token-value") {
		t.Fatal("packet retained prohibited provider content")
	}

	withOwnCheck := snapshot
	withOwnCheck.Checks = append(withOwnCheck.Checks, Check{ID: 99, Name: RequiredCheck, Status: "completed", Conclusion: "success", AppSlug: "github-actions"})
	second, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: readerFunc(func(context.Context, Repository, int64) (Snapshot, error) {
		return withOwnCheck, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Artifact.CanonicalJSON(), second.Artifact.CanonicalJSON()) {
		t.Fatal("the Goalrail check recursively changed its own packet")
	}

	merged := snapshot
	merged.PullRequest.State = "closed"
	mergedAt := snapshot.ObservedAt.Add(-time.Minute)
	merged.PullRequest.MergedAt = &mergedAt
	merged.PullRequest.MergeCommitSHA = strings.Repeat("c", 40)
	closed, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: readerFunc(func(context.Context, Repository, int64) (Snapshot, error) {
		return merged, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	closedPacket, err := domain.DecodeAdmissionPacket(bytes.NewReader(closed.Artifact.CanonicalJSON()))
	if err != nil {
		t.Fatal(err)
	}
	if len(closedPacket.Evidence) != len(packet.Evidence)+1 {
		t.Fatalf("merged closure evidence = %d, want %d", len(closedPacket.Evidence), len(packet.Evidence)+1)
	}
	merged.PullRequest.MergedAt = nil
	merged.PullRequest.MergeCommitSHA = ""
	if _, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: readerFunc(func(context.Context, Repository, int64) (Snapshot, error) {
		return merged, nil
	})}); err != nil {
		t.Fatalf("rejection closure was not collectable: %v", err)
	}
}

func TestCollectorRejectsChangedRangeAuthorityAndUntrustedTime(t *testing.T) {
	seed, event, snapshot := collectorFixture(t)
	cases := []struct {
		name string
		edit func(*Snapshot)
		want error
	}{
		{name: "changed-head", edit: func(value *Snapshot) { value.PullRequest.HeadSHA = strings.Repeat("c", 40) }, want: ErrRangeMismatch},
		{name: "authority-unavailable", edit: func(value *Snapshot) { value.AuthorityAvailable = false }, want: ErrAuthorityUnavailable},
		{name: "unauthenticated", edit: func(value *Snapshot) { value.Authenticated = false }, want: ErrUntrustedObservation},
		{name: "untrusted-time", edit: func(value *Snapshot) { value.ObservedAt = time.Time{} }, want: ErrUntrustedObservation},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			changed := snapshot
			test.edit(&changed)
			_, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: readerFunc(func(context.Context, Repository, int64) (Snapshot, error) {
				return changed, nil
			})})
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestCTX4CandidatesBindTheExactRangeAndAuthenticatedAuthority(t *testing.T) {
	seed, event, snapshot := collectorFixture(t)
	collection, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: readerFunc(func(context.Context, Repository, int64) (Snapshot, error) {
		return snapshot, nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	found := make(map[domain.LineageRelation]admission.ProviderCandidate, len(collection.Candidates))
	for _, candidate := range collection.Candidates {
		if candidate.BaseRevision != seed.BaseRevision || candidate.HeadRevision != seed.HeadRevision {
			t.Fatalf("candidate left the frozen range: %+v", candidate)
		}
		if !candidate.Authenticated || candidate.ObservedAt.IsZero() || candidate.AdapterID != AdapterID {
			t.Fatalf("candidate lost its authenticated provenance: %+v", candidate)
		}
		found[candidate.Relation] = candidate
	}
	for _, relation := range []domain.LineageRelation{
		domain.LineagePullRequest, domain.LineageReviewIndex, domain.LineageCheckSet, domain.LineageOwnerDecision,
	} {
		if _, ok := found[relation]; !ok {
			t.Fatalf("adapter omitted the %s candidate: %+v", relation, collection.Candidates)
		}
	}
	decision := found[domain.LineageOwnerDecision]
	if decision.Outcome != domain.OwnerDecisionAllow || decision.ActorRef != "github-user:owner" ||
		decision.AuthorityRef != "github-permission:admin" {
		t.Fatalf("owner-decision candidate = %+v", decision)
	}
	// A packet is audit evidence. Nothing in its bytes may reconstruct a
	// candidate, so the audit trail cannot become an authority channel.
	if bytes.Contains(collection.Artifact.CanonicalJSON(), []byte("candidates")) {
		t.Fatal("packet serialized the ephemeral candidate set")
	}
}

func TestCTX4LatestCurrentHeadReviewPerActorDecidesTheOwnerBoundary(t *testing.T) {
	seed, event, base := collectorFixture(t)
	for _, fixture := range []struct {
		name        string
		reviews     []Review
		actors      map[string]string
		wantOutcome domain.OwnerDecisionOutcome
		wantPresent bool
	}{
		{name: "approval-at-head", reviews: base.Reviews, actors: base.AuthorizedActors,
			wantOutcome: domain.OwnerDecisionAllow, wantPresent: true},
		{name: "later-dismissal-shadows-approval", reviews: append(append([]Review(nil), base.Reviews...), Review{
			ID: 3, Actor: "owner", State: "DISMISSED", CommitSHA: seed.HeadRevision,
			SubmittedAt: base.ObservedAt.Add(-10 * time.Minute),
		}), actors: base.AuthorizedActors},
		{name: "later-changes-requested-shadows-approval", reviews: append(append([]Review(nil), base.Reviews...), Review{
			ID: 4, Actor: "owner", State: "CHANGES_REQUESTED", CommitSHA: seed.HeadRevision,
			SubmittedAt: base.ObservedAt.Add(-5 * time.Minute),
		}), actors: base.AuthorizedActors, wantOutcome: domain.OwnerDecisionReject, wantPresent: true},
		{name: "later-neutral-comment-shadows-approval", reviews: append(append([]Review(nil), base.Reviews...), Review{
			ID: 5, Actor: "owner", State: "COMMENTED", CommitSHA: seed.HeadRevision,
			SubmittedAt: base.ObservedAt.Add(-time.Minute),
		}), actors: base.AuthorizedActors},
		{name: "approval-only-on-a-stale-head", reviews: []Review{{
			ID: 6, Actor: "owner", State: "APPROVED", CommitSHA: strings.Repeat("c", 40),
			SubmittedAt: base.ObservedAt.Add(-time.Hour),
		}}, actors: base.AuthorizedActors},
		{name: "actor-without-authority", reviews: base.Reviews, actors: map[string]string{}},
		{name: "rejection-dominates-a-second-approval", reviews: append(append([]Review(nil), base.Reviews...), Review{
			ID: 7, Actor: "second", State: "CHANGES_REQUESTED", CommitSHA: seed.HeadRevision,
			SubmittedAt: base.ObservedAt.Add(-2 * time.Minute),
		}), actors: map[string]string{"owner": "github-permission:admin", "second": "github-permission:write"},
			wantOutcome: domain.OwnerDecisionReject, wantPresent: true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			snapshot := base
			snapshot.Reviews = fixture.reviews
			snapshot.AuthorizedActors = fixture.actors
			collection, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: readerFunc(func(context.Context, Repository, int64) (Snapshot, error) {
				return snapshot, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			var decision *admission.ProviderCandidate
			for index, candidate := range collection.Candidates {
				if candidate.Relation == domain.LineageOwnerDecision {
					decision = &collection.Candidates[index]
				}
			}
			if !fixture.wantPresent {
				if decision != nil {
					t.Fatalf("owner-decision candidate survived: %+v", decision)
				}
				return
			}
			if decision == nil || decision.Outcome != fixture.wantOutcome {
				t.Fatalf("owner-decision candidate = %+v, want outcome %s", decision, fixture.wantOutcome)
			}
		})
	}
}

func TestRESTReaderStripsBodiesAndNeverRetainsCredential(t *testing.T) {
	const credential = "github_pat_fixture_abcdefghijklmnopqrstuvwxyz"
	observed := time.Date(2026, 8, 5, 13, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Errorf("method = %s", request.Method)
		}
		if request.Header.Get("Authorization") != "Bearer "+credential {
			t.Error("read-only credential was not conveyed in the request header")
		}
		writer.Header().Set("Date", observed.Format(http.TimeFormat))
		path := request.URL.Path
		switch {
		case strings.HasSuffix(path, "/pulls/42"):
			fmt.Fprint(writer, `{"number":42,"state":"open","user":{"login":"alice"},"base":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"head":{"sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"updated_at":"2026-08-05T12:00:00Z","merged_at":null,"merge_commit_sha":"","body":"raw pull body"}`)
		case strings.HasSuffix(path, "/pulls/42/reviews"):
			fmt.Fprint(writer, `[{"id":7,"state":"APPROVED","commit_id":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","user":{"login":"owner"},"submitted_at":"2026-08-05T12:30:00Z","body":"raw review body"}]`)
		case strings.HasSuffix(path, "/issues/42/comments"):
			fmt.Fprint(writer, `[{"id":8,"body":"raw issue comment"}]`)
		case strings.HasSuffix(path, "/pulls/42/comments"):
			fmt.Fprint(writer, `[]`)
		case strings.Contains(path, "/check-runs"):
			fmt.Fprint(writer, `{"total_count":1,"check_runs":[{"id":9,"name":"unit-test","status":"completed","conclusion":"success","app":{"slug":"ci"},"completed_at":"2026-08-05T12:40:00Z","output":{"text":"raw output body"}}]}`)
		case strings.Contains(path, "/collaborators/owner/permission"):
			fmt.Fprint(writer, `{"permission":"admin","user":{"login":"owner"}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	reader, err := NewRESTReader(server.URL, credential, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	seed, event, _ := collectorFixture(t)
	artifact, err := Collect(context.Background(), CollectRequest{Seed: seed, Event: event, Reader: reader})
	if err != nil {
		t.Fatal(err)
	}
	if ContainsProhibitedPacketContent(artifact.Artifact.CanonicalJSON(), credential, "raw pull body", "raw review body", "raw issue comment", "raw output body") {
		t.Fatalf("packet retained a credential or body: %s", artifact.Artifact.CanonicalJSON())
	}
}

func TestRESTReaderRejectsPlaintextNonLoopbackEndpoint(t *testing.T) {
	if _, err := NewRESTReader("http://github.example/api/v3", "fixture-token", http.DefaultClient); err == nil {
		t.Fatal("GitHub reader accepted a plaintext non-loopback endpoint")
	}
}

func TestWorkflowEventShapeIsBoundedAndAllowlisted(t *testing.T) {
	raw := `{"action":"synchronize","number":42,"repository":{"full_name":"acme/widget"},"pull_request":{"number":42,"base":{"sha":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"head":{"sha":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"body":"ignored raw body"}}`
	event, err := DecodeWorkflowEvent("pull_request", strings.NewReader(raw))
	if err != nil || event.Repository.String() != "acme/widget" {
		t.Fatalf("event = %+v, %v", event, err)
	}
	if _, err := DecodeWorkflowEvent("check_run", strings.NewReader(raw)); !errors.Is(err, ErrUnsupportedEvent) {
		t.Fatalf("unsupported event error = %v", err)
	}
}

func collectorFixture(t *testing.T) (domain.AdmissionPacket, WorkflowEvent, Snapshot) {
	t.Helper()
	evaluation := time.Date(2026, 8, 5, 13, 0, 0, 0, time.UTC)
	seed := domain.AdmissionPacket{
		Schema: domain.AdmissionPacketSchemaV1, ProjectID: "prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DeclarationDigest: testDigest('1'), PolicyDigest: testDigest('2'),
		BaseRevision: strings.Repeat("a", 40), HeadRevision: strings.Repeat("b", 40),
		WorkUnitRef: domain.ContentAddressedEvidenceReference{
			ArtifactKind: "work_unit", Identity: "work-unit:wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Version: "1",
			Digest: testDigest('3'), SourceRef: "repo:root/.goalrail/work-units/wu_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/unit.json", AdapterID: "goalrail",
		},
		Evidence: []domain.ContentAddressedEvidenceReference{}, Provenance: []domain.AdmissionProviderProvenance{},
	}
	event := WorkflowEvent{Name: "pull_request", Action: "synchronize", Repository: Repository{Owner: "acme", Name: "widget"}, Number: 42, BaseSHA: seed.BaseRevision, HeadSHA: seed.HeadRevision}
	snapshot := Snapshot{
		PullRequest: PullRequest{Number: 42, BaseSHA: seed.BaseRevision, HeadSHA: seed.HeadRevision, State: "open", Author: "alice", UpdatedAt: evaluation.Add(-time.Hour)},
		Reviews: []Review{
			{ID: 1, Actor: "owner", State: "APPROVED", CommitSHA: strings.Repeat("c", 40), SubmittedAt: evaluation.Add(-40 * time.Minute)},
			{ID: 2, Actor: "owner", State: "APPROVED", CommitSHA: seed.HeadRevision, SubmittedAt: evaluation.Add(-20 * time.Minute)},
		},
		IssueCommentIDs: []int64{}, ReviewCommentIDs: []int64{},
		Checks:           []Check{{ID: 3, Name: "unit-test", Status: "completed", Conclusion: "success", AppSlug: "ci"}},
		AuthorizedActors: map[string]string{"owner": "github-permission:admin"}, AuthorityAvailable: true,
		ObservedAt: evaluation, Authenticated: true,
	}
	return seed, event, snapshot
}

func testDigest(character byte) domain.SHA256Digest {
	return domain.SHA256Digest("sha256:" + strings.Repeat(string(character), 64))
}
