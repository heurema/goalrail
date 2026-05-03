package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresIntakeStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresIntakeStore(pool *pgxpool.Pool) *PostgresIntakeStore {
	db := newPostgresDB(pool)
	return NewPostgresIntakeStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresIntakeStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresIntakeStore {
	return &PostgresIntakeStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresIntakeStore) Create(ctx context.Context, record spine.IntakeRecord) error {
	id, err := uuidValue(record.ID, "intake record id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(record.OrganizationID, "intake record organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(record.ProjectID, "intake record project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(record.RepoBindingID, "intake record repo binding id")
	if err != nil {
		return err
	}
	source, err := json.Marshal(record.Source)
	if err != nil {
		return fmt.Errorf("marshal intake source: %w", err)
	}
	requestAuthor, err := json.Marshal(record.RequestAuthor)
	if err != nil {
		return fmt.Errorf("marshal intake request author: %w", err)
	}
	intentOwner, err := json.Marshal(record.IntentOwner)
	if err != nil {
		return fmt.Errorf("marshal intake intent owner: %w", err)
	}

	createdAt := record.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	stmt := s.psql.
		Insert("intake_records").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"source",
			"title",
			"body",
			"request_author",
			"intent_owner",
			"state",
			"canonical_contract_created",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			repoBindingID,
			source,
			record.Title,
			record.Body,
			requestAuthor,
			intentOwner,
			record.State,
			record.CanonicalContractCreated,
			createdAt,
			createdAt,
		)

	if err := execSQL(ctx, s.exec, "create intake record", stmt); err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresIntakeStore) Get(ctx context.Context, id spine.IntakeID) (spine.IntakeRecord, bool, error) {
	intakeID, err := uuidValue(id, "intake record id")
	if err != nil {
		return spine.IntakeRecord{}, false, err
	}
	stmt := s.psql.
		Select(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"source",
			"title",
			"body",
			"request_author",
			"intent_owner",
			"state",
			"canonical_contract_created",
			"created_at",
		).
		From("intake_records").
		Where(squirrel.Eq{"id": intakeID})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.IntakeRecord{}, false, fmt.Errorf("get intake record SQL: %w", err)
	}

	record, err := scanIntakeRecord(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.IntakeRecord{}, false, nil
		}
		return spine.IntakeRecord{}, false, err
	}
	return record, true, nil
}

func scanIntakeRecord(row pgx.Row) (spine.IntakeRecord, error) {
	var record spine.IntakeRecord
	var id string
	var organizationID string
	var projectID string
	var repoBindingID string
	var source []byte
	var requestAuthor []byte
	var intentOwner []byte
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&repoBindingID,
		&source,
		&record.Title,
		&record.Body,
		&requestAuthor,
		&intentOwner,
		&record.State,
		&record.CanonicalContractCreated,
		&record.CreatedAt,
	); err != nil {
		return spine.IntakeRecord{}, err
	}
	record.ID = spine.IntakeID(id)
	record.OrganizationID = spine.OrganizationID(organizationID)
	record.ProjectID = spine.ProjectID(projectID)
	record.RepoBindingID = spine.RepoBindingID(repoBindingID)
	if err := json.Unmarshal(source, &record.Source); err != nil {
		return spine.IntakeRecord{}, fmt.Errorf("unmarshal intake source: %w", err)
	}
	if err := json.Unmarshal(requestAuthor, &record.RequestAuthor); err != nil {
		return spine.IntakeRecord{}, fmt.Errorf("unmarshal intake request author: %w", err)
	}
	if err := json.Unmarshal(intentOwner, &record.IntentOwner); err != nil {
		return spine.IntakeRecord{}, fmt.Errorf("unmarshal intake intent owner: %w", err)
	}
	record.CreatedAt = record.CreatedAt.UTC()
	return record, nil
}
