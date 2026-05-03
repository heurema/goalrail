package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	errPostgresClarificationGoalNotFound = errors.New("clarification goal not found")
)

type PostgresClarificationRequestStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresClarificationRequestStore(pool *pgxpool.Pool) *PostgresClarificationRequestStore {
	db := newPostgresDB(pool)
	return NewPostgresClarificationRequestStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresClarificationRequestStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresClarificationRequestStore {
	return &PostgresClarificationRequestStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresClarificationRequestStore) Create(ctx context.Context, created spine.ClarificationRequest) error {
	id, err := uuidValue(created.ID, "clarification request id")
	if err != nil {
		return err
	}
	goalID, err := uuidValue(created.GoalID, "clarification request goal id")
	if err != nil {
		return err
	}
	reasonCodes := created.ReasonCodes
	if reasonCodes == nil {
		reasonCodes = []spine.GoalReadinessReasonCode{}
	}
	reasons, err := json.Marshal(reasonCodes)
	if err != nil {
		return fmt.Errorf("marshal clarification request reason codes: %w", err)
	}
	questions := created.Questions
	if questions == nil {
		questions = []spine.ClarificationQuestion{}
	}
	questionBytes, err := json.Marshal(questions)
	if err != nil {
		return fmt.Errorf("marshal clarification request questions: %w", err)
	}
	target, err := json.Marshal(created.Target)
	if err != nil {
		return fmt.Errorf("marshal clarification request target: %w", err)
	}
	createdAt := created.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	selectGoalContext := s.psql.
		Select().
		Column("?", id).
		Column("organization_id").
		Column("project_id").
		Column("repo_binding_id").
		Column("id").
		Column("?", created.State).
		Column("?", reasons).
		Column("?", questionBytes).
		Column("?", target).
		Column("?", createdAt).
		Column("?", createdAt).
		From("goals").
		Where(squirrel.Eq{"id": goalID})

	stmt := s.psql.
		Insert("clarification_requests").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"goal_id",
			"state",
			"reason_codes",
			"questions",
			"target",
			"created_at",
			"updated_at",
		).
		Select(selectGoalContext)

	tag, err := execClarificationSQLTag(ctx, s.exec, "create clarification request", stmt)
	if err != nil {
		if uniqueViolationConstraint(err) == "clarification_requests_one_open_per_goal_idx" {
			return ErrClarificationRequestAlreadyOpen
		}
		if isUniqueViolation(err) {
			return ErrClarificationRequestAlreadyExists
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return errPostgresClarificationGoalNotFound
	}
	return nil
}

func (s *PostgresClarificationRequestStore) Get(ctx context.Context, id spine.ClarificationRequestID) (spine.ClarificationRequest, bool, error) {
	requestID, err := uuidValue(id, "clarification request id")
	if err != nil {
		return spine.ClarificationRequest{}, false, err
	}
	return s.getOne(ctx, "get clarification request", squirrel.Eq{"id": requestID})
}

func (s *PostgresClarificationRequestStore) GetOpenByGoalID(ctx context.Context, goalID spine.GoalID) (spine.ClarificationRequest, bool, error) {
	goalUUID, err := uuidValue(goalID, "clarification request goal id")
	if err != nil {
		return spine.ClarificationRequest{}, false, err
	}
	return s.getOne(ctx, "get open clarification request by goal id", squirrel.Eq{
		"goal_id": goalUUID,
		"state":   spine.ClarificationRequestStateOpen,
	})
}

func (s *PostgresClarificationRequestStore) UpdateState(ctx context.Context, id spine.ClarificationRequestID, state spine.ClarificationRequestState) (spine.ClarificationRequest, bool, error) {
	requestID, err := uuidValue(id, "clarification request id")
	if err != nil {
		return spine.ClarificationRequest{}, false, err
	}
	stmt := s.psql.
		Update("clarification_requests").
		Set("state", state).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": requestID}).
		Suffix(returningClarificationRequestColumns())

	request, ok, err := s.queryRequest(ctx, "update clarification request state", stmt)
	if err != nil {
		if uniqueViolationConstraint(err) == "clarification_requests_one_open_per_goal_idx" {
			return spine.ClarificationRequest{}, true, ErrClarificationRequestAlreadyOpen
		}
		return spine.ClarificationRequest{}, false, err
	}
	return request, ok, nil
}

func (s *PostgresClarificationRequestStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.ClarificationRequest, bool, error) {
	stmt := s.psql.
		Select(clarificationRequestColumns()...).
		From("clarification_requests").
		Where(where)

	return s.queryRequest(ctx, op, stmt)
}

func (s *PostgresClarificationRequestStore) queryRequest(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.ClarificationRequest, bool, error) {
	row, err := queryClarificationRow(ctx, s.query, op, sqlizer)
	if err != nil {
		return spine.ClarificationRequest{}, false, err
	}
	request, err := scanClarificationRequest(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ClarificationRequest{}, false, nil
		}
		return spine.ClarificationRequest{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return request, true, nil
}

func scanClarificationRequest(row pgx.Row) (spine.ClarificationRequest, error) {
	var request spine.ClarificationRequest
	var id string
	var goalID string
	var reasonCodes []byte
	var questions []byte
	var target []byte
	if err := row.Scan(
		&id,
		&goalID,
		&request.State,
		&reasonCodes,
		&questions,
		&target,
		&request.CreatedAt,
	); err != nil {
		return spine.ClarificationRequest{}, err
	}
	request.ID = spine.ClarificationRequestID(id)
	request.GoalID = spine.GoalID(goalID)
	if err := json.Unmarshal(reasonCodes, &request.ReasonCodes); err != nil {
		return spine.ClarificationRequest{}, fmt.Errorf("unmarshal clarification request reason codes: %w", err)
	}
	if err := json.Unmarshal(questions, &request.Questions); err != nil {
		return spine.ClarificationRequest{}, fmt.Errorf("unmarshal clarification request questions: %w", err)
	}
	if err := json.Unmarshal(target, &request.Target); err != nil {
		return spine.ClarificationRequest{}, fmt.Errorf("unmarshal clarification request target: %w", err)
	}
	request.CreatedAt = request.CreatedAt.UTC()
	return request, nil
}

func clarificationRequestColumns() []string {
	return []string{
		"id",
		"goal_id",
		"state",
		"reason_codes",
		"questions",
		"target",
		"created_at",
	}
}

func returningClarificationRequestColumns() string {
	return "RETURNING id, goal_id, state, reason_codes, questions, target, created_at"
}

type PostgresClarificationAnswerStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresClarificationAnswerStore(pool *pgxpool.Pool) *PostgresClarificationAnswerStore {
	db := newPostgresDB(pool)
	return NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresClarificationAnswerStore {
	return &PostgresClarificationAnswerStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresClarificationAnswerStore) Create(ctx context.Context, created spine.ClarificationAnswer) error {
	id, err := uuidValue(created.ID, "clarification answer id")
	if err != nil {
		return err
	}
	requestID, err := uuidValue(created.RequestID, "clarification answer request id")
	if err != nil {
		return err
	}
	answers := created.Answers
	if answers == nil {
		answers = []spine.ClarificationAnswerItem{}
	}
	answerBytes, err := json.Marshal(answers)
	if err != nil {
		return fmt.Errorf("marshal clarification answer items: %w", err)
	}
	submittedBy, err := json.Marshal(created.SubmittedBy)
	if err != nil {
		return fmt.Errorf("marshal clarification answer submitted by: %w", err)
	}
	createdAt := created.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	selectRequestContext := s.psql.
		Select().
		Column("?", id).
		Column("organization_id").
		Column("project_id").
		Column("repo_binding_id").
		Column("goal_id").
		Column("id").
		Column("?", submittedBy).
		Column("?", answerBytes).
		Column("?", createdAt).
		Column("?", createdAt).
		From("clarification_requests").
		Where(squirrel.Eq{"id": requestID})

	stmt := s.psql.
		Insert("clarification_answers").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"goal_id",
			"clarification_request_id",
			"submitted_by",
			"answers",
			"created_at",
			"updated_at",
		).
		Select(selectRequestContext)

	if err := execClarificationSQL(ctx, s.exec, "create clarification answer", stmt); err != nil {
		if uniqueViolationConstraint(err) == "clarification_answers_request_id_unique" {
			return ErrClarificationAnswerAlreadyRecorded
		}
		if isUniqueViolation(err) {
			return ErrClarificationAnswerAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresClarificationAnswerStore) Get(ctx context.Context, id spine.ClarificationAnswerID) (spine.ClarificationAnswer, bool, error) {
	answerID, err := uuidValue(id, "clarification answer id")
	if err != nil {
		return spine.ClarificationAnswer{}, false, err
	}
	return s.getOne(ctx, "get clarification answer", squirrel.Eq{"id": answerID})
}

func (s *PostgresClarificationAnswerStore) GetByRequestID(ctx context.Context, requestID spine.ClarificationRequestID) (spine.ClarificationAnswer, bool, error) {
	requestUUID, err := uuidValue(requestID, "clarification answer request id")
	if err != nil {
		return spine.ClarificationAnswer{}, false, err
	}
	return s.getOne(ctx, "get clarification answer by request id", squirrel.Eq{"clarification_request_id": requestUUID})
}

func (s *PostgresClarificationAnswerStore) MarkApplied(ctx context.Context, id spine.ClarificationAnswerID, appliedBy spine.ActorRef, appliedAt time.Time) (bool, error) {
	answerID, err := uuidValue(id, "clarification answer id")
	if err != nil {
		return false, err
	}
	appliedByBytes, err := json.Marshal(appliedBy)
	if err != nil {
		return false, fmt.Errorf("marshal clarification answer applied by: %w", err)
	}
	stmt := s.psql.
		Update("clarification_answers").
		Set("applied", true).
		Set("applied_by", appliedByBytes).
		Set("applied_at", appliedAt.UTC()).
		Set("updated_at", appliedAt.UTC()).
		Where(squirrel.Eq{"id": answerID}).
		Where(squirrel.Eq{"applied": false})

	tag, err := execClarificationSQLTag(ctx, s.exec, "mark clarification answer applied", stmt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *PostgresClarificationAnswerStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.ClarificationAnswer, bool, error) {
	stmt := s.psql.
		Select(clarificationAnswerColumns()...).
		From("clarification_answers").
		Where(where)

	return s.queryAnswer(ctx, op, stmt)
}

func (s *PostgresClarificationAnswerStore) queryAnswer(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.ClarificationAnswer, bool, error) {
	row, err := queryClarificationRow(ctx, s.query, op, sqlizer)
	if err != nil {
		return spine.ClarificationAnswer{}, false, err
	}
	answer, err := scanClarificationAnswer(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ClarificationAnswer{}, false, nil
		}
		return spine.ClarificationAnswer{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return answer, true, nil
}

func scanClarificationAnswer(row pgx.Row) (spine.ClarificationAnswer, error) {
	var answer spine.ClarificationAnswer
	var id string
	var requestID string
	var goalID string
	var submittedBy []byte
	var answers []byte
	if err := row.Scan(
		&id,
		&requestID,
		&goalID,
		&submittedBy,
		&answers,
		&answer.CreatedAt,
	); err != nil {
		return spine.ClarificationAnswer{}, err
	}
	answer.ID = spine.ClarificationAnswerID(id)
	answer.RequestID = spine.ClarificationRequestID(requestID)
	answer.GoalID = spine.GoalID(goalID)
	answer.State = spine.ClarificationAnswerStateRecorded
	if err := json.Unmarshal(submittedBy, &answer.SubmittedBy); err != nil {
		return spine.ClarificationAnswer{}, fmt.Errorf("unmarshal clarification answer submitted by: %w", err)
	}
	if err := json.Unmarshal(answers, &answer.Answers); err != nil {
		return spine.ClarificationAnswer{}, fmt.Errorf("unmarshal clarification answer items: %w", err)
	}
	answer.CreatedAt = answer.CreatedAt.UTC()
	return answer, nil
}

func clarificationAnswerColumns() []string {
	return []string{
		"id",
		"clarification_request_id",
		"goal_id",
		"submitted_by",
		"answers",
		"created_at",
	}
}

func queryClarificationRow(ctx context.Context, query postgresRowQuerier, op string, sqlizer squirrel.Sqlizer) (pgx.Row, error) {
	if query == nil {
		return nil, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s SQL: %w", op, err)
	}
	return query.QueryRow(ctx, sqlText, args...), nil
}

func execClarificationSQL(ctx context.Context, exec postgresExecer, op string, sqlizer squirrel.Sqlizer) error {
	_, err := execClarificationSQLTag(ctx, exec, op, sqlizer)
	return err
}

func execClarificationSQLTag(ctx context.Context, exec postgresExecer, op string, sqlizer squirrel.Sqlizer) (pgconn.CommandTag, error) {
	if exec == nil {
		return pgconn.CommandTag{}, fmt.Errorf("%s executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return pgconn.CommandTag{}, fmt.Errorf("%s SQL: %w", op, err)
	}
	tag, err := exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return pgconn.CommandTag{}, fmt.Errorf("%s: %w", op, err)
	}
	return tag, nil
}
