package httpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/qualificationfeed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestQualificationFeedListRequiresBearerAuth(t *testing.T) {
	handler := httpserver.NewQualificationFeedHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, &fakeQualificationFeedService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.List), http.MethodGet, "/v1/qualification-feed", "", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.code, http.StatusUnauthorized)
	}
}

func TestQualificationFeedListPassesFiltersAndReturnsItems(t *testing.T) {
	service := &fakeQualificationFeedService{
		result: spine.QualificationFeed{
			Items: []spine.QualificationFeedItem{
				{
					IntakeID:           "018f0000-0000-7000-8000-000000000101",
					GoalID:             "018f0000-0000-7000-8000-000000000201",
					OrganizationID:     "018f0000-0000-7000-8000-000000000002",
					ProjectID:          "018f0000-0000-7000-8000-000000000003",
					RepoBindingID:      "018f0000-0000-7000-8000-000000000004",
					RepositoryFullName: "heurema/goalrail",
					Title:              "Improve billing error handling",
					Lane:               spine.QualificationLaneClarification,
					IntakeState:        spine.IntakeStateReceived,
					GoalState:          spine.GoalStateNeedsClarification,
					Readiness: spine.QualificationReadinessSnapshot{
						Ready:       false,
						ReasonCodes: []spine.GoalReadinessReasonCode{spine.GoalReadinessReasonMissingScopeHint},
						Source:      spine.QualificationReadinessSourceGoalSnapshot,
					},
					OpenClarificationRequest: &spine.QualificationOpenClarificationRequest{
						ID:    "018f0000-0000-7000-8000-000000000a01",
						State: spine.ClarificationRequestStateOpen,
						Questions: []spine.ClarificationQuestion{
							{
								ID:         "018f0000-0000-7000-8000-000000000a11",
								Text:       "What is the intended scope at a high level?",
								WhyNeeded:  "A scope hint is required before contract seed readiness.",
								AnswerType: spine.ClarificationAnswerTypeText,
								MapsTo:     spine.ClarificationMapsToGoalScopeHint,
							},
						},
					},
					NextAction: spine.QualificationNextAction{
						Kind:      spine.QualificationNextActionAnswerClarification,
						Available: true,
						Blocking:  true,
					},
				},
			},
		},
	}
	handler := httpserver.NewQualificationFeedHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, service)

	response := doAuthRequest(t, http.HandlerFunc(handler.List), http.MethodGet, "/v1/qualification-feed?project_id=018f0000-0000-7000-8000-000000000003&repo_binding_id=018f0000-0000-7000-8000-000000000004&state=needs_clarification&limit=25", "", "Bearer access-token")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	if service.input.Membership.OrganizationID != repoBindingProfile().OrganizationMembership.OrganizationID {
		t.Fatalf("membership organization = %q, want auth profile organization", service.input.Membership.OrganizationID)
	}
	if service.input.ProjectID != "018f0000-0000-7000-8000-000000000003" || service.input.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("filters = %#v, want project and repo binding filters", service.input)
	}
	if service.input.GoalState != spine.GoalStateNeedsClarification || service.input.Limit != 25 {
		t.Fatalf("state/limit = %q/%d, want needs_clarification/25", service.input.GoalState, service.input.Limit)
	}

	var body spine.QualificationFeed
	if err := json.Unmarshal([]byte(response.body), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Lane != spine.QualificationLaneClarification {
		t.Fatalf("items = %#v, want one clarification item", body.Items)
	}
}

func TestQualificationFeedListRejectsInvalidLimit(t *testing.T) {
	handler := httpserver.NewQualificationFeedHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeQualificationFeedService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.List), http.MethodGet, "/v1/qualification-feed?limit=nope", "", "Bearer access-token")
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
	}
}

type fakeQualificationFeedService struct {
	input  qualificationfeed.ListInput
	result spine.QualificationFeed
	err    error
}

func (s *fakeQualificationFeedService) List(_ context.Context, input qualificationfeed.ListInput) (spine.QualificationFeed, error) {
	s.input = input
	if s.err != nil {
		return spine.QualificationFeed{}, s.err
	}
	if s.result.Items == nil {
		return spine.QualificationFeed{}, errors.New("missing test result")
	}
	return s.result, nil
}
