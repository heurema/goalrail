package store

import "errors"

var (
	ErrAlreadyExists = errors.New("intake record already exists")

	ErrGoalAlreadyExists = errors.New("goal already exists")

	ErrClarificationRequestAlreadyExists = errors.New("clarification request already exists")
	ErrClarificationRequestAlreadyOpen   = errors.New("clarification request already open")

	ErrClarificationAnswerAlreadyExists   = errors.New("clarification answer already exists")
	ErrClarificationAnswerAlreadyRecorded = errors.New("clarification answer already recorded")

	ErrContractAlreadyExists = errors.New("contract already exists")
	ErrContractAlreadySeeded = errors.New("goal already has contract")
	ErrContractNotFound      = errors.New("contract not found")

	ErrContractSeedAlreadyExists = errors.New("contract seed already exists")
	ErrContractSeedAlreadySeeded = errors.New("contract seed already seeded")

	ErrContractDraftAlreadyExists  = errors.New("contract draft already exists")
	ErrContractDraftAlreadyDrafted = errors.New("contract seed already drafted")
	ErrContractDraftNotFound       = errors.New("contract draft not found")

	ErrApprovedContractAlreadyExists   = errors.New("approved contract already exists")
	ErrApprovedContractAlreadyApproved = errors.New("contract draft already approved")
	ErrApprovedContractNotFound        = errors.New("approved contract not found")

	ErrWorkItemAlreadyExists  = errors.New("work item already exists")
	ErrWorkItemAlreadyPlanned = errors.New("approved contract already planned")
	ErrWorkItemNotFound       = errors.New("work item not found")

	ErrWorkItemPlanAlreadyExists      = errors.New("work item plan already exists")
	ErrWorkItemPlanAlreadyPlanned     = errors.New("contract already has work item plan")
	ErrWorkItemPlanNotFound           = errors.New("work item plan not found")
	ErrWorkItemPlanProposalExists     = errors.New("work item plan proposal already exists")
	ErrWorkItemPlanAlreadyHasProposal = errors.New("work item plan already has proposal")
	ErrWorkItemPlanProposalNotFound   = errors.New("work item plan proposal not found")
)

type uniqueConstraintError struct {
	constraint string
	err        error
}

func (e uniqueConstraintError) Error() string {
	return e.err.Error()
}

func (e uniqueConstraintError) Unwrap() error {
	return e.err
}

func (e uniqueConstraintError) ConstraintName() string {
	return e.constraint
}

func wrapUniqueConstraint(err error) error {
	constraint := uniqueViolationConstraint(err)
	if constraint == "" {
		return err
	}
	return uniqueConstraintError{
		constraint: constraint,
		err:        err,
	}
}
