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

type PostgresContractSeedStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresContractSeedStore(pool *pgxpool.Pool) *PostgresContractSeedStore {
	db := newPostgresDB(pool)
	return NewPostgresContractSeedStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresContractSeedStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresContractSeedStore {
	return &PostgresContractSeedStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresContractSeedStore) Create(ctx context.Context, created spine.ContractSeed) error {
	id, err := uuidValue(created.ID, "contract seed id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(created.OrganizationID, "contract seed organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(created.ProjectID, "contract seed project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(created.RepoBindingID, "contract seed repo binding id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(created.ContractID, "contract seed contract id")
	if err != nil {
		return err
	}
	goalID, err := uuidValue(created.GoalID, "contract seed goal id")
	if err != nil {
		return err
	}
	intentOwner, err := json.Marshal(created.IntentOwner)
	if err != nil {
		return fmt.Errorf("marshal contract seed intent owner: %w", err)
	}
	sourceRefsValue := created.SourceRefs
	if sourceRefsValue == nil {
		sourceRefsValue = []spine.SourceRef{}
	}
	sourceRefs, err := json.Marshal(sourceRefsValue)
	if err != nil {
		return fmt.Errorf("marshal contract seed source refs: %w", err)
	}

	createdAt := created.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	stmt := s.psql.
		Insert("contract_seeds").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"contract_id",
			"goal_id",
			"title",
			"intent_summary",
			"intent_owner",
			"scope_hint",
			"acceptance_hint",
			"source_refs",
			"state",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			repoBindingID,
			contractID,
			goalID,
			created.Title,
			created.IntentSummary,
			intentOwner,
			created.ScopeHint,
			created.AcceptanceHint,
			sourceRefs,
			created.State,
			createdAt,
			createdAt,
		)

	if err := s.execSQL(ctx, "create contract seed", stmt); err != nil {
		if uniqueViolationConstraint(err) == "contract_seeds_goal_id_unique" {
			return ErrContractSeedAlreadySeeded
		}
		if isUniqueViolation(err) {
			return ErrContractSeedAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresContractSeedStore) Get(ctx context.Context, id spine.ContractSeedID) (spine.ContractSeed, bool, error) {
	seedID, err := uuidValue(id, "contract seed id")
	if err != nil {
		return spine.ContractSeed{}, false, err
	}
	return s.getOne(ctx, "get contract seed", squirrel.Eq{"id": seedID})
}

func (s *PostgresContractSeedStore) GetByGoalID(ctx context.Context, id spine.GoalID) (spine.ContractSeed, bool, error) {
	goalID, err := uuidValue(id, "contract seed goal id")
	if err != nil {
		return spine.ContractSeed{}, false, err
	}
	return s.getOne(ctx, "get contract seed by goal id", squirrel.Eq{"goal_id": goalID})
}

func (s *PostgresContractSeedStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.ContractSeed, bool, error) {
	stmt := s.psql.
		Select(contractSeedColumns()...).
		From("contract_seeds").
		Where(where)

	return s.queryContractSeed(ctx, op, stmt)
}

func (s *PostgresContractSeedStore) queryContractSeed(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.ContractSeed, bool, error) {
	if s.query == nil {
		return spine.ContractSeed{}, false, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return spine.ContractSeed{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	seed, err := scanContractSeed(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ContractSeed{}, false, nil
		}
		return spine.ContractSeed{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return seed, true, nil
}

func scanContractSeed(row pgx.Row) (spine.ContractSeed, error) {
	var seed spine.ContractSeed
	var id string
	var organizationID string
	var projectID string
	var repoBindingID string
	var contractID string
	var goalID string
	var intentOwner []byte
	var sourceRefs []byte
	var state string
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&repoBindingID,
		&contractID,
		&goalID,
		&seed.Title,
		&seed.IntentSummary,
		&intentOwner,
		&seed.ScopeHint,
		&seed.AcceptanceHint,
		&sourceRefs,
		&state,
		&seed.CreatedAt,
	); err != nil {
		return spine.ContractSeed{}, err
	}
	seed.ID = spine.ContractSeedID(id)
	seed.OrganizationID = spine.OrganizationID(organizationID)
	seed.ProjectID = spine.ProjectID(projectID)
	seed.RepoBindingID = spine.RepoBindingID(repoBindingID)
	seed.ContractID = spine.ContractID(contractID)
	seed.GoalID = spine.GoalID(goalID)
	seed.State = spine.ContractSeedState(state)
	if err := json.Unmarshal(intentOwner, &seed.IntentOwner); err != nil {
		return spine.ContractSeed{}, fmt.Errorf("unmarshal contract seed intent owner: %w", err)
	}
	if err := json.Unmarshal(sourceRefs, &seed.SourceRefs); err != nil {
		return spine.ContractSeed{}, fmt.Errorf("unmarshal contract seed source refs: %w", err)
	}
	seed.CreatedAt = seed.CreatedAt.UTC()
	return seed, nil
}

func contractSeedColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"repo_binding_id",
		"contract_id",
		"goal_id",
		"title",
		"intent_summary",
		"intent_owner",
		"scope_hint",
		"acceptance_hint",
		"source_refs",
		"state",
		"created_at",
	}
}

func (s *PostgresContractSeedStore) execSQL(ctx context.Context, op string, sqlizer squirrel.Sqlizer) error {
	if s.exec == nil {
		return fmt.Errorf("%s executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("%s SQL: %w", op, err)
	}
	if _, err := s.exec.Exec(ctx, sqlText, args...); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
