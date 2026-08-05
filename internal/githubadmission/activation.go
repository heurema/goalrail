package githubadmission

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/doctor"
)

var ErrUnsupportedRuleset = errors.New("GitHub ruleset syntax is not supported by the bounded observer")

type RequiredCheckObservation struct {
	Target         string
	Checks         []string
	ObservedAt     time.Time
	EvidenceSource string
}

type ActivationReader interface {
	RequiredChecks(context.Context, Repository, string) (RequiredCheckObservation, error)
}

type Observer struct {
	Repository Repository
	Target     string
	Reader     ActivationReader
	Now        func() time.Time
	MaxAge     time.Duration
}

func (observer Observer) ObserveActivation(ctx context.Context, request doctor.ActivationRequest) doctor.ActivationEvidence {
	evidence := doctor.ActivationEvidence{
		AdapterID: request.AdapterID, Provider: Provider,
		Boundary: "github-protected-branch", RequiredCheck: request.RequiredCheck,
		Freshness: doctor.ActivationFreshUnverified,
	}
	if request.AdapterID != AdapterID {
		evidence.State = doctor.ActivationUnknown
		evidence.Detail = "configured adapter does not match the GitHub admission observer"
		return evidence
	}
	if observer.Reader == nil {
		evidence.State = doctor.ActivationUnknown
		evidence.Detail = "GitHub read-only authentication is unavailable"
		return evidence
	}
	if strings.TrimSpace(observer.Target) == "" {
		evidence.State = doctor.ActivationUnknown
		evidence.Detail = "protected GitHub target is not configured"
		return evidence
	}
	observation, err := observer.Reader.RequiredChecks(ctx, observer.Repository, observer.Target)
	if err != nil {
		var status HTTPStatusError
		switch {
		case errors.As(err, &status) && status.Status == http.StatusForbidden:
			evidence.State = doctor.ActivationUnknown
			evidence.Detail = "GitHub denied read-only protection observation; activation remains unknown"
		case errors.Is(err, ErrUnsupportedRuleset):
			evidence.State = doctor.ActivationUnknown
			evidence.Detail = "GitHub protection uses unsupported ruleset syntax"
		default:
			evidence.State = doctor.ActivationUnknown
			evidence.Detail = "GitHub protected-target state could not be observed"
		}
		return evidence
	}
	evidence.ObservedTarget = observation.Target
	evidence.ObservedAt = observation.ObservedAt.UTC()
	evidence.EvidenceSource = observation.EvidenceSource
	if observation.Target != observer.Target {
		evidence.State = doctor.ActivationUnknown
		evidence.Detail = "GitHub protection evidence names a different target"
		return evidence
	}
	now := time.Now
	if observer.Now != nil {
		now = observer.Now
	}
	maxAge := observer.MaxAge
	if maxAge <= 0 {
		maxAge = 15 * time.Minute
	}
	current := now().UTC()
	if observation.ObservedAt.IsZero() || observation.ObservedAt.After(current.Add(time.Minute)) || current.Sub(observation.ObservedAt.UTC()) > maxAge {
		evidence.State = doctor.ActivationUnknown
		evidence.Freshness = doctor.ActivationFreshStale
		evidence.Detail = "GitHub protection evidence is stale or has an untrusted time; activation remains unknown"
		return evidence
	}
	evidence.Freshness = doctor.ActivationFreshCurrent
	for _, check := range observation.Checks {
		if check == request.RequiredCheck {
			evidence.State = doctor.ActivationActive
			evidence.Detail = "the exact Goalrail check is currently required at the protected target"
			return evidence
		}
	}
	evidence.State = doctor.ActivationInactive
	evidence.Detail = "the protected target does not currently require the exact Goalrail check"
	return evidence
}

type rulesetListItem struct {
	ID          int64  `json:"id"`
	Target      string `json:"target"`
	Enforcement string `json:"enforcement"`
}

type rulesetWire struct {
	ID          int64  `json:"id"`
	Target      string `json:"target"`
	Enforcement string `json:"enforcement"`
	Conditions  struct {
		RefName struct {
			Include []string `json:"include"`
			Exclude []string `json:"exclude"`
		} `json:"ref_name"`
	} `json:"conditions"`
	Rules []struct {
		Type       string `json:"type"`
		Parameters struct {
			RequiredStatusChecks []struct {
				Context string `json:"context"`
			} `json:"required_status_checks"`
		} `json:"parameters"`
	} `json:"rules"`
}

type branchProtectionWire struct {
	Contexts []string `json:"contexts"`
	Checks   []struct {
		Context string `json:"context"`
	} `json:"checks"`
}

func (reader *RESTReader) RequiredChecks(ctx context.Context, repository Repository, target string) (RequiredCheckObservation, error) {
	if !validTarget(target) {
		return RequiredCheckObservation{}, errors.New("GitHub protected target must be one canonical branch name")
	}
	prefix := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name)
	checks := make(map[string]struct{})
	var observedAt time.Time
	var list []rulesetListItem
	when, err := reader.get(ctx, prefix+"/rulesets?includes_parents=true&per_page=100", &list)
	if err != nil {
		var status HTTPStatusError
		if !errors.As(err, &status) || status.Status != http.StatusNotFound {
			return RequiredCheckObservation{}, err
		}
	} else {
		observedAt = later(observedAt, when)
		if len(list) > 100 {
			return RequiredCheckObservation{}, errors.New("GitHub ruleset list exceeds the bounded observer")
		}
		for _, item := range list {
			if item.ID <= 0 || item.Target != "branch" || strings.EqualFold(item.Enforcement, "disabled") {
				continue
			}
			var ruleset rulesetWire
			when, err := reader.get(ctx, prefix+"/rulesets/"+strconv.FormatInt(item.ID, 10), &ruleset)
			if err != nil {
				return RequiredCheckObservation{}, err
			}
			observedAt = later(observedAt, when)
			applies, err := rulesetApplies(ruleset, target)
			if err != nil {
				return RequiredCheckObservation{}, err
			}
			if !applies {
				continue
			}
			for _, rule := range ruleset.Rules {
				if rule.Type != "required_status_checks" {
					continue
				}
				for _, required := range rule.Parameters.RequiredStatusChecks {
					if !validCheckName(required.Context) {
						return RequiredCheckObservation{}, ErrUnsupportedRuleset
					}
					checks[required.Context] = struct{}{}
				}
			}
		}
	}

	var protection branchProtectionWire
	when, protectionErr := reader.get(ctx, prefix+"/branches/"+url.PathEscape(target)+"/protection/required_status_checks", &protection)
	if protectionErr != nil {
		var status HTTPStatusError
		if !errors.As(protectionErr, &status) || status.Status != http.StatusNotFound {
			return RequiredCheckObservation{}, protectionErr
		}
	} else {
		observedAt = later(observedAt, when)
		for _, context := range protection.Contexts {
			if !validCheckName(context) {
				return RequiredCheckObservation{}, ErrUnsupportedRuleset
			}
			checks[context] = struct{}{}
		}
		for _, check := range protection.Checks {
			if !validCheckName(check.Context) {
				return RequiredCheckObservation{}, ErrUnsupportedRuleset
			}
			checks[check.Context] = struct{}{}
		}
	}
	if observedAt.IsZero() {
		return RequiredCheckObservation{}, errors.New("GitHub protection target was not observed")
	}
	result := RequiredCheckObservation{
		Target: target, ObservedAt: observedAt.UTC(),
		EvidenceSource: fmt.Sprintf("github:repos/%s/branches/%s/protection", repository.String(), target),
	}
	for check := range checks {
		result.Checks = append(result.Checks, check)
	}
	sort.Strings(result.Checks)
	return result, nil
}

func rulesetApplies(ruleset rulesetWire, target string) (bool, error) {
	if ruleset.Target != "branch" || strings.EqualFold(ruleset.Enforcement, "disabled") {
		return false, nil
	}
	want := "refs/heads/" + target
	for _, excluded := range ruleset.Conditions.RefName.Exclude {
		if unsupportedRefPattern(excluded) {
			return false, ErrUnsupportedRuleset
		}
		if excluded == want {
			return false, nil
		}
	}
	for _, included := range ruleset.Conditions.RefName.Include {
		if unsupportedRefPattern(included) {
			return false, ErrUnsupportedRuleset
		}
		if included == want {
			return true, nil
		}
	}
	return false, nil
}

func unsupportedRefPattern(value string) bool {
	return value == "" || strings.ContainsAny(value, "*?[]") || strings.HasPrefix(value, "~")
}

func validTarget(value string) bool {
	return value != "" && !strings.Contains(value, "..") && !strings.HasPrefix(value, "/") && !strings.HasSuffix(value, "/") && !strings.ContainsAny(value, "\\ ~^:?*[")
}

func validCheckName(value string) bool {
	return strings.TrimSpace(value) == value && value != "" && len(value) <= 128 && !strings.ContainsRune(value, 0) && !strings.ContainsAny(value, "\r\n")
}
