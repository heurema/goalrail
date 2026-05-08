package qualificationfeed

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	defaultLimit = 50
	maxLimit     = 100
)

var ErrMembershipRequired = errors.New("active organization membership is required")

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

type Store interface {
	List(context.Context, spine.QualificationFeedFilter) ([]spine.QualificationFeedRecord, error)
}

type Service struct {
	Store Store
}

type ListInput struct {
	Membership    spine.OrganizationMembership
	ProjectID     spine.ProjectID
	RepoBindingID spine.RepoBindingID
	GoalState     spine.GoalState
	Limit         int
}

func NewService(store Store) *Service {
	return &Service{Store: store}
}

func (s *Service) List(ctx context.Context, input ListInput) (spine.QualificationFeed, error) {
	if err := authorize(input.Membership); err != nil {
		return spine.QualificationFeed{}, err
	}
	limit, err := normalizeLimit(input.Limit)
	if err != nil {
		return spine.QualificationFeed{}, err
	}
	if err := validateGoalState(input.GoalState); err != nil {
		return spine.QualificationFeed{}, err
	}
	if s.Store == nil {
		return spine.QualificationFeed{}, errors.New("qualification feed store is nil")
	}

	records, err := s.Store.List(ctx, spine.QualificationFeedFilter{
		OrganizationID: input.Membership.OrganizationID,
		ProjectID:      input.ProjectID,
		RepoBindingID:  input.RepoBindingID,
		GoalState:      input.GoalState,
		Limit:          limit,
	})
	if err != nil {
		return spine.QualificationFeed{}, fmt.Errorf("list qualification feed: %w", err)
	}

	items := make([]spine.QualificationFeedItem, 0, len(records))
	for _, record := range records {
		items = append(items, itemFromRecord(record))
	}
	return spine.QualificationFeed{Items: items}, nil
}

func authorize(membership spine.OrganizationMembership) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
	}
	return nil
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultLimit, nil
	}
	if limit < 0 {
		return 0, &ValidationError{Field: "limit", Message: "must be positive"}
	}
	if limit > maxLimit {
		return 0, &ValidationError{Field: "limit", Message: "must be <= 100"}
	}
	return limit, nil
}

func validateGoalState(state spine.GoalState) error {
	switch state {
	case "", spine.GoalStateCreated, spine.GoalStateNeedsClarification, spine.GoalStateReadyForContractSeed, spine.GoalStateRejected:
		return nil
	default:
		return &ValidationError{Field: "state", Message: "must be a known goal state"}
	}
}

func itemFromRecord(record spine.QualificationFeedRecord) spine.QualificationFeedItem {
	lane, action := laneAndAction(record)
	reasons := record.ReadinessReasonCodes
	if reasons == nil {
		reasons = []spine.GoalReadinessReasonCode{}
	}
	return spine.QualificationFeedItem{
		IntakeID:           record.IntakeID,
		GoalID:             record.GoalID,
		OrganizationID:     record.OrganizationID,
		ProjectID:          record.ProjectID,
		RepoBindingID:      record.RepoBindingID,
		RepositoryFullName: record.RepositoryFullName,
		Title:              record.Title,
		Lane:               lane,
		IntakeState:        record.IntakeState,
		GoalState:          record.GoalState,
		Readiness: spine.QualificationReadinessSnapshot{
			Ready:       record.GoalState == spine.GoalStateReadyForContractSeed,
			ReasonCodes: reasons,
			Source:      spine.QualificationReadinessSourceGoalSnapshot,
		},
		OpenClarificationRequest: record.OpenClarificationRequest,
		LinkedContract:           record.LinkedContract,
		NextAction:               action,
		CreatedAt:                record.CreatedAt,
	}
}

func laneAndAction(record spine.QualificationFeedRecord) (spine.QualificationLane, spine.QualificationNextAction) {
	if record.LinkedContract != nil {
		return spine.QualificationLaneContract, contractAction(record.LinkedContract.State)
	}

	switch record.GoalState {
	case spine.GoalStateNeedsClarification:
		if record.OpenClarificationRequest != nil {
			return spine.QualificationLaneClarification, spine.QualificationNextAction{
				Kind:      spine.QualificationNextActionAnswerClarification,
				Available: true,
				Blocking:  true,
			}
		}
		return spine.QualificationLaneQualification, spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionContinueGoal,
			Available: true,
			Blocking:  true,
		}
	case spine.GoalStateReadyForContractSeed:
		return spine.QualificationLaneQualification, spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionDraftContract,
			Available: true,
			Blocking:  false,
		}
	case spine.GoalStateRejected:
		return spine.QualificationLaneBlocked, spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionBlocked,
			Available: false,
			Blocking:  true,
		}
	default:
		return spine.QualificationLaneQualification, spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionContinueGoal,
			Available: true,
			Blocking:  false,
		}
	}
}

func contractAction(state spine.ContractState) spine.QualificationNextAction {
	switch state {
	case spine.ContractStateSeeded:
		return spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionDraftContract,
			Available: true,
			Blocking:  false,
		}
	case spine.ContractStateDraft:
		return spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionUpdateContract,
			Available: true,
			Blocking:  false,
		}
	case spine.ContractStateReadyForApproval:
		return spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionApproveContract,
			Available: true,
			Blocking:  false,
		}
	case spine.ContractStateApproved:
		return spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionPlanWork,
			Available: true,
			Blocking:  false,
		}
	default:
		return spine.QualificationNextAction{
			Kind:      spine.QualificationNextActionNone,
			Available: false,
			Blocking:  false,
		}
	}
}
