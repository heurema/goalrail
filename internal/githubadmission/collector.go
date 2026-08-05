// Package githubadmission is the isolated GitHub adapter for Goalrail
// admission. It converts authenticated provider observations into the existing
// provider-neutral packet contract; it owns no materiality or verdict logic.
package githubadmission

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/admission"
	"github.com/heurema/goalrail/internal/domain"
)

const (
	AdapterID       = "github-actions"
	Provider        = "github"
	RequiredCheck   = "goalrail/lineage"
	maxEventBytes   = 2 << 20
	maxProviderRefs = 256
)

var (
	ErrUnsupportedEvent     = errors.New("unsupported GitHub workflow event")
	ErrRangeMismatch        = errors.New("GitHub pull-request range differs from the frozen admission range")
	ErrAuthorityUnavailable = errors.New("GitHub actor authority is unavailable")
	ErrUntrustedObservation = errors.New("GitHub observation is not authenticated or has no trusted timestamp")
)

type Repository struct {
	Owner string
	Name  string
}

func ParseRepository(value string) (Repository, error) {
	owner, name, ok := strings.Cut(strings.TrimSpace(value), "/")
	if !ok || strings.Contains(name, "/") || !domain.IsCanonicalID(owner) || !domain.IsCanonicalID(name) {
		return Repository{}, errors.New("GitHub repository must be one canonical owner/name pair")
	}
	return Repository{Owner: owner, Name: name}, nil
}

func (repository Repository) String() string { return repository.Owner + "/" + repository.Name }

type WorkflowEvent struct {
	Name       string
	Action     string
	Repository Repository
	Number     int64
	BaseSHA    string
	HeadSHA    string
}

type eventEnvelope struct {
	Action     string `json:"action"`
	Number     int64  `json:"number"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	PullRequest struct {
		Number int64 `json:"number"`
		Base   struct {
			SHA string `json:"sha"`
		} `json:"base"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

func DecodeWorkflowEvent(name string, reader io.Reader) (WorkflowEvent, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxEventBytes+1))
	if err != nil {
		return WorkflowEvent{}, fmt.Errorf("read GitHub workflow event: %w", err)
	}
	if len(raw) > maxEventBytes {
		return WorkflowEvent{}, fmt.Errorf("GitHub workflow event exceeds %d bytes", maxEventBytes)
	}
	var envelope eventEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return WorkflowEvent{}, fmt.Errorf("decode GitHub workflow event: %w", err)
	}
	if !supportedEvent(name, envelope.Action) {
		return WorkflowEvent{}, ErrUnsupportedEvent
	}
	repository, err := ParseRepository(envelope.Repository.FullName)
	if err != nil {
		return WorkflowEvent{}, err
	}
	number := envelope.Number
	if number == 0 {
		number = envelope.PullRequest.Number
	}
	if number <= 0 || envelope.PullRequest.Base.SHA == "" || envelope.PullRequest.Head.SHA == "" {
		return WorkflowEvent{}, errors.New("GitHub workflow event omits pull-request range identity")
	}
	return WorkflowEvent{
		Name: name, Action: envelope.Action, Repository: repository, Number: number,
		BaseSHA: envelope.PullRequest.Base.SHA, HeadSHA: envelope.PullRequest.Head.SHA,
	}, nil
}

func supportedEvent(name, action string) bool {
	allowed := map[string]map[string]bool{
		"pull_request": {
			"opened": true, "reopened": true, "synchronize": true,
			"ready_for_review": true, "closed": true,
		},
		"pull_request_review": {"submitted": true, "edited": true, "dismissed": true},
	}
	return allowed[name][action]
}

type PullRequest struct {
	Number         int64
	BaseSHA        string
	HeadSHA        string
	State          string
	Author         string
	UpdatedAt      time.Time
	MergedAt       *time.Time
	MergeCommitSHA string
}

type Review struct {
	ID          int64
	Actor       string
	State       string
	CommitSHA   string
	SubmittedAt time.Time
}

type Check struct {
	ID          int64
	Name        string
	Status      string
	Conclusion  string
	AppSlug     string
	CompletedAt *time.Time
}

type Snapshot struct {
	PullRequest        PullRequest
	Reviews            []Review
	IssueCommentIDs    []int64
	ReviewCommentIDs   []int64
	Checks             []Check
	AuthorizedActors   map[string]string
	AuthorityAvailable bool
	ObservedAt         time.Time
	Authenticated      bool
}

type Reader interface {
	Read(context.Context, Repository, int64) (Snapshot, error)
}

type CollectRequest struct {
	Seed   domain.AdmissionPacket
	Event  WorkflowEvent
	Reader Reader
}

// Collection carries both products of one authenticated observation: the
// canonical audit packet, and the in-memory candidates that alone can complete
// a provider-owned relation. The candidates are deliberately not recoverable
// from the packet.
type Collection struct {
	Artifact   domain.CanonicalArtifact
	Candidates []admission.ProviderCandidate
}

func Collect(ctx context.Context, request CollectRequest) (Collection, error) {
	if request.Reader == nil {
		return Collection{}, errors.New("GitHub admission reader is required")
	}
	if request.Seed.BaseRevision != request.Event.BaseSHA || request.Seed.HeadRevision != request.Event.HeadSHA {
		return Collection{}, ErrRangeMismatch
	}
	snapshot, err := request.Reader.Read(ctx, request.Event.Repository, request.Event.Number)
	if err != nil {
		return Collection{}, err
	}
	if snapshot.PullRequest.Number != request.Event.Number || snapshot.PullRequest.BaseSHA != request.Event.BaseSHA || snapshot.PullRequest.HeadSHA != request.Event.HeadSHA {
		return Collection{}, ErrRangeMismatch
	}
	if !snapshot.AuthorityAvailable {
		return Collection{}, ErrAuthorityUnavailable
	}
	if !snapshot.Authenticated || snapshot.ObservedAt.IsZero() {
		return Collection{}, ErrUntrustedObservation
	}
	if snapshot.ObservedAt.Before(snapshot.PullRequest.UpdatedAt) {
		return Collection{}, errors.New("GitHub observation time predates the pull-request state")
	}

	references, decisions, err := providerReferences(request.Event.Repository, snapshot)
	if err != nil {
		return Collection{}, err
	}
	packet := request.Seed
	evaluationTime := snapshot.ObservedAt.UTC()
	packet.EvaluationTime = &evaluationTime
	packet.TimeAuthorityRef = fmt.Sprintf("github:repos/%s/pulls/%d/observation", request.Event.Repository.String(), request.Event.Number)
	packet.Evidence = append(append([]domain.ContentAddressedEvidenceReference(nil), packet.Evidence...), references...)
	for _, reference := range references {
		packet.Provenance = append(packet.Provenance, domain.AdmissionProviderProvenance{
			AdapterID: AdapterID, ProviderRef: reference.SourceRef, EvidenceDigest: reference.Digest,
			ObservedAt: evaluationTime, Authenticated: true,
		})
	}
	if len(references) > 0 {
		// The declared time authority needs its own authenticated provenance
		// entry; without it the packet names an evaluation time that nothing in
		// the packet vouches for.
		packet.Provenance = append(packet.Provenance, domain.AdmissionProviderProvenance{
			AdapterID: AdapterID, ProviderRef: packet.TimeAuthorityRef, EvidenceDigest: references[0].Digest,
			ObservedAt: evaluationTime, Authenticated: true,
		})
	}
	artifact, err := domain.FreezeAdmissionPacket(packet)
	if err != nil {
		return Collection{}, fmt.Errorf("freeze GitHub admission packet: %w", err)
	}
	if domain.ContainsSecretShapedContent(string(artifact.CanonicalJSON())) {
		return Collection{}, errors.New("GitHub packet contains secret-shaped content")
	}
	return Collection{
		Artifact:   artifact,
		Candidates: providerCandidates(request.Event, snapshot, references, decisions, evaluationTime),
	}, nil
}

// providerCandidates converts the same authenticated observation into bounded
// in-memory candidates. Every candidate names the exact frozen range and the
// observation time, so a candidate from another range or a later state cannot
// be reused.
func providerCandidates(
	event WorkflowEvent,
	snapshot Snapshot,
	references []domain.ContentAddressedEvidenceReference,
	decisions []indexedDecision,
	observedAt time.Time,
) []admission.ProviderCandidate {
	relations := map[string]domain.LineageRelation{
		"pull_request":   domain.LineagePullRequest,
		"review_index":   domain.LineageReviewIndex,
		"check_set":      domain.LineageCheckSet,
		"owner_decision": domain.LineageOwnerDecision,
		"closure":        domain.LineageClosure,
	}
	decision, hasDecision := selectOwnerDecision(decisions)
	candidates := make([]admission.ProviderCandidate, 0, len(references))
	for _, reference := range references {
		relation, known := relations[reference.ArtifactKind]
		if !known {
			continue
		}
		candidate := admission.ProviderCandidate{
			Relation: relation, Identity: reference.Identity, Digest: reference.Digest,
			AdapterID: AdapterID, ProviderRef: reference.SourceRef,
			BaseRevision: event.BaseSHA, HeadRevision: event.HeadSHA,
			ObservedAt: observedAt, Authenticated: snapshot.Authenticated,
		}
		if relation == domain.LineageOwnerDecision {
			if !hasDecision {
				continue
			}
			candidate.ActorRef = decision.ActorRef
			candidate.AuthorityRef = decision.Authority
			candidate.Outcome = domain.OwnerDecisionOutcome(decision.Outcome)
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

// selectOwnerDecision collapses the authorized current-head decisions into one
// candidate for the single-valued owner relation. A rejection by any authorized
// owner dominates every approval; without that rule, one extra approval could
// erase a recorded objection.
func selectOwnerDecision(decisions []indexedDecision) (indexedDecision, bool) {
	sorted := append([]indexedDecision(nil), decisions...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ActorRef < sorted[j].ActorRef })
	for _, decision := range sorted {
		if decision.Outcome == "reject" {
			return decision, true
		}
	}
	for _, decision := range sorted {
		if decision.Outcome == "allow" {
			return decision, true
		}
	}
	return indexedDecision{}, false
}

type pullRequestEvidence struct {
	Schema         string     `json:"schema"`
	Repository     string     `json:"repository"`
	Number         int64      `json:"number"`
	BaseSHA        string     `json:"base_sha"`
	HeadSHA        string     `json:"head_sha"`
	State          string     `json:"state"`
	AuthorRef      string     `json:"author_ref"`
	UpdatedAt      time.Time  `json:"updated_at"`
	MergedAt       *time.Time `json:"merged_at"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
}

type indexedReview struct {
	ID          int64     `json:"id"`
	ActorRef    string    `json:"actor_ref"`
	State       string    `json:"state"`
	CommitSHA   string    `json:"commit_sha"`
	SubmittedAt time.Time `json:"submitted_at"`
	CurrentHead bool      `json:"current_head"`
}

type reviewIndexEvidence struct {
	Schema         string          `json:"schema"`
	Repository     string          `json:"repository"`
	PullRequest    int64           `json:"pull_request"`
	Reviews        []indexedReview `json:"reviews"`
	IssueComments  []int64         `json:"issue_comments"`
	ReviewComments []int64         `json:"review_comments"`
}

type indexedCheck struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Conclusion  string     `json:"conclusion"`
	AppRef      string     `json:"app_ref"`
	CompletedAt *time.Time `json:"completed_at"`
}

type checkSetEvidence struct {
	Schema     string         `json:"schema"`
	Repository string         `json:"repository"`
	HeadSHA    string         `json:"head_sha"`
	Checks     []indexedCheck `json:"checks"`
}

type indexedDecision struct {
	ActorRef   string    `json:"actor_ref"`
	Authority  string    `json:"authority"`
	Outcome    string    `json:"outcome"`
	ReviewID   int64     `json:"review_id"`
	ObservedAt time.Time `json:"observed_at"`
}

type ownerDecisionEvidence struct {
	Schema      string            `json:"schema"`
	Repository  string            `json:"repository"`
	PullRequest int64             `json:"pull_request"`
	HeadSHA     string            `json:"head_sha"`
	Decisions   []indexedDecision `json:"decisions"`
}

type closureEvidence struct {
	Schema         string     `json:"schema"`
	Repository     string     `json:"repository"`
	PullRequest    int64      `json:"pull_request"`
	Outcome        string     `json:"outcome"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
	ObservedAt     *time.Time `json:"observed_at"`
}

func providerReferences(repository Repository, snapshot Snapshot) ([]domain.ContentAddressedEvidenceReference, []indexedDecision, error) {
	repo := repository.String()
	pr := snapshot.PullRequest
	prEvidence := pullRequestEvidence{
		Schema: "goalrail.github-pull-request/v1", Repository: repo, Number: pr.Number,
		BaseSHA: pr.BaseSHA, HeadSHA: pr.HeadSHA, State: strings.ToLower(pr.State),
		AuthorRef: "github-user:" + pr.Author, UpdatedAt: pr.UpdatedAt.UTC(),
		MergedAt: utcPointer(pr.MergedAt), MergeCommitSHA: pr.MergeCommitSHA,
	}
	if !domain.IsEvidenceReference(prEvidence.AuthorRef) || pr.UpdatedAt.IsZero() {
		return nil, nil, errors.New("GitHub pull request contains an invalid actor or timestamp")
	}
	refs := []domain.ContentAddressedEvidenceReference{
		reference("pull_request", fmt.Sprintf("github:repos/%s/pulls/%d", repo, pr.Number), prEvidence),
	}

	reviewEvidence, decisions, err := buildReviewIndex(repo, snapshot)
	if err != nil {
		return nil, nil, err
	}
	refs = append(refs, reference("review_index", fmt.Sprintf("github:repos/%s/pulls/%d/review-index", repo, pr.Number), reviewEvidence))
	checks, err := buildCheckSet(repo, snapshot)
	if err != nil {
		return nil, nil, err
	}
	refs = append(refs, reference("check_set", fmt.Sprintf("github:repos/%s/commits/%s/checks", repo, pr.HeadSHA), checks))
	if len(decisions) > 0 {
		decisionEvidence := ownerDecisionEvidence{
			Schema: "goalrail.github-owner-decision/v1", Repository: repo,
			PullRequest: pr.Number, HeadSHA: pr.HeadSHA, Decisions: decisions,
		}
		refs = append(refs, reference("owner_decision", fmt.Sprintf("github:repos/%s/pulls/%d/owner-decision", repo, pr.Number), decisionEvidence))
	}
	if strings.EqualFold(pr.State, "closed") {
		outcome := "rejected"
		if pr.MergedAt != nil {
			outcome = "merged"
		}
		closure := closureEvidence{
			Schema: "goalrail.github-closure/v1", Repository: repo, PullRequest: pr.Number,
			Outcome: outcome, MergeCommitSHA: pr.MergeCommitSHA, ObservedAt: utcPointer(pr.MergedAt),
		}
		refs = append(refs, reference("closure", fmt.Sprintf("github:repos/%s/pulls/%d/closure", repo, pr.Number), closure))
	}
	if len(refs) > maxProviderRefs {
		return nil, nil, fmt.Errorf("GitHub evidence exceeds %d references", maxProviderRefs)
	}
	return refs, decisions, nil
}

func buildReviewIndex(repository string, snapshot Snapshot) (reviewIndexEvidence, []indexedDecision, error) {
	reviews := append([]Review(nil), snapshot.Reviews...)
	sort.Slice(reviews, func(i, j int) bool {
		if reviews[i].SubmittedAt.Equal(reviews[j].SubmittedAt) {
			return reviews[i].ID < reviews[j].ID
		}
		return reviews[i].SubmittedAt.Before(reviews[j].SubmittedAt)
	})
	index := reviewIndexEvidence{
		Schema: "goalrail.github-review-index/v1", Repository: repository,
		PullRequest:   snapshot.PullRequest.Number,
		IssueComments: sortedIDs(snapshot.IssueCommentIDs), ReviewComments: sortedIDs(snapshot.ReviewCommentIDs),
	}
	latest := make(map[string]Review)
	for _, review := range reviews {
		actorRef := "github-user:" + review.Actor
		if review.ID <= 0 || !domain.IsEvidenceReference(actorRef) || review.SubmittedAt.IsZero() {
			return reviewIndexEvidence{}, nil, errors.New("GitHub review contains an invalid identity or timestamp")
		}
		state := strings.ToLower(review.State)
		index.Reviews = append(index.Reviews, indexedReview{
			ID: review.ID, ActorRef: actorRef, State: state, CommitSHA: review.CommitSHA,
			SubmittedAt: review.SubmittedAt.UTC(), CurrentHead: review.CommitSHA == snapshot.PullRequest.HeadSHA,
		})
		if review.CommitSHA == snapshot.PullRequest.HeadSHA {
			latest[review.Actor] = review
		}
	}
	actors := make([]string, 0, len(latest))
	for actor := range latest {
		actors = append(actors, actor)
	}
	sort.Strings(actors)
	var decisions []indexedDecision
	for _, actor := range actors {
		authority, authorized := snapshot.AuthorizedActors[actor]
		if !authorized {
			continue
		}
		review := latest[actor]
		outcome := ""
		switch strings.ToUpper(review.State) {
		case "APPROVED":
			outcome = "allow"
		case "CHANGES_REQUESTED":
			outcome = "reject"
		default:
			continue
		}
		decisions = append(decisions, indexedDecision{
			ActorRef: "github-user:" + actor, Authority: authority, Outcome: outcome,
			ReviewID: review.ID, ObservedAt: review.SubmittedAt.UTC(),
		})
	}
	return index, decisions, nil
}

func buildCheckSet(repository string, snapshot Snapshot) (checkSetEvidence, error) {
	checks := append([]Check(nil), snapshot.Checks...)
	sort.Slice(checks, func(i, j int) bool {
		if checks[i].Name != checks[j].Name {
			return checks[i].Name < checks[j].Name
		}
		return checks[i].ID < checks[j].ID
	})
	result := checkSetEvidence{
		Schema: "goalrail.github-check-set/v1", Repository: repository,
		HeadSHA: snapshot.PullRequest.HeadSHA,
	}
	for _, check := range checks {
		if check.Name == RequiredCheck {
			continue
		}
		if check.ID <= 0 || !validCheckName(check.Name) || (check.AppSlug != "" && !domain.IsCanonicalID(check.AppSlug)) {
			return checkSetEvidence{}, errors.New("GitHub check contains an invalid bounded identity")
		}
		appRef := "github-app:unknown"
		if check.AppSlug != "" {
			appRef = "github-app:" + check.AppSlug
		}
		result.Checks = append(result.Checks, indexedCheck{
			ID: check.ID, Name: check.Name, Status: strings.ToLower(check.Status),
			Conclusion: strings.ToLower(check.Conclusion), AppRef: appRef, CompletedAt: utcPointer(check.CompletedAt),
		})
	}
	return result, nil
}

func reference(kind, identity string, value any) domain.ContentAddressedEvidenceReference {
	raw, _ := json.Marshal(value)
	return domain.ContentAddressedEvidenceReference{
		ArtifactKind: kind, Identity: identity, Version: "1",
		Digest: domain.DigestCanonicalJSON(raw), SourceRef: identity, AdapterID: AdapterID,
	}
}

func sortedIDs(values []int64) []int64 {
	result := append([]int64(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func utcPointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func StableReferenceSummary(packet domain.AdmissionPacket) string {
	return strconv.Itoa(len(packet.Evidence)) + " provider-neutral evidence references"
}

func ContainsProhibitedPacketContent(raw []byte, prohibited ...string) bool {
	lower := bytes.ToLower(raw)
	for _, value := range prohibited {
		if value != "" && bytes.Contains(lower, bytes.ToLower([]byte(value))) {
			return true
		}
	}
	return false
}
