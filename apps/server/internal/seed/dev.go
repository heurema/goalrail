package seed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

const (
	DevUserID                   spine.UserID                   = "018f0000-0000-7000-8000-000000000001"
	DevInstallationID           spine.InstallationID           = "018f0000-0000-7000-8000-000000000006"
	DevOrganizationID           spine.OrganizationID           = "018f0000-0000-7000-8000-000000000002"
	DevProjectID                spine.ProjectID                = "018f0000-0000-7000-8000-000000000003"
	DevRepoBindingID            spine.RepoBindingID            = "018f0000-0000-7000-8000-000000000004"
	DevOrganizationMembershipID spine.OrganizationMembershipID = "018f0000-0000-7000-8000-000000000005"
)

type ProjectContextStore interface {
	UpsertUser(ctx context.Context, user spine.User) error
	UpsertInstallation(ctx context.Context, installation spine.Installation) error
	UpsertOrganization(ctx context.Context, org spine.Organization) error
	UpsertOrganizationMembership(ctx context.Context, membership spine.OrganizationMembership) error
	UpsertProject(ctx context.Context, project spine.Project) error
	UpsertRepoBinding(ctx context.Context, binding spine.RepoBinding) error
}

func RunDev(ctx context.Context, projectContext ProjectContextStore, now time.Time) error {
	createdAt := now.UTC()

	user := spine.User{
		ID:          DevUserID,
		DisplayName: "Goalrail Dev Owner",
		Email:       "dev-owner@goalrail.local",
		State:       spine.EntityStateActive,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	if err := projectContext.UpsertUser(ctx, user); err != nil {
		return err
	}

	installation := spine.Installation{
		ID:            DevInstallationID,
		Mode:          spine.InstallationModeSelfHosted,
		PublicBaseURL: "http://localhost:8080",
		State:         spine.EntityStateActive,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
	}
	if err := projectContext.UpsertInstallation(ctx, installation); err != nil {
		return err
	}

	org := spine.Organization{
		ID:             DevOrganizationID,
		InstallationID: DevInstallationID,
		Slug:           "dev-default",
		DisplayName:    "Goalrail Dev Organization",
		State:          spine.EntityStateActive,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	if err := projectContext.UpsertOrganization(ctx, org); err != nil {
		return err
	}

	membership := spine.OrganizationMembership{
		ID:             DevOrganizationMembershipID,
		OrganizationID: DevOrganizationID,
		UserID:         DevUserID,
		Role:           spine.OrganizationMembershipRoleOwner,
		State:          spine.EntityStateActive,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	if err := projectContext.UpsertOrganizationMembership(ctx, membership); err != nil {
		return err
	}

	project := spine.Project{
		ID:              DevProjectID,
		OrganizationID:  DevOrganizationID,
		CreatedByUserID: DevUserID,
		Slug:            "dev-default",
		DisplayName:     "Goalrail Dev Project",
		State:           spine.EntityStateActive,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}
	if err := projectContext.UpsertProject(ctx, project); err != nil {
		return err
	}

	binding := spine.RepoBinding{
		ID:                 DevRepoBindingID,
		OrganizationID:     DevOrganizationID,
		ProjectID:          DevProjectID,
		CreatedByUserID:    DevUserID,
		Provider:           "custom_git",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "https://example.invalid/heurema/goalrail.git",
		DefaultBranch:      "main",
		WorkflowBaseBranch: "main",
		PathScope:          ".",
		AccessMode:         spine.RepoBindingAccessModeMetadataOnly,
		State:              spine.EntityStateActive,
		CreatedAt:          createdAt,
		UpdatedAt:          createdAt,
	}
	return projectContext.UpsertRepoBinding(ctx, binding)
}

func RunDevWithPool(ctx context.Context, pool *pgxpool.Pool, now time.Time) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := RunDev(ctx, store.NewProjectContextStoreWithExecutor(tx), now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
