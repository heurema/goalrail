package lineage

import (
	"context"
	"fmt"

	"github.com/heurema/goalrail/internal/domain"
	projectstate "github.com/heurema/goalrail/internal/project"
)

type AttachOptions struct {
	Repository string
	Event      domain.LineageEvent
	Replica    *Replica
}

func Attach(ctx context.Context, options AttachOptions) (AttachReceipt, error) {
	inspection, err := projectstate.Inspect(ctx, options.Repository)
	if err != nil {
		return AttachReceipt{}, err
	}
	if inspection.State != projectstate.ClaimManaged {
		return AttachReceipt{}, fmt.Errorf("lineage attach requires one valid managed project: %s", inspection.Detail)
	}
	artifacts, err := projectstate.InspectGoverningArtifacts(inspection)
	if err != nil {
		return AttachReceipt{}, err
	}
	if !artifacts.PolicyReady() {
		return AttachReceipt{}, fmt.Errorf("lineage attach requires the current committed policy: %s", artifacts.Policy.Detail)
	}
	store, err := NewStore(inspection.WorktreeRoot)
	if err != nil {
		return AttachReceipt{}, err
	}
	unit, _, err := store.LoadWorkUnit(options.Event.WorkUnitID)
	if err != nil {
		return AttachReceipt{}, err
	}
	if unit.ProjectID != inspection.Declaration.ProjectID ||
		unit.DeclarationDigest != inspection.DeclarationDigest ||
		unit.PolicyDigest != inspection.Declaration.Policy.Digest {
		return AttachReceipt{}, fmt.Errorf("work-unit authority does not match the current managed project")
	}
	if err := inspection.Revalidate(); err != nil {
		return AttachReceipt{}, fmt.Errorf("managed project changed during lineage attach: %w", err)
	}
	return store.Attach(options.Event, options.Replica)
}
