package workflow

import (
	"context"

	"github.com/rockbears/log"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

//Import is able to create a new workflow and all its components
func Import(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, oldW, w *sdk.Workflow, u sdk.Identifiable, force bool, msgChan chan<- sdk.Message) error {
	ctx, end := telemetry.Span(ctx, "workflow.Import")
	defer end()

	w.ProjectKey = proj.Key
	w.ProjectID = proj.ID

	// Default value of history length is 20
	if w.HistoryLength == 0 {
		w.HistoryLength = 20
	}

	if w.WorkflowData.Node.Context == nil {
		w.WorkflowData.Node.Context = &sdk.NodeContext{}
	}

	// create the workflow if not exists
	if oldW == nil {
		if err := Insert(ctx, db, store, proj, w); err != nil {
			return sdk.WrapError(err, "Unable to insert workflow")
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedInserted, w.Name)
		}
		return nil
	}

	if oldW.Icon != "" && w.Icon == "" {
		w.Icon = oldW.Icon
	}

	if !force {
		return sdk.NewErrorFrom(sdk.ErrAlreadyExist, "workflow exists")
	}

	if force && oldW != nil && oldW.FromRepository != "" && w.FromRepository == "" {
		if err := detachResourceFromRepository(db, proj.ID, oldW, msgChan); err != nil {
			return err
		}
		msgChan <- sdk.NewMessage(sdk.MsgWorkflowDetached, oldW.Name, oldW.FromRepository)
		log.Debug(ctx, "workflow.Import>> Force import workflow %s in project %s without fromRepository", oldW.Name, proj.Key)
	}

	// Retrieve existing hook
	oldHooksByRef := oldW.WorkflowData.GetHooksMapRef()
	for i := range w.WorkflowData.Node.Hooks {
		h := &w.WorkflowData.Node.Hooks[i]
		if h.UUID == "" {
			if oldH, has := oldHooksByRef[h.Ref()]; has {
				if len(h.Config) == 0 {
					h.Config = oldH.Config.Clone()
					// the oldW can have a different name than the workflow to import
					//we have to rename the workflow name in the hook config retrieve from old workflow
					h.Config[sdk.HookConfigWorkflow] = sdk.WorkflowNodeHookConfigValue{
						Value:        w.Name,
						Configurable: false,
					}
				}
				h.UUID = oldH.UUID
				continue
			}
		}
	}
	w.ID = oldW.ID

	// HookRegistration after workflow.Update.  It needs hooks to be created on DB
	// Hook registration must only be done on default branch in case of workflow as-code
	// The derivation branch is set in workflow parser it is not coming from the default branch
	uptOptions := UpdateOptions{
		DisableHookManagement: w.DerivationBranch != "",
	}

	if err := importWorkflowGroups(db, w); err != nil {
		return err
	}

	if err := Update(ctx, db, store, proj, w, uptOptions); err != nil {
		return sdk.WrapError(err, "Unable to update workflow")
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedUpdated, w.Name)
	}
	return nil
}

func detachResourceFromRepository(db gorp.SqlExecutor, projectID int64, oldW *sdk.Workflow, msgChan chan<- sdk.Message) error {
	// delete ascode event if exists on this workflow
	if err := ascode.DeleteAsCodeEventByWorkflowID(db, oldW.ID); err != nil {
		return err
	}
	// reset fromRepository for all pipeline using it
	pips, err := pipeline.LoadAllNamesByFromRepository(db, projectID, oldW.FromRepository)
	if err != nil {
		return err
	}

	if err := pipeline.ResetFromRepository(db, projectID, oldW.FromRepository); err != nil {
		return sdk.WrapError(err, "could not reset fromRepository %s from pipelines", oldW.FromRepository)
	}

	for _, pip := range pips {
		msgChan <- sdk.NewMessage(sdk.MsgPipelineDetached, pip.Name, oldW.FromRepository)
	}

	// reset fromRepository for all app using it
	apps, err := application.LoadAllNamesByFromRepository(db, projectID, oldW.FromRepository)
	if err != nil {
		return err
	}

	if err := application.ResetFromRepository(db, projectID, oldW.FromRepository); err != nil {
		return sdk.WrapError(err, "could not reset fromRepository %s from applications", oldW.FromRepository)
	}

	for _, app := range apps {
		msgChan <- sdk.NewMessage(sdk.MsgApplicationDetached, app.Name, oldW.FromRepository)
	}

	// reset fromRepository for all env using it
	envs, err := environment.LoadAllNamesByFromRepository(db, projectID, oldW.FromRepository)
	if err != nil {
		return err
	}

	if err := environment.ResetFromRepository(db, projectID, oldW.FromRepository); err != nil {
		return sdk.WrapError(err, "could not reset fromRepository %s from environments", oldW.FromRepository)
	}

	for _, env := range envs {
		msgChan <- sdk.NewMessage(sdk.MsgEnvironmentDetached, env.Name, oldW.FromRepository)
	}

	return nil
}

func importWorkflowGroups(db gorp.SqlExecutor, w *sdk.Workflow) error {
	if len(w.Groups) > 0 {
		if err := group.DeleteAllWorkflowGroups(db, w.ID); err != nil {
			return err
		}
		for i := range w.Groups {
			g, err := group.LoadByName(context.Background(), db, w.Groups[i].Group.Name)
			if err != nil {
				return sdk.WrapError(err, "unable to load group %s", w.Groups[i].Group.Name)
			}
			w.Groups[i].Group = *g
		}
		if err := group.UpsertAllWorkflowGroups(db, w, w.Groups); err != nil {
			return sdk.WrapError(err, "unable to update workflow")
		}
	}
	return nil
}
