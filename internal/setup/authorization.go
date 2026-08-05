package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/heurema/goalrail/internal/domain"
)

type ConsentDecision string
type ConsentScope string

const (
	ConsentGrant  ConsentDecision = "grant"
	ConsentRefuse ConsentDecision = "refuse"

	ConsentExactPlan ConsentScope = "exact_plan"
	ConsentGeneral   ConsentScope = "general"
)

var (
	ErrAuthorizationMissing  = errors.New("setup authorization is missing")
	ErrAuthorizationRefused  = errors.New("setup authorization was refused")
	ErrAuthorizationGeneral  = errors.New("general consent cannot authorize setup")
	ErrAuthorizationInvalid  = errors.New("setup authorization is invalid")
	ErrAuthorizationStale    = errors.New("setup authorization is stale")
	ErrAuthorizationConsumed = errors.New("setup authorization attempt is already consumed")
	ErrPlanIncomplete        = errors.New("setup plan is incomplete")
	ErrPlanInvalid           = errors.New("setup plan is invalid")
)

// Consent records the owner's explicit decision and its scope. Only a grant
// carrying a canonical authorization reference for this exact plan is usable.
type Consent struct {
	Decision      ConsentDecision
	Scope         ConsentScope
	Authorization *domain.PlanAuthorizationReference
}

// BoundAuthorization is an immutable plan binding with a single pre-mutation
// attempt. It deliberately exposes no mutation method.
type BoundAuthorization struct {
	mu                    sync.Mutex
	consumed              bool
	options               PlanOptions
	planArtifact          domain.CanonicalArtifact
	authorizationArtifact domain.CanonicalArtifact
}

// AuthorizedAttempt is the proof passed to the later apply boundary. Both
// artifacts are immutable and return copies of their canonical bytes.
type AuthorizedAttempt struct {
	planArtifact          domain.CanonicalArtifact
	authorizationArtifact domain.CanonicalArtifact
}

// BindAuthorization validates explicit consent against the exact canonical
// complete plan. It performs no filesystem or network mutation.
func BindAuthorization(plan Result, options PlanOptions, consent Consent) (*BoundAuthorization, error) {
	switch consent.Decision {
	case ConsentRefuse:
		return nil, ErrAuthorizationRefused
	case ConsentGrant:
	default:
		return nil, ErrAuthorizationMissing
	}
	if consent.Scope == ConsentGeneral {
		return nil, ErrAuthorizationGeneral
	}
	if consent.Scope != ConsentExactPlan || consent.Authorization == nil {
		return nil, ErrAuthorizationMissing
	}

	planArtifact, err := validateCompletePlan(plan)
	if err != nil {
		return nil, err
	}

	authorizationArtifact, err := domain.FreezePlanAuthorizationReference(*consent.Authorization)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthorizationInvalid, err)
	}
	authorization, err := domain.DecodePlanAuthorizationReference(bytes.NewReader(authorizationArtifact.CanonicalJSON()))
	if err != nil {
		return nil, fmt.Errorf("%w: canonical authorization could not be decoded: %v", ErrAuthorizationInvalid, err)
	}
	if authorization.ProjectID != plan.Plan.ProjectID || authorization.PlanDigest != planArtifact.Digest() {
		return nil, ErrAuthorizationStale
	}

	normalized, err := normalizeOptions(clonePlanOptions(options))
	if err != nil {
		return nil, fmt.Errorf("%w: normalize revalidation inputs: %v", ErrAuthorizationInvalid, err)
	}

	return &BoundAuthorization{
		options:               normalized,
		planArtifact:          planArtifact,
		authorizationArtifact: authorizationArtifact,
	}, nil
}

// BeforeFirstMutation consumes the one authorized attempt and re-runs the
// complete read-only planner. Any inability to reproduce the exact canonical
// plan makes the authorization stale; callers must obtain a new plan and grant.
func (binding *BoundAuthorization) BeforeFirstMutation(ctx context.Context) (AuthorizedAttempt, error) {
	if binding == nil {
		return AuthorizedAttempt{}, ErrAuthorizationMissing
	}

	binding.mu.Lock()
	defer binding.mu.Unlock()
	if binding.consumed {
		return AuthorizedAttempt{}, ErrAuthorizationConsumed
	}
	binding.consumed = true

	current, err := Generate(ctx, clonePlanOptions(binding.options))
	if err != nil {
		return AuthorizedAttempt{}, fmt.Errorf("%w: re-resolve setup inputs: %v", ErrAuthorizationStale, err)
	}
	currentArtifact, err := validateCompletePlan(current)
	if err != nil {
		return AuthorizedAttempt{}, fmt.Errorf("%w: %v", ErrAuthorizationStale, err)
	}
	if currentArtifact.Digest() != binding.planArtifact.Digest() ||
		!bytes.Equal(currentArtifact.CanonicalJSON(), binding.planArtifact.CanonicalJSON()) {
		return AuthorizedAttempt{}, ErrAuthorizationStale
	}

	return AuthorizedAttempt{
		planArtifact:          binding.planArtifact,
		authorizationArtifact: binding.authorizationArtifact,
	}, nil
}

// PlanArtifact returns the exact complete plan admitted by revalidation.
func (attempt AuthorizedAttempt) PlanArtifact() domain.CanonicalArtifact {
	return attempt.planArtifact
}

// AuthorizationArtifact returns the canonical owner authorization bound to
// PlanArtifact's digest.
func (attempt AuthorizedAttempt) AuthorizationArtifact() domain.CanonicalArtifact {
	return attempt.authorizationArtifact
}

func validateCompletePlan(result Result) (domain.CanonicalArtifact, error) {
	artifact, err := domain.FreezeSetupPlan(result.Plan)
	if err != nil {
		return domain.CanonicalArtifact{}, fmt.Errorf("%w: %v", ErrPlanInvalid, err)
	}
	if artifact.Digest() != result.Artifact.Digest() ||
		!bytes.Equal(artifact.CanonicalJSON(), result.Artifact.CanonicalJSON()) {
		return domain.CanonicalArtifact{}, fmt.Errorf("%w: value and canonical artifact disagree", ErrPlanInvalid)
	}
	if result.Plan.State != domain.SetupPlanComplete || len(result.Issues) != 0 {
		return domain.CanonicalArtifact{}, ErrPlanIncomplete
	}
	return artifact, nil
}

func clonePlanOptions(options PlanOptions) PlanOptions {
	clone := options
	if options.Evidence != nil {
		clone.Evidence = &ReleaseEvidence{
			MetadataRaw: append([]byte(nil), options.Evidence.MetadataRaw...),
			ManifestRaw: append([]byte(nil), options.Evidence.ManifestRaw...),
		}
	}
	return clone
}
