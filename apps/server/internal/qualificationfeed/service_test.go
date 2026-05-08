package qualificationfeed_test

import (
	"context"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/qualificationfeed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestListUsesMembershipOrganizationScope(t *testing.T) {
	store := &fakeStore{
		records: []spine.QualificationFeedRecord{
			qualificationRecord(spine.GoalStateCreated, nil, nil),
		},
	}
	service := qualificationfeed.NewService(store)

	feed, err := service.List(context.Background(), qualificationfeed.ListInput{
		Membership: activeMembership(),
		ProjectID:  "018f0000-0000-7000-8000-000000000003",
		Limit:      25,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(feed.Items))
	}
	if store.filter.OrganizationID != activeMembership().OrganizationID {
		t.Fatalf("OrganizationID = %q, want membership organization", store.filter.OrganizationID)
	}
	if store.filter.ProjectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("ProjectID = %q, want query project filter", store.filter.ProjectID)
	}
	if store.filter.Limit != 25 {
		t.Fatalf("Limit = %d, want 25", store.filter.Limit)
	}
}

func TestListRequiresActiveMembership(t *testing.T) {
	service := qualificationfeed.NewService(&fakeStore{})

	_, err := service.List(context.Background(), qualificationfeed.ListInput{
		Membership: spine.OrganizationMembership{State: spine.EntityStateInactive},
	})
	if err != qualificationfeed.ErrMembershipRequired {
		t.Fatalf("List() error = %v, want ErrMembershipRequired", err)
	}
}

func TestListMapsQualificationClarificationContractAndBlockedLanes(t *testing.T) {
	openRequest := &spine.QualificationOpenClarificationRequest{
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
	}
	draftContract := &spine.QualificationLinkedContract{
		ID:    "018f0000-0000-7000-8000-000000000c01",
		State: spine.ContractStateDraft,
	}
	approvedContract := &spine.QualificationLinkedContract{
		ID:    "018f0000-0000-7000-8000-000000000c02",
		State: spine.ContractStateApproved,
	}
	service := qualificationfeed.NewService(&fakeStore{
		records: []spine.QualificationFeedRecord{
			qualificationRecord(spine.GoalStateCreated, nil, nil),
			qualificationRecord(spine.GoalStateNeedsClarification, openRequest, nil),
			qualificationRecord(spine.GoalStateReadyForContractSeed, nil, nil),
			qualificationRecord(spine.GoalStateReadyForContractSeed, nil, draftContract),
			qualificationRecord(spine.GoalStateReadyForContractSeed, nil, approvedContract),
			qualificationRecord(spine.GoalStateRejected, nil, nil),
		},
	})

	feed, err := service.List(context.Background(), qualificationfeed.ListInput{Membership: activeMembership()})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := len(feed.Items), 6; got != want {
		t.Fatalf("items len = %d, want %d", got, want)
	}

	assertLaneAction(t, feed.Items[0], spine.QualificationLaneQualification, spine.QualificationNextActionContinueGoal, true, false)
	assertLaneAction(t, feed.Items[1], spine.QualificationLaneClarification, spine.QualificationNextActionAnswerClarification, true, true)
	if feed.Items[1].OpenClarificationRequest == nil || len(feed.Items[1].OpenClarificationRequest.Questions) != 1 {
		t.Fatalf("open clarification = %#v, want one question", feed.Items[1].OpenClarificationRequest)
	}
	assertLaneAction(t, feed.Items[2], spine.QualificationLaneQualification, spine.QualificationNextActionDraftContract, true, false)
	if !feed.Items[2].Readiness.Ready || feed.Items[2].Readiness.Source != spine.QualificationReadinessSourceGoalSnapshot {
		t.Fatalf("readiness = %#v, want ready goal snapshot", feed.Items[2].Readiness)
	}
	assertLaneAction(t, feed.Items[3], spine.QualificationLaneContract, spine.QualificationNextActionUpdateContract, true, false)
	if feed.Items[3].LinkedContract == nil || feed.Items[3].LinkedContract.ID != draftContract.ID {
		t.Fatalf("linked contract = %#v, want draft contract", feed.Items[3].LinkedContract)
	}
	assertLaneAction(t, feed.Items[4], spine.QualificationLaneContract, spine.QualificationNextActionPlanWork, true, false)
	assertLaneAction(t, feed.Items[5], spine.QualificationLaneBlocked, spine.QualificationNextActionBlocked, false, true)
}

func TestListRejectsUnknownStateFilter(t *testing.T) {
	service := qualificationfeed.NewService(&fakeStore{})

	_, err := service.List(context.Background(), qualificationfeed.ListInput{
		Membership: activeMembership(),
		GoalState:  "done",
	})
	if err == nil {
		t.Fatal("List() error = nil, want validation error")
	}
}

type fakeStore struct {
	filter  spine.QualificationFeedFilter
	records []spine.QualificationFeedRecord
	err     error
}

func (s *fakeStore) List(_ context.Context, filter spine.QualificationFeedFilter) ([]spine.QualificationFeedRecord, error) {
	s.filter = filter
	if s.err != nil {
		return nil, s.err
	}
	return append([]spine.QualificationFeedRecord(nil), s.records...), nil
}

func activeMembership() spine.OrganizationMembership {
	return spine.OrganizationMembership{
		ID:             "018f0000-0000-7000-8000-000000000901",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		UserID:         "018f0000-0000-7000-8000-000000000001",
		Role:           spine.OrganizationMembershipRoleMember,
		State:          spine.EntityStateActive,
	}
}

func qualificationRecord(
	state spine.GoalState,
	openRequest *spine.QualificationOpenClarificationRequest,
	linkedContract *spine.QualificationLinkedContract,
) spine.QualificationFeedRecord {
	return spine.QualificationFeedRecord{
		IntakeID:                 "018f0000-0000-7000-8000-000000000101",
		GoalID:                   "018f0000-0000-7000-8000-000000000201",
		OrganizationID:           "018f0000-0000-7000-8000-000000000002",
		ProjectID:                "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:            "018f0000-0000-7000-8000-000000000004",
		RepositoryFullName:       "heurema/goalrail",
		Title:                    "Improve billing error handling",
		IntakeState:              spine.IntakeStateReceived,
		GoalState:                state,
		ReadinessReasonCodes:     []spine.GoalReadinessReasonCode{spine.GoalReadinessReasonMissingScopeHint},
		OpenClarificationRequest: openRequest,
		LinkedContract:           linkedContract,
		CreatedAt:                time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC),
	}
}

func assertLaneAction(
	t *testing.T,
	item spine.QualificationFeedItem,
	lane spine.QualificationLane,
	action spine.QualificationNextActionKind,
	available bool,
	blocking bool,
) {
	t.Helper()
	if item.Lane != lane {
		t.Fatalf("lane = %q, want %q for item %#v", item.Lane, lane, item)
	}
	if item.NextAction.Kind != action || item.NextAction.Available != available || item.NextAction.Blocking != blocking {
		t.Fatalf("next_action = %#v, want kind=%q available=%v blocking=%v", item.NextAction, action, available, blocking)
	}
}
