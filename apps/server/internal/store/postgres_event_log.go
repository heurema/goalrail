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

type PostgresEventLog struct {
	exec  postgresExecer
	query postgresRowsQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresEventLog(pool *pgxpool.Pool) *PostgresEventLog {
	db := newPostgresDB(pool)
	return NewPostgresEventLogWithExecutorAndQuerier(db, db)
}

func NewPostgresEventLogWithExecutorAndQuerier(exec postgresExecer, query postgresRowsQuerier) *PostgresEventLog {
	return &PostgresEventLog{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (l *PostgresEventLog) Append(ctx context.Context, event spine.Event) error {
	id, err := uuidValue(event.ID, "event id")
	if err != nil {
		return err
	}
	entityID, err := uuidValue(event.EntityID, "event entity id")
	if err != nil {
		return err
	}
	orgID, err := nullableUUIDValue(event.OrganizationID, "event organization id")
	if err != nil {
		return err
	}
	projectID, err := nullableUUIDValue(event.ProjectID, "event project id")
	if err != nil {
		return err
	}
	repoBindingID, err := nullableUUIDValue(event.RepoBindingID, "event repo binding id")
	if err != nil {
		return err
	}
	artifactRefs, err := json.Marshal([]string{})
	if err != nil {
		return fmt.Errorf("marshal event artifact refs: %w", err)
	}

	stmt := l.psql.
		Insert("events").
		Columns(
			"id",
			"type",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"entity_type",
			"entity_id",
			"occurred_at",
			"payload",
			"artifact_refs",
		).
		Values(
			id,
			event.Type,
			orgID,
			projectID,
			repoBindingID,
			event.EntityType,
			entityID,
			event.Timestamp.UTC(),
			event.Payload,
			artifactRefs,
		)

	if err := l.execSQL(ctx, "append event", stmt); err != nil {
		return err
	}
	return nil
}

func (l *PostgresEventLog) Events(ctx context.Context) ([]spine.Event, error) {
	if l.query == nil {
		return nil, fmt.Errorf("event log query executor is nil")
	}
	stmt := l.psql.
		Select(
			"id",
			"type",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"entity_type",
			"entity_id",
			"occurred_at",
			"payload",
		).
		From("events").
		OrderBy("event_sequence ASC")

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("list events SQL: %w", err)
	}
	rows, err := l.query.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []spine.Event
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events rows: %w", err)
	}
	return events, nil
}

func scanEvent(row pgx.Row) (spine.Event, error) {
	var id pgtype.UUID
	var organizationID pgtype.UUID
	var projectID pgtype.UUID
	var repoBindingID pgtype.UUID
	var entityID pgtype.UUID
	var event spine.Event
	if err := row.Scan(
		&id,
		&event.Type,
		&organizationID,
		&projectID,
		&repoBindingID,
		&event.EntityType,
		&entityID,
		&event.Timestamp,
		&event.Payload,
	); err != nil {
		return spine.Event{}, err
	}
	event.ID = spine.EventID(uuidString(id))
	event.OrganizationID = spine.OrganizationID(uuidString(organizationID))
	event.ProjectID = spine.ProjectID(uuidString(projectID))
	event.RepoBindingID = spine.RepoBindingID(uuidString(repoBindingID))
	event.EntityID = uuidString(entityID)
	event.Timestamp = event.Timestamp.UTC()
	if event.Payload != nil {
		event.Payload = append([]byte(nil), event.Payload...)
	}
	return event, nil
}

func (l *PostgresEventLog) execSQL(ctx context.Context, op string, sqlizer squirrel.Sqlizer) error {
	if l.exec == nil {
		return fmt.Errorf("%s executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("%s SQL: %w", op, err)
	}
	if _, err := l.exec.Exec(ctx, sqlText, args...); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
