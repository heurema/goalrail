package seed

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

const (
	DevUserID                   spine.UserID                   = "usr_dev_owner"
	DevOrganizationID           spine.OrganizationID           = "org_dev_default"
	DevOrganizationMembershipID spine.OrganizationMembershipID = "omem_dev_owner"
	DevProjectID                spine.ProjectID                = "prj_dev_default"
	DevRepoBindingID            spine.RepoBindingID            = "rpb_dev_default"
)

type ProjectContextStore interface {
	UpsertUser(ctx context.Context, user spine.User) error
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

	org := spine.Organization{
		ID:          DevOrganizationID,
		Slug:        "dev-default",
		DisplayName: "Goalrail Dev Organization",
		State:       spine.EntityStateActive,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
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
		ID:             DevProjectID,
		OrganizationID: DevOrganizationID,
		Slug:           "dev-default",
		DisplayName:    "Goalrail Dev Project",
		State:          spine.EntityStateActive,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	if err := projectContext.UpsertProject(ctx, project); err != nil {
		return err
	}

	binding := spine.RepoBinding{
		ID:                 DevRepoBindingID,
		OrganizationID:     DevOrganizationID,
		ProjectID:          DevProjectID,
		Provider:           "custom_git",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "https://example.invalid/heurema/goalrail.git",
		DefaultBranch:      "main",
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
