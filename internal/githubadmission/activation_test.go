package githubadmission

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/doctor"
)

type activationReaderFunc func(context.Context, Repository, string) (RequiredCheckObservation, error)

func (function activationReaderFunc) RequiredChecks(ctx context.Context, repository Repository, target string) (RequiredCheckObservation, error) {
	return function(ctx, repository, target)
}

func TestActivationObservationIsCurrentExactAndReadOnly(t *testing.T) {
	now := time.Date(2026, 8, 5, 14, 0, 0, 0, time.UTC)
	request := doctor.ActivationRequest{AdapterID: AdapterID, RequiredCheck: RequiredCheck}
	observer := Observer{
		Repository: Repository{Owner: "acme", Name: "widget"}, Target: "main", Now: func() time.Time { return now },
		Reader: activationReaderFunc(func(context.Context, Repository, string) (RequiredCheckObservation, error) {
			return RequiredCheckObservation{Target: "main", Checks: []string{RequiredCheck}, ObservedAt: now.Add(-time.Minute), EvidenceSource: "github:repos/acme/widget/branches/main/protection"}, nil
		}),
	}
	evidence := observer.ObserveActivation(context.Background(), request)
	if evidence.State != doctor.ActivationActive || evidence.Freshness != doctor.ActivationFreshCurrent || evidence.ObservedTarget != "main" {
		t.Fatalf("active evidence = %+v", evidence)
	}

	observer.Reader = activationReaderFunc(func(context.Context, Repository, string) (RequiredCheckObservation, error) {
		return RequiredCheckObservation{Target: "main", Checks: []string{"other"}, ObservedAt: now.Add(-time.Minute), EvidenceSource: "github:repos/acme/widget/branches/main/protection"}, nil
	})
	if evidence := observer.ObserveActivation(context.Background(), request); evidence.State != doctor.ActivationInactive {
		t.Fatalf("inactive evidence = %+v", evidence)
	}
}

func TestActivationObservationMapsEveryUncertainStateExplicitly(t *testing.T) {
	now := time.Date(2026, 8, 5, 14, 0, 0, 0, time.UTC)
	request := doctor.ActivationRequest{AdapterID: AdapterID, RequiredCheck: RequiredCheck}
	base := Observer{Repository: Repository{Owner: "acme", Name: "widget"}, Target: "main", Now: func() time.Time { return now }}
	cases := []struct {
		name   string
		reader ActivationReader
		want   doctor.ActivationState
	}{
		{name: "missing-auth", reader: nil, want: doctor.ActivationUnknown},
		{name: "access-denied", reader: activationReaderFunc(func(context.Context, Repository, string) (RequiredCheckObservation, error) {
			return RequiredCheckObservation{}, HTTPStatusError{Status: http.StatusForbidden, Route: "/rulesets"}
		}), want: doctor.ActivationUnknown},
		{name: "unsupported-ruleset", reader: activationReaderFunc(func(context.Context, Repository, string) (RequiredCheckObservation, error) {
			return RequiredCheckObservation{}, ErrUnsupportedRuleset
		}), want: doctor.ActivationUnknown},
		{name: "stale", reader: activationReaderFunc(func(context.Context, Repository, string) (RequiredCheckObservation, error) {
			return RequiredCheckObservation{Target: "main", Checks: []string{RequiredCheck}, ObservedAt: now.Add(-time.Hour), EvidenceSource: "github:ruleset/1"}, nil
		}), want: doctor.ActivationUnknown},
		{name: "target-mismatch", reader: activationReaderFunc(func(context.Context, Repository, string) (RequiredCheckObservation, error) {
			return RequiredCheckObservation{Target: "release", Checks: []string{RequiredCheck}, ObservedAt: now, EvidenceSource: "github:ruleset/1"}, nil
		}), want: doctor.ActivationUnknown},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			observer := base
			observer.Reader = test.reader
			if got := observer.ObserveActivation(context.Background(), request); got.State != test.want {
				t.Fatalf("state = %s, want %s: %+v", got.State, test.want, got)
			} else if test.name == "stale" && got.Freshness != doctor.ActivationFreshStale {
				t.Fatalf("stale freshness = %s, want %s: %+v", got.Freshness, doctor.ActivationFreshStale, got)
			}
		})
	}
	if !errors.Is(ErrUnsupportedRuleset, ErrUnsupportedRuleset) {
		t.Fatal("sentinel error is not stable")
	}
}

func TestRulesetMatcherRejectsUnboundedSyntax(t *testing.T) {
	var ruleset rulesetWire
	ruleset.Target = "branch"
	ruleset.Enforcement = "active"
	ruleset.Conditions.RefName.Include = []string{"refs/heads/main"}
	if applies, err := rulesetApplies(ruleset, "main"); err != nil || !applies {
		t.Fatalf("exact ruleset = %v, %v", applies, err)
	}
	ruleset.Conditions.RefName.Include = []string{"refs/heads/*"}
	if _, err := rulesetApplies(ruleset, "main"); !errors.Is(err, ErrUnsupportedRuleset) {
		t.Fatalf("glob ruleset error = %v", err)
	}
}
