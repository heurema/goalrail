package store

import (
	"context"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresRunnerCapabilityReportStore struct {
	exec postgresExecer
	psql squirrel.StatementBuilderType
}

func NewPostgresRunnerCapabilityReportStore(pool *pgxpool.Pool) *PostgresRunnerCapabilityReportStore {
	return NewPostgresRunnerCapabilityReportStoreWithExecutor(newPostgresDB(pool))
}

func NewPostgresRunnerCapabilityReportStoreWithExecutor(exec postgresExecer) *PostgresRunnerCapabilityReportStore {
	return &PostgresRunnerCapabilityReportStore{
		exec: exec,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresRunnerCapabilityReportStore) Create(ctx context.Context, report spine.RunnerCapabilityReport) error {
	id, err := uuidValue(report.ID, "runner capability report id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(report.OrganizationID, "runner capability report organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(report.ProjectID, "runner capability report project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(report.RepoBindingID, "runner capability report repo binding id")
	if err != nil {
		return err
	}
	reportedAt := report.ReportedAt.UTC()
	if reportedAt.IsZero() {
		reportedAt = time.Now().UTC()
	}
	createdAt := report.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = reportedAt
	}
	updatedAt := report.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	stmt := s.psql.
		Insert("runner_capability_reports").
		Columns(
			"id",
			"runner_id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"network_isolation_declared",
			"workspace_write_isolation_declared",
			"process_tree_control_declared",
			"stdout_stderr_policy_declared",
			"artifact_policy_declared",
			"trust_state",
			"reported_at",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			report.RunnerID,
			orgID,
			projectID,
			repoBindingID,
			report.NetworkIsolationDeclared,
			report.WorkspaceWriteIsolationDeclared,
			report.ProcessTreeControlDeclared,
			report.StdoutStderrPolicyDeclared,
			report.ArtifactPolicyDeclared,
			report.TrustState,
			reportedAt,
			createdAt,
			updatedAt,
		)
	if err := execSQL(ctx, s.exec, "create runner capability report", stmt); err != nil {
		if isUniqueViolation(err) {
			return ErrRunnerCapabilityReportAlreadyExists
		}
		return err
	}
	return nil
}
