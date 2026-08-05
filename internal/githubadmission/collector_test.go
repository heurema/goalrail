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
	packet, err := domain.DecodeAdmissionPacket(bytes.NewReader(first.CanonicalJSON()))
	if err != nil {
		t.Fatal(err)
	}
	if len(packet.Evidence) != 4 { // PR, explicit empty review/comment index, filtered checks, owner decision.
		t.Fatalf("open packet evidence = %d, want 4: %s", len(packet.Evidence), first.CanonicalJSON())
	}
	if ContainsProhibitedPacketContent(first.CanonicalJSON(), "old approval body", "review body", "token-value") {
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
	if !bytes.Equal(first.CanonicalJSON(), second.CanonicalJSON()) {
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
	closedPacket, err := domain.DecodeAdmissionPacket(bytes.NewReader(closed.CanonicalJSON()))
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
	if ContainsProhibitedPacketContent(artifact.CanonicalJSON(), credential, "raw pull body", "raw review body", "raw issue comment", "raw output body") {
		t.Fatalf("packet retained a credential or body: %s", artifact.CanonicalJSON())
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
