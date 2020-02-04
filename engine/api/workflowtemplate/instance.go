package workflowtemplate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

func SetTemplateData(ctx context.Context, db gorp.SqlExecutor, p *sdk.Project, w *sdk.Workflow, u sdk.Identifiable, wt *sdk.WorkflowTemplate) error {
	// set the workflow id on template instance if exist
	if wt == nil {
		return nil
	}

	// check that group exists
	grp, err := group.LoadByName(ctx, db, wt.Group.Name)
	if err != nil {
		return err
	}

	wt, err = LoadBySlugAndGroupID(ctx, db, wt.Slug, grp.ID)
	if err != nil {
		return err
	}
	if wt == nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Could not find given workflow template")
	}

	wti, err := GetInstanceByWorkflowNameAndTemplateIDAndProjectID(db, w.Name, wt.ID, p.ID)
	if err != nil {
		return err
	}
	if wti == nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Could not find a template instance for workflow %s", w.Name)
	}

	// remove existing relations between workflow and template
	if err := DeleteInstanceNotIDAndWorkflowID(db, wti.ID, w.ID); err != nil {
		return err
	}

	old := sdk.WorkflowTemplateInstance(*wti)

	// set the workflow id on target instance
	wti.WorkflowID = &w.ID
	if err := UpdateInstance(db, wti); err != nil {
		return err
	}

	event.PublishWorkflowTemplateInstanceUpdate(ctx, old, *wti, u)

	return nil
}
