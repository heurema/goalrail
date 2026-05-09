package store

import (
	"context"
	"encoding/json"
	"fmt"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresQualificationFeedStore struct {
	rows postgresRowsQuerier
	psql squirrel.StatementBuilderType
}

func NewPostgresQualificationFeedStore(pool *pgxpool.Pool) *PostgresQualificationFeedStore {
	db := newPostgresDB(pool)
	return NewPostgresQualificationFeedStoreWithRowsQuerier(db)
}

func NewPostgresQualificationFeedStoreWithRowsQuerier(rows postgresRowsQuerier) *PostgresQualificationFeedStore {
	return &PostgresQualificationFeedStore{
		rows: rows,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresQualificationFeedStore) List(ctx context.Context, filter spine.QualificationFeedFilter) ([]spine.QualificationFeedRecord, error) {
	if s.rows == nil {
		return nil, fmt.Errorf("qualification feed rows executor is nil")
	}
	orgID, err := uuidValue(filter.OrganizationID, "qualification feed organization id")
	if err != nil {
		return nil, err
	}

	stmt := s.psql.
		Select(qualificationFeedColumns()...).
		From("goals g").
		Join("intake_records ir ON ir.id = g.intake_id").
		Join("repo_bindings rb ON rb.id = g.repo_binding_id").
		LeftJoin("clarification_requests cr ON cr.goal_id = g.id AND cr.state = 'open'").
		LeftJoin("contracts c ON c.goal_id = g.id").
		Where(squirrel.Eq{
			"g.organization_id":  orgID,
			"ir.organization_id": orgID,
			"rb.organization_id": orgID,
		}).
		OrderBy("ir.created_at DESC", "ir.id DESC")

	if filter.ProjectID != "" {
		projectID, err := uuidValue(filter.ProjectID, "qualification feed project id")
		if err != nil {
			return nil, err
		}
		stmt = stmt.Where(squirrel.Eq{"g.project_id": projectID})
	}
	if filter.RepoBindingID != "" {
		repoBindingID, err := uuidValue(filter.RepoBindingID, "qualification feed repo binding id")
		if err != nil {
			return nil, err
		}
		stmt = stmt.Where(squirrel.Eq{"g.repo_binding_id": repoBindingID})
	}
	if filter.GoalState != "" {
		stmt = stmt.Where(squirrel.Eq{"g.state": filter.GoalState})
	}
	if filter.Limit > 0 {
		stmt = stmt.Limit(uint64(filter.Limit))
	}

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("list qualification feed SQL: %w", err)
	}
	rows, err := s.rows.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list qualification feed: %w", err)
	}
	defer rows.Close()

	var records []spine.QualificationFeedRecord
	for rows.Next() {
		record, err := scanQualificationFeedRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan qualification feed record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate qualification feed records: %w", err)
	}
	return records, nil
}

func scanQualificationFeedRecord(row pgx.Row) (spine.QualificationFeedRecord, error) {
	var record spine.QualificationFeedRecord
	var intakeID string
	var goalID string
	var organizationID string
	var projectID string
	var repoBindingID string
	var intakeState string
	var goalState string
	var readinessReasons []byte
	var clarificationID pgtype.UUID
	var clarificationState string
	var clarificationQuestions []byte
	var contractID pgtype.UUID
	var contractState string
	if err := row.Scan(
		&intakeID,
		&goalID,
		&organizationID,
		&projectID,
		&repoBindingID,
		&record.RepositoryFullName,
		&record.Title,
		&intakeState,
		&goalState,
		&readinessReasons,
		&clarificationID,
		&clarificationState,
		&clarificationQuestions,
		&contractID,
		&contractState,
		&record.CreatedAt,
	); err != nil {
		return spine.QualificationFeedRecord{}, err
	}

	record.IntakeID = spine.IntakeID(intakeID)
	record.GoalID = spine.GoalID(goalID)
	record.OrganizationID = spine.OrganizationID(organizationID)
	record.ProjectID = spine.ProjectID(projectID)
	record.RepoBindingID = spine.RepoBindingID(repoBindingID)
	record.IntakeState = spine.IntakeState(intakeState)
	record.GoalState = spine.GoalState(goalState)
	if err := json.Unmarshal(readinessReasons, &record.ReadinessReasonCodes); err != nil {
		return spine.QualificationFeedRecord{}, fmt.Errorf("unmarshal qualification feed readiness reason codes: %w", err)
	}
	if clarificationID.Valid {
		var questions []spine.ClarificationQuestion
		if err := json.Unmarshal(clarificationQuestions, &questions); err != nil {
			return spine.QualificationFeedRecord{}, fmt.Errorf("unmarshal qualification feed clarification questions: %w", err)
		}
		record.OpenClarificationRequest = &spine.QualificationOpenClarificationRequest{
			ID:        spine.ClarificationRequestID(uuidString(clarificationID)),
			State:     spine.ClarificationRequestState(clarificationState),
			Questions: questions,
		}
	}
	if contractID.Valid {
		record.LinkedContract = &spine.QualificationLinkedContract{
			ID:    spine.ContractID(uuidString(contractID)),
			State: spine.ContractState(contractState),
		}
	}
	record.CreatedAt = record.CreatedAt.UTC()
	return record, nil
}

func qualificationFeedColumns() []string {
	return []string{
		"ir.id",
		"g.id",
		"g.organization_id",
		"g.project_id",
		"g.repo_binding_id",
		"rb.repository_full_name",
		"g.title",
		"ir.state",
		"g.state",
		"g.last_readiness_reason_codes",
		"cr.id",
		"COALESCE(cr.state, '')",
		"COALESCE(cr.questions, '[]'::jsonb)",
		"c.id",
		"COALESCE(c.state, '')",
		"ir.created_at",
	}
}
