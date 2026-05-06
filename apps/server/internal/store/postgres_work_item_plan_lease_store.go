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
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

type PostgresWorkItemPlanLeaseStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresWorkItemPlanLeaseStore(pool *pgxpool.Pool) *PostgresWorkItemPlanLeaseStore {
	db := newPostgresDB(pool)
	return NewPostgresWorkItemPlanLeaseStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresWorkItemPlanLeaseStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresWorkItemPlanLeaseStore {
	return &PostgresWorkItemPlanLeaseStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresWorkItemPlanLeaseStore) AcquireNextLease(ctx context.Context, input workitemplan.LeaseAcquireInput) (spine.WorkItemPlanLease, bool, error) {
	if s.exec == nil || s.query == nil {
		return spine.WorkItemPlanLease{}, false, fmt.Errorf("work item plan lease store executor is nil")
	}
	now := input.CreatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	row := s.query.QueryRow(ctx, `
SELECT id, organization_id, project_id, contract_id, approved_contract_id, repo_binding_id, state, requested_by, current_lease_id, leased_by, lease_expires_at, created_at, updated_at
FROM work_item_plans
WHERE state = $1 OR (state = $2 AND lease_expires_at <= $3)
ORDER BY created_at ASC, id ASC
LIMIT 1
FOR UPDATE SKIP LOCKED
`, spine.WorkItemPlanStateQueued, spine.WorkItemPlanStateLeased, now)
	plan, err := scanWorkItemPlan(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.WorkItemPlanLease{}, false, nil
		}
		return spine.WorkItemPlanLease{}, false, fmt.Errorf("select next work item plan lease candidate: %w", err)
	}

	if plan.State == spine.WorkItemPlanStateLeased && plan.CurrentLeaseID != nil {
		previousID, err := uuidValue(*plan.CurrentLeaseID, "current work item plan lease id")
		if err != nil {
			return spine.WorkItemPlanLease{}, false, err
		}
		if _, err := s.exec.Exec(ctx, `
UPDATE work_item_plan_leases
SET state = $1, updated_at = $2
WHERE id = $3 AND state = $4
`, spine.WorkItemPlanLeaseStateExpired, now, previousID, spine.WorkItemPlanLeaseStateActive); err != nil {
			return spine.WorkItemPlanLease{}, false, fmt.Errorf("expire previous work item plan lease: %w", err)
		}
	}

	lease := spine.WorkItemPlanLease{
		ID:                 input.ID,
		PlanID:             plan.ID,
		ContractID:         plan.ContractID,
		ApprovedContractID: plan.ApprovedContractID,
		RepoBindingID:      plan.RepoBindingID,
		LeasedBy:           input.LeasedBy,
		State:              spine.WorkItemPlanLeaseStateActive,
		LeaseTokenHash:     input.LeaseTokenHash,
		ExpiresAt:          input.ExpiresAt.UTC(),
		CreatedAt:          now,
		UpdatedAt:          input.UpdatedAt.UTC(),
	}
	if lease.UpdatedAt.IsZero() {
		lease.UpdatedAt = now
	}
	if err := s.insertLease(ctx, lease); err != nil {
		return spine.WorkItemPlanLease{}, false, err
	}
	if err := s.markPlanLeased(ctx, plan.ID, lease); err != nil {
		return spine.WorkItemPlanLease{}, false, err
	}
	return lease, true, nil
}

func (s *PostgresWorkItemPlanLeaseStore) Get(ctx context.Context, id spine.WorkItemPlanLeaseID) (spine.WorkItemPlanLease, bool, error) {
	leaseID, err := uuidValue(id, "work item plan lease id")
	if err != nil {
		return spine.WorkItemPlanLease{}, false, err
	}
	stmt := s.psql.Select(workItemPlanLeaseColumns()...).From("work_item_plan_leases").Where(squirrel.Eq{"id": leaseID})
	row, err := queryRow(ctx, s.query, "get work item plan lease", stmt)
	if err != nil {
		return spine.WorkItemPlanLease{}, false, err
	}
	lease, err := scanWorkItemPlanLease(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.WorkItemPlanLease{}, false, nil
		}
		return spine.WorkItemPlanLease{}, false, fmt.Errorf("get work item plan lease: %w", err)
	}
	return lease, true, nil
}

func (s *PostgresWorkItemPlanLeaseStore) Renew(ctx context.Context, id spine.WorkItemPlanLeaseID, tokenHash string, expiresAt time.Time, updatedAt time.Time) (spine.WorkItemPlanLease, bool, error) {
	if s.query == nil {
		return spine.WorkItemPlanLease{}, false, fmt.Errorf("renew work item plan lease query executor is nil")
	}
	leaseID, err := uuidValue(id, "work item plan lease id")
	if err != nil {
		return spine.WorkItemPlanLease{}, false, err
	}
	row := s.query.QueryRow(ctx, `
WITH updated_plan AS (
	UPDATE work_item_plans AS p
	SET lease_expires_at = $3, updated_at = $4
	FROM work_item_plan_leases AS l
	WHERE p.id = l.plan_id
		AND p.current_lease_id = l.id
		AND p.state = $5
		AND l.id = $1
		AND l.lease_token_hash = $2
		AND l.state = $6
		AND l.expires_at > $4
	RETURNING l.id
),
renewed_lease AS (
	UPDATE work_item_plan_leases AS l
	SET expires_at = $3, updated_at = $4
	FROM updated_plan AS p
	WHERE l.id = p.id
	RETURNING l.id, l.plan_id, l.contract_id, l.approved_contract_id, l.repo_binding_id, l.leased_by, l.state, l.lease_token_hash, l.expires_at, l.created_at, l.updated_at
)
SELECT id, plan_id, contract_id, approved_contract_id, repo_binding_id, leased_by, state, lease_token_hash, expires_at, created_at, updated_at
FROM renewed_lease
`, leaseID, tokenHash, expiresAt.UTC(), updatedAt.UTC(), spine.WorkItemPlanStateLeased, spine.WorkItemPlanLeaseStateActive)
	lease, err := scanWorkItemPlanLease(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.WorkItemPlanLease{}, false, nil
		}
		return spine.WorkItemPlanLease{}, false, fmt.Errorf("renew work item plan lease: %w", err)
	}
	return lease, true, nil
}

func (s *PostgresWorkItemPlanLeaseStore) MarkCompleted(ctx context.Context, id spine.WorkItemPlanLeaseID, tokenHash string, completedAt time.Time) (bool, error) {
	if s.exec == nil {
		return false, fmt.Errorf("complete work item plan lease executor is nil")
	}
	leaseID, err := uuidValue(id, "work item plan lease id")
	if err != nil {
		return false, err
	}
	stmt := s.psql.
		Update("work_item_plan_leases").
		Set("state", spine.WorkItemPlanLeaseStateCompleted).
		Set("updated_at", completedAt.UTC()).
		Where(squirrel.Eq{"id": leaseID, "lease_token_hash": tokenHash, "state": spine.WorkItemPlanLeaseStateActive}).
		Where(squirrel.Gt{"expires_at": completedAt.UTC()})
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("complete work item plan lease SQL: %w", err)
	}
	result, err := s.exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return false, fmt.Errorf("complete work item plan lease: %w", err)
	}
	return result.RowsAffected() > 0, nil
}

func (s *PostgresWorkItemPlanLeaseStore) insertLease(ctx context.Context, lease spine.WorkItemPlanLease) error {
	id, err := uuidValue(lease.ID, "work item plan lease id")
	if err != nil {
		return err
	}
	planID, err := uuidValue(lease.PlanID, "work item plan lease plan id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(lease.ContractID, "work item plan lease contract id")
	if err != nil {
		return err
	}
	approvedContractID, err := uuidValue(lease.ApprovedContractID, "work item plan lease approved contract id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(lease.RepoBindingID, "work item plan lease repo binding id")
	if err != nil {
		return err
	}
	leasedBy, err := json.Marshal(lease.LeasedBy)
	if err != nil {
		return fmt.Errorf("marshal work item plan lease leased_by: %w", err)
	}
	stmt := s.psql.
		Insert("work_item_plan_leases").
		Columns("id", "plan_id", "contract_id", "approved_contract_id", "repo_binding_id", "leased_by", "state", "lease_token_hash", "expires_at", "created_at", "updated_at").
		Values(id, planID, contractID, approvedContractID, repoBindingID, leasedBy, lease.State, lease.LeaseTokenHash, lease.ExpiresAt.UTC(), lease.CreatedAt.UTC(), lease.UpdatedAt.UTC())
	return execSQL(ctx, s.exec, "create work item plan lease", stmt)
}

func (s *PostgresWorkItemPlanLeaseStore) markPlanLeased(ctx context.Context, planID spine.WorkItemPlanID, lease spine.WorkItemPlanLease) error {
	parsedPlanID, err := uuidValue(planID, "work item plan id")
	if err != nil {
		return err
	}
	leaseID, err := uuidValue(lease.ID, "work item plan current lease id")
	if err != nil {
		return err
	}
	leasedBy, err := json.Marshal(lease.LeasedBy)
	if err != nil {
		return fmt.Errorf("marshal work item plan leased_by: %w", err)
	}
	stmt := s.psql.
		Update("work_item_plans").
		Set("state", spine.WorkItemPlanStateLeased).
		Set("current_lease_id", leaseID).
		Set("leased_by", leasedBy).
		Set("lease_expires_at", lease.ExpiresAt.UTC()).
		Set("updated_at", lease.UpdatedAt.UTC()).
		Where(squirrel.Eq{"id": parsedPlanID})
	return execUpdate(ctx, s.exec, "mark work item plan leased", ErrWorkItemPlanNotFound, stmt)
}

func (s *PostgresWorkItemPlanLeaseStore) updatePlanLeaseExpiry(ctx context.Context, planID spine.WorkItemPlanID, leaseID spine.WorkItemPlanLeaseID, expiresAt time.Time, updatedAt time.Time) error {
	parsedPlanID, err := uuidValue(planID, "work item plan id")
	if err != nil {
		return err
	}
	parsedLeaseID, err := uuidValue(leaseID, "work item plan current lease id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Update("work_item_plans").
		Set("lease_expires_at", expiresAt.UTC()).
		Set("updated_at", updatedAt.UTC()).
		Where(squirrel.Eq{"id": parsedPlanID, "current_lease_id": parsedLeaseID, "state": spine.WorkItemPlanStateLeased})
	return execUpdate(ctx, s.exec, "update work item plan lease expiry", ErrWorkItemPlanNotFound, stmt)
}

func scanWorkItemPlanLease(row pgx.Row) (spine.WorkItemPlanLease, error) {
	var lease spine.WorkItemPlanLease
	var id string
	var planID string
	var contractID string
	var approvedContractID string
	var repoBindingID string
	var leasedBy []byte
	var state string
	if err := row.Scan(
		&id,
		&planID,
		&contractID,
		&approvedContractID,
		&repoBindingID,
		&leasedBy,
		&state,
		&lease.LeaseTokenHash,
		&lease.ExpiresAt,
		&lease.CreatedAt,
		&lease.UpdatedAt,
	); err != nil {
		return spine.WorkItemPlanLease{}, err
	}
	lease.ID = spine.WorkItemPlanLeaseID(id)
	lease.PlanID = spine.WorkItemPlanID(planID)
	lease.ContractID = spine.ContractID(contractID)
	lease.ApprovedContractID = spine.ApprovedContractID(approvedContractID)
	lease.RepoBindingID = spine.RepoBindingID(repoBindingID)
	if err := json.Unmarshal(leasedBy, &lease.LeasedBy); err != nil {
		return spine.WorkItemPlanLease{}, fmt.Errorf("unmarshal work item plan lease leased_by: %w", err)
	}
	lease.State = spine.WorkItemPlanLeaseState(state)
	lease.ExpiresAt = lease.ExpiresAt.UTC()
	lease.CreatedAt = lease.CreatedAt.UTC()
	lease.UpdatedAt = lease.UpdatedAt.UTC()
	return lease, nil
}

func workItemPlanLeaseColumns() []string {
	return []string{
		"id",
		"plan_id",
		"contract_id",
		"approved_contract_id",
		"repo_binding_id",
		"leased_by",
		"state",
		"lease_token_hash",
		"expires_at",
		"created_at",
		"updated_at",
	}
}

func joinSQLColumns(columns []string) string {
	out := ""
	for i, column := range columns {
		if i > 0 {
			out += ", "
		}
		out += column
	}
	return out
}
