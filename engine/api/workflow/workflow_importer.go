package workflow

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Import is able to create a new workflow and all its components
func Import(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, u *sdk.User, force bool, msgChan chan<- sdk.Message, dryRun bool) error {
	ctx, end := observability.Span(ctx, "workflow.Import")
	defer end()

	w.ProjectKey = proj.Key
	w.ProjectID = proj.ID

	mError := new(sdk.MultiError)

	w.Pipelines = map[int64]sdk.Pipeline{}
	var pipelineLoader = func(n *sdk.WorkflowNode) {
		log.Info("loading pipeline %s", n.PipelineName)
		pip, err := pipeline.LoadPipeline(db, proj.Key, n.PipelineName, true)
		if err != nil {
			log.Warning("workflow.Import> %s > Pipeline %s not found: %v", w.Name, n.PipelineName, err)
			mError.Append(fmt.Errorf("pipeline %s/%s not found", proj.Key, n.PipelineName))
			return
		}
		w.Pipelines[n.PipelineID] = *pip
		n.PipelineID = pip.ID
	}
	w.Visit(pipelineLoader)

	var applicationLoader = func(n *sdk.WorkflowNode) {
		if _, has := n.Application(); !has {
			return
		}
		app, err := application.LoadByName(db, store, proj.Key, n.Context.Application.Name, u, application.LoadOptions.WithClearDeploymentStrategies, application.LoadOptions.WithVariables)
		if err != nil {
			log.Warning("workflow.Import> %s > Application %s not found: %v", w.Name, n.Context.Application.Name, err)
			mError.Append(fmt.Errorf("application %s/%s not found", proj.Key, n.Context.Application.Name))
			return
		}
		n.Context.Application = app
	}
	w.Visit(applicationLoader)

	var envLoader = func(n *sdk.WorkflowNode) {
		if _, has := n.Environment(); !has {
			return
		}
		env, err := environment.LoadEnvironmentByName(db, proj.Key, n.Context.Environment.Name)
		if err != nil {
			log.Warning("workflow.Import> %s > Environment %s not found: %v", w.Name, n.Context.Environment.Name, err)
			mError.Append(fmt.Errorf("environment %s/%s not found", proj.Key, n.Context.Environment.Name))
			return
		}
		n.Context.Environment = env
	}
	w.Visit(envLoader)

	var hookLoad = func(n *sdk.WorkflowNode) {
		for i := range n.Hooks {
			h := &n.Hooks[i]
			m, err := LoadHookModelByName(db, h.WorkflowHookModel.Name)
			if err != nil {
				log.Warning("workflow.Import> %s > Hook %s not found: %v", w.Name, h.WorkflowHookModel.Name, err)
				mError.Append(fmt.Errorf("hook %s not found", h.WorkflowHookModel.Name))
				return
			}
			h.WorkflowHookModel = *m
			h.WorkflowHookModelID = m.ID
			for k, v := range m.DefaultConfig {
				if _, has := h.Config[k]; !has {
					h.Config[k] = v
				}
			}
		}
	}
	w.Visit(hookLoad)

	var projectPlatformLoad = func(n *sdk.WorkflowNode) {
		if _, has := n.ProjectPlatform(); !has {
			return
		}
		ppf, err := platform.LoadPlatformsByName(db, proj.Key, n.Context.ProjectPlatform.Name, true)
		if err != nil {
			log.Warning("workflow.Import> %s > Project platform %s not found: %v", n.Context.ProjectPlatform.Name, err)
			mError.Append(fmt.Errorf("Project platform %s not found", n.Context.ProjectPlatform.Name))
			return
		}
		n.Context.ProjectPlatform = &ppf
	}
	w.Visit(projectPlatformLoad)

	if !mError.IsEmpty() {
		return sdk.NewError(sdk.ErrWrongRequest, mError)
	}

	// Default value of history length is 20
	if w.HistoryLength == 0 {
		w.HistoryLength = 20
	}

	doUpdate, errE := Exists(db, proj.Key, w.Name)
	if errE != nil {
		return sdk.WrapError(errE, "Import> Cannot check if workflow exists")
	}

	if !doUpdate {
		if err := Insert(db, store, w, proj, u); err != nil {
			return sdk.WrapError(err, "Import> Unable to insert workflow")
		}
		if msgChan != nil {
			msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedInserted, w.Name)
		}

		// HookRegistration after workflow.Update.  It needs hooks to be created on DB
		if errHr := HookRegistration(ctx, db, store, nil, *w, proj); errHr != nil {
			return sdk.WrapError(errHr, "Import> Cannot register hook")
		}

		return importWorkflowGroups(db, w)
	}

	if !force {
		return sdk.NewError(sdk.ErrConflict, fmt.Errorf("Workflow exists"))
	}

	oldW, errO := Load(ctx, db, store, proj, w.Name, u, LoadOptions{WithIcon: true})
	if errO != nil {
		return sdk.WrapError(errO, "Import> Unable to load old workflow")
	}

	w.ID = oldW.ID
	if err := Update(db, store, w, oldW, proj, u); err != nil {
		return sdk.WrapError(err, "Import> Unable to update workflow")
	}

	if !dryRun {
		// HookRegistration after workflow.Update.  It needs hooks to be created on DB
		if errHr := HookRegistration(ctx, db, store, oldW, *w, proj); errHr != nil {
			return sdk.WrapError(errHr, "Import> Cannot register hook")
		}
	}

	if err := importWorkflowGroups(db, w); err != nil {
		return err
	}

	if msgChan != nil {
		msgChan <- sdk.NewMessage(sdk.MsgWorkflowImportedUpdated, w.Name)
	}
	return nil
}

func importWorkflowGroups(db gorp.SqlExecutor, w *sdk.Workflow) error {
	if len(w.Groups) > 0 {
		for i := range w.Groups {
			g, err := group.LoadGroup(db, w.Groups[i].Group.Name)
			if err != nil {
				return sdk.WrapError(err, "importWorkflowGroups> Unable to load group %s", w.Groups[i].Group.Name)
			}
			w.Groups[i].Group = *g
		}
		if err := upsertAllGroups(db, w, w.Groups); err != nil {
			return sdk.WrapError(err, "importWorkflowGroups> Unable to update workflow")
		}
	}
	return nil
}
