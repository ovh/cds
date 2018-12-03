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
	"github.com/ovh/cds/sdk/log"
)

//Import is able to create a new workflow and all its components
func Import(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, u *sdk.User, force bool, msgChan chan<- sdk.Message, dryRun bool) error {
	ctx, end := observability.Span(ctx, "workflow.Import")
	defer end()

	w.ProjectKey = proj.Key
	w.ProjectID = proj.ID

	wTemplate := w.Template

	// Default value of history length is 20
	if w.HistoryLength == 0 {
		w.HistoryLength = 20
	}

	doUpdate, errE := Exists(db, proj.Key, w.Name)
	if errE != nil {
		return sdk.WrapError(errE, "Import> Cannot check if workflow exists")
	}

	//Manage default payload
	var err error
	if w.Root.Context == nil {
		w.Root.Context = &sdk.WorkflowNodeContext{}
	}
	if w.WorkflowData.Node.Context == nil {
		w.WorkflowData.Node.Context = &sdk.NodeContext{}
	}

	// TODO compute on WD.Node
	if w.Root.Context.DefaultPayload, err = DefaultPayload(ctx, db, store, proj, u, w); err != nil {
		log.Warning("workflow.Import> Cannot set default payload : %v", err)
	}
	w.WorkflowData.Node.Context.DefaultPayload = w.Root.Context.DefaultPayload

	if !doUpdate {
		if err := Insert(db, store, w, proj, u); err != nil {
			return sdk.WrapError(err, "Unable to insert workflow")
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedInserted, w.Name)
		}

		// HookRegistration after workflow.Update.  It needs hooks to be created on DB
		if errHr := HookRegistration(ctx, db, store, nil, *w, proj); errHr != nil {
			return sdk.WrapError(errHr, "Cannot register hook")
		}

		// set the workflow id on template instance if exist
		if err := setTemplateData(db, proj, w, u, wTemplate); err != nil {
			return err
		}

		return importWorkflowGroups(db, w)
	}

	if !force {
		return sdk.NewError(sdk.ErrConflict, fmt.Errorf("Workflow exists"))
	}

	oldW, errO := Load(ctx, db, store, proj, w.Name, u, LoadOptions{WithIcon: true})
	if errO != nil {
		return sdk.WrapError(errO, "Unable to load old workflow")
	}

	w.ID = oldW.ID
	if err := Update(ctx, db, store, w, oldW, proj, u); err != nil {
		return sdk.WrapError(err, "Unable to update workflow")
	}

	if !dryRun {
		// HookRegistration after workflow.Update.  It needs hooks to be created on DB
		if errHr := HookRegistration(ctx, db, store, oldW, *w, proj); errHr != nil {
			return sdk.WrapError(errHr, "Cannot register hook")
		}
	}

	if err := importWorkflowGroups(db, w); err != nil {
		return err
	}

	// set the workflow id on template instance if exist
	if err := setTemplateData(db, proj, w, u, wTemplate); err != nil {
		return err
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedUpdated, w.Name)
	}
	return nil
}

func setTemplateData(db gorp.SqlExecutor, p *sdk.Project, w *sdk.Workflow, u *sdk.User, wt *sdk.WorkflowTemplate) error {
	// set the workflow id on template instance if exist
	if wt == nil {
		return nil
	}

	var group *sdk.Group
	for _, g := range u.Groups {
		if g.Name == wt.Group.Name {
			group = &g
			break
		}
	}
	if group == nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "Could not find given workflow template")
	}

	wt, err := workflowtemplate.GetBySlugAndGroupIDs(db, wt.Slug, []int64{group.ID})
	if err != nil {
		return err
	}
	if wt == nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "Could not find given workflow template")
	}

	wtis, err := workflowtemplate.GetInstancesByTemplateIDAndProjectIDs(db, wt.ID, []int64{p.ID})
	if err != nil {
		return err
	}
	var wTemplateInstance *sdk.WorkflowTemplateInstance
	for _, wti := range wtis {
		if wti.Request.WorkflowName == w.Name {
			wTemplateInstance = &wti
			break
		}
	}
	if wTemplateInstance == nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "Could not find a template instance for workflow %s", w.Name)
	}

	// remove existing relations between workflow and template
	if err := workflowtemplate.DeleteInstanceNotIDAndWorkflowID(db, wTemplateInstance.ID, w.ID); err != nil {
		return err
	}

	old := sdk.WorkflowTemplateInstance(*wTemplateInstance)

	// set the workflow id on target instance
	wTemplateInstance.WorkflowID = &w.ID
	if err := workflowtemplate.UpdateInstance(db, wTemplateInstance); err != nil {
		return err
	}

	event.PublishWorkflowTemplateInstanceUpdate(old, *wTemplateInstance, u)

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
		if err := upsertAllGroups(db, w, w.Groups); err != nil {
			return sdk.WrapError(err, "Unable to update workflow")
		}
	}
	return nil
}
