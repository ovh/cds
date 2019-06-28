package workflow

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
)

//Import is able to create a new workflow and all its components
func Import(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, oldW, w *sdk.Workflow, u *sdk.User, force bool, msgChan chan<- sdk.Message) error {
	ctx, end := observability.Span(ctx, "workflow.Import")
	defer end()

	w.ProjectKey = proj.Key
	w.ProjectID = proj.ID

	wTemplate := w.Template

	// Default value of history length is 20
	if w.HistoryLength == 0 {
		w.HistoryLength = 20
	}

	if w.WorkflowData.Node.Context == nil {
		w.WorkflowData.Node.Context = &sdk.NodeContext{}
	}

	// create the workflow if not exists
	if oldW == nil {
		if err := Insert(db, store, w, proj, u); err != nil {
			return sdk.WrapError(err, "Unable to insert workflow")
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedInserted, w.Name)
		}

		// set the workflow id on template instance if exist
		if err := setTemplateData(ctx, db, proj, w, u, wTemplate); err != nil {
			return err
		}

		return nil
	}

	if oldW.Icon != "" && w.Icon == "" {
		w.Icon = oldW.Icon
	}

	if !force {
		return sdk.NewError(sdk.ErrConflict, fmt.Errorf("Workflow exists"))
	}

	// Retrieve existing hook
	oldHooksByRef := oldW.WorkflowData.GetHooksMapRef()
	for i := range w.WorkflowData.Node.Hooks {
		h := &w.WorkflowData.Node.Hooks[i]
		if h.Ref != "" {
			if oldH, has := oldHooksByRef[h.Ref]; has {
				if len(h.Config) == 0 {
					h.Config = oldH.Config
				}
				h.UUID = oldH.UUID
				continue
			}
		}
	}
	w.ID = oldW.ID

	// HookRegistration after workflow.Update.  It needs hooks to be created on DB
	// Hook registration must only be done on default branch in case of workflow as-code
	// The derivation branch is set in workflow parser it is not comming from the default branch
	uptOptions := UpdateOptions{
		DisableHookManagement: w.DerivationBranch != "",
		OldWorkflow:           oldW,
	}

	if err := Update(ctx, db, store, w, proj, u, uptOptions); err != nil {
		return sdk.WrapError(err, "Unable to update workflow")
	}

	if err := importWorkflowGroups(db, w); err != nil {
		return err
	}

	// set the workflow id on template instance if exist
	if err := setTemplateData(ctx, db, proj, w, u, wTemplate); err != nil {
		return err
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedUpdated, w.Name)
	}
	return nil
}

func setTemplateData(ctx context.Context, db gorp.SqlExecutor, p *sdk.Project, w *sdk.Workflow, u *sdk.User, wt *sdk.WorkflowTemplate) error {
	// set the workflow id on template instance if exist
	if wt == nil {
		return nil
	}

	// check that group exists
	grp, err := group.LoadGroup(db, wt.Group.Name)
	if err != nil {
		return err
	}
	if err := group.CheckUserIsGroupMember(grp, u); err != nil {
		return err
	}

	wt, err = workflowtemplate.LoadBySlugAndGroupID(ctx, db, wt.Slug, grp.ID)
	if err != nil {
		return err
	}
	if wt == nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Could not find given workflow template")
	}

	wti, err := workflowtemplate.GetInstanceByWorkflowNameAndTemplateIDAndProjectID(db, w.Name, wt.ID, p.ID)
	if err != nil {
		return err
	}
	if wti == nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Could not find a template instance for workflow %s", w.Name)
	}

	// remove existing relations between workflow and template
	if err := workflowtemplate.DeleteInstanceNotIDAndWorkflowID(db, wti.ID, w.ID); err != nil {
		return err
	}

	old := sdk.WorkflowTemplateInstance(*wti)

	// set the workflow id on target instance
	wti.WorkflowID = &w.ID
	if err := workflowtemplate.UpdateInstance(db, wti); err != nil {
		return err
	}

	event.PublishWorkflowTemplateInstanceUpdate(old, *wti, u)

	return nil
}

func importWorkflowGroups(db gorp.SqlExecutor, w *sdk.Workflow) error {
	if len(w.Groups) > 0 {
		for i := range w.Groups {
			g, err := group.LoadGroup(db, w.Groups[i].Group.Name)
			if err != nil {
				return sdk.WrapError(err, "Unable to load group %s", w.Groups[i].Group.Name)
			}
			w.Groups[i].Group = *g
		}
		if err := group.UpsertAllWorkflowGroups(db, w, w.Groups); err != nil {
			return sdk.WrapError(err, "Unable to update workflow")
		}
	}
	return nil
}
