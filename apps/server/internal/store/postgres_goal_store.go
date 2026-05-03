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

type PostgresGoalStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresGoalStore(pool *pgxpool.Pool) *PostgresGoalStore {
	db := newPostgresDB(pool)
	return NewPostgresGoalStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresGoalStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresGoalStore {
	return &PostgresGoalStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresGoalStore) Create(ctx context.Context, created spine.Goal) error {
	id, err := uuidValue(created.ID, "goal id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(created.OrganizationID, "goal organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(created.ProjectID, "goal project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(created.RepoBindingID, "goal repo binding id")
	if err != nil {
		return err
	}
	intakeID, err := uuidValue(created.IntakeID, "goal intake id")
	if err != nil {
		return err
	}
	sourceRefsValue := created.SourceRefs
	if sourceRefsValue == nil {
		sourceRefsValue = []spine.SourceRef{}
	}
	sourceRefs, err := json.Marshal(sourceRefsValue)
	if err != nil {
		return fmt.Errorf("marshal goal source refs: %w", err)
	}
	requestAuthor, err := json.Marshal(created.RequestAuthor)
	if err != nil {
		return fmt.Errorf("marshal goal request author: %w", err)
	}
	intentOwner, err := json.Marshal(created.IntentOwner)
	if err != nil {
		return fmt.Errorf("marshal goal intent owner: %w", err)
	}
	readinessReasonValues := created.LastReadinessReasonCodes
	if readinessReasonValues == nil {
		readinessReasonValues = []spine.GoalReadinessReasonCode{}
	}
	readinessReasons, err := json.Marshal(readinessReasonValues)
	if err != nil {
		return fmt.Errorf("marshal goal readiness reason codes: %w", err)
	}

	createdAt := created.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	stmt := s.psql.
		Insert("goals").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"intake_id",
			"title",
			"summary",
			"scope_hint",
			"acceptance_hint",
			"source_refs",
			"request_author",
			"intent_owner",
			"state",
			"last_readiness_reason_codes",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			repoBindingID,
			intakeID,
			created.Title,
			created.Summary,
			created.ScopeHint,
			created.AcceptanceHint,
			sourceRefs,
			requestAuthor,
			intentOwner,
			created.State,
			readinessReasons,
			createdAt,
			createdAt,
		)

	if err := execSQL(ctx, s.exec, "create goal", stmt); err != nil {
		if isUniqueViolation(err) {
			return ErrGoalAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresGoalStore) Get(ctx context.Context, id spine.GoalID) (spine.Goal, bool, error) {
	goalID, err := uuidValue(id, "goal id")
	if err != nil {
		return spine.Goal{}, false, err
	}
	return s.getOne(ctx, "get goal", squirrel.Eq{"id": goalID})
}

func (s *PostgresGoalStore) GetByIntakeID(ctx context.Context, id spine.IntakeID) (spine.Goal, bool, error) {
	intakeID, err := uuidValue(id, "goal intake id")
	if err != nil {
		return spine.Goal{}, false, err
	}
	return s.getOne(ctx, "get goal by intake id", squirrel.Eq{"intake_id": intakeID})
}

func (s *PostgresGoalStore) UpdateState(ctx context.Context, id spine.GoalID, state spine.GoalState) (spine.Goal, bool, error) {
	return s.UpdateReadiness(ctx, id, state, nil)
}

func (s *PostgresGoalStore) UpdateReadiness(ctx context.Context, id spine.GoalID, state spine.GoalState, reasonCodes []spine.GoalReadinessReasonCode) (spine.Goal, bool, error) {
	goalID, err := uuidValue(id, "goal id")
	if err != nil {
		return spine.Goal{}, false, err
	}
	readinessReasonValues := reasonCodes
	if readinessReasonValues == nil {
		readinessReasonValues = []spine.GoalReadinessReasonCode{}
	}
	readinessReasons, err := json.Marshal(readinessReasonValues)
	if err != nil {
		return spine.Goal{}, false, fmt.Errorf("marshal goal readiness reason codes: %w", err)
	}
	stmt := s.psql.
		Update("goals").
		Set("state", state).
		Set("last_readiness_reason_codes", readinessReasons).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": goalID}).
		Suffix(returningGoalColumns())

	return s.queryGoal(ctx, "update goal readiness", stmt)
}

func (s *PostgresGoalStore) UpdateHints(ctx context.Context, id spine.GoalID, update spine.GoalHintUpdate) (spine.Goal, bool, error) {
	goalID, err := uuidValue(id, "goal id")
	if err != nil {
		return spine.Goal{}, false, err
	}
	stmt := s.psql.
		Update("goals").
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": goalID})
	if update.Summary != nil {
		stmt = stmt.Set("summary", *update.Summary)
	}
	if update.ScopeHint != nil {
		stmt = stmt.Set("scope_hint", *update.ScopeHint)
	}
	if update.AcceptanceHint != nil {
		stmt = stmt.Set("acceptance_hint", *update.AcceptanceHint)
	}
	if update.IntentOwner != nil {
		intentOwner, err := json.Marshal(*update.IntentOwner)
		if err != nil {
			return spine.Goal{}, false, fmt.Errorf("marshal goal intent owner: %w", err)
		}
		stmt = stmt.Set("intent_owner", intentOwner)
	}
	stmt = stmt.Suffix(returningGoalColumns())

	return s.queryGoal(ctx, "update goal hints", stmt)
}

func (s *PostgresGoalStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.Goal, bool, error) {
	stmt := s.psql.
		Select(goalColumns()...).
		From("goals").
		Where(where)

	return s.queryGoal(ctx, op, stmt)
}

func (s *PostgresGoalStore) queryGoal(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.Goal, bool, error) {
	row, err := queryRow(ctx, s.query, op, sqlizer)
	if err != nil {
		return spine.Goal{}, false, err
	}
	goal, err := scanGoal(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Goal{}, false, nil
		}
		return spine.Goal{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return goal, true, nil
}

func scanGoal(row pgx.Row) (spine.Goal, error) {
	var goal spine.Goal
	var id string
	var organizationID string
	var projectID string
	var repoBindingID string
	var intakeID string
	var sourceRefs []byte
	var requestAuthor []byte
	var intentOwner []byte
	var readinessReasons []byte
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&repoBindingID,
		&intakeID,
		&goal.Title,
		&goal.Summary,
		&goal.ScopeHint,
		&goal.AcceptanceHint,
		&sourceRefs,
		&requestAuthor,
		&intentOwner,
		&goal.State,
		&readinessReasons,
		&goal.CreatedAt,
	); err != nil {
		return spine.Goal{}, err
	}
	goal.ID = spine.GoalID(id)
	goal.OrganizationID = spine.OrganizationID(organizationID)
	goal.ProjectID = spine.ProjectID(projectID)
	goal.RepoBindingID = spine.RepoBindingID(repoBindingID)
	goal.IntakeID = spine.IntakeID(intakeID)
	if err := json.Unmarshal(sourceRefs, &goal.SourceRefs); err != nil {
		return spine.Goal{}, fmt.Errorf("unmarshal goal source refs: %w", err)
	}
	if err := json.Unmarshal(requestAuthor, &goal.RequestAuthor); err != nil {
		return spine.Goal{}, fmt.Errorf("unmarshal goal request author: %w", err)
	}
	if err := json.Unmarshal(intentOwner, &goal.IntentOwner); err != nil {
		return spine.Goal{}, fmt.Errorf("unmarshal goal intent owner: %w", err)
	}
	if err := json.Unmarshal(readinessReasons, &goal.LastReadinessReasonCodes); err != nil {
		return spine.Goal{}, fmt.Errorf("unmarshal goal readiness reason codes: %w", err)
	}
	goal.CreatedAt = goal.CreatedAt.UTC()
	return goal, nil
}

func goalColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"repo_binding_id",
		"intake_id",
		"title",
		"summary",
		"scope_hint",
		"acceptance_hint",
		"source_refs",
		"request_author",
		"intent_owner",
		"state",
		"last_readiness_reason_codes",
		"created_at",
	}
}

func returningGoalColumns() string {
	return "RETURNING id, organization_id, project_id, repo_binding_id, intake_id, title, summary, scope_hint, acceptance_hint, source_refs, request_author, intent_owner, state, last_readiness_reason_codes, created_at"
}
