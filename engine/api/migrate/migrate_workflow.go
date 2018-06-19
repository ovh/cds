package migrate

import (
	"fmt"
	"strings"

	"database/sql"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

const (
	STATUS_START    = "STARTED"
	STATUS_CLEANING = "CLEANING"
	STATUS_DONE     = "DONE"
)

// ToWorkflow migrates old workflow to new workflow
func ToWorkflow(db gorp.SqlExecutor, store cache.Store, cdTree []sdk.CDPipeline, proj *sdk.Project, u *sdk.User, force, disablePrefix, withCurrentVersion, withRepositoryWebHook bool) ([]sdk.Workflow, error) {
	workflows := make([]sdk.Workflow, len(cdTree))
	for i := range cdTree {
		oldW := cdTree[i]
		name := oldW.Application.Name
		if !disablePrefix {
			name = "w" + name
		}

		if len(cdTree) > 1 {
			name = fmt.Sprintf("%s_%d", name, i)
		}
		newW := sdk.Workflow{
			Name:       name,
			ProjectID:  proj.ID,
			ProjectKey: proj.Key,
		}

		if err := addGroupOnWorkflow(db, &newW, &oldW.Application); err != nil {
			return nil, sdk.WrapError(err, "MigrateToWorkflow")
		}

		currentApplicationID := oldW.Application.ID

		n, err := migratePipeline(db, store, proj, oldW, currentApplicationID, u)
		if err != nil {
			return nil, sdk.WrapError(err, "MigrateToWorkflow migratePipeline>")
		}
		newW.Root = n

		if force {
			w, err := workflow.Load(db, store, proj, newW.Name, u, workflow.LoadOptions{})
			if err == nil {
				if errD := workflow.Delete(db, store, proj, w, u); errD != nil {
					return nil, sdk.WrapError(errD, "MigrateToWorkflow workflow.Load>")
				}
			}
		}

		if errW := workflow.Insert(db, store, &newW, proj, u); errW != nil {
			return nil, sdk.WrapError(errW, "MigrateToWorkflow workflow.Insert>")
		}

		if withRepositoryWebHook && newW.Root.Context.Application != nil && newW.Root.Context.Application.VCSServer != "" && newW.Root.Context.Application.RepositoryFullname != "" {
			h := &sdk.WorkflowNodeHook{}
			m, err := workflow.LoadHookModelByName(db, sdk.RepositoryWebHookModelName)
			if err != nil {
				return nil, sdk.WrapError(err, "migratePipeline> hook %s not found", h.WorkflowHookModel.Name)
			}
			h.WorkflowHookModel = *m
			h.WorkflowHookModelID = m.ID
			h.Config = make(map[string]sdk.WorkflowNodeHookConfigValue, len(m.DefaultConfig))
			for k, v := range m.DefaultConfig {
				if _, has := h.Config[k]; !has {
					h.Config[k] = v
				}
			}
			newW.Root.Hooks = []sdk.WorkflowNodeHook{*h}

			oldW, errO := workflow.Load(db, store, proj, newW.Name, u, workflow.LoadOptions{})
			if errO != nil {
				return nil, sdk.WrapError(errO, "migratePipeline> Unable to load old workflow")
			}

			newW.ID = oldW.ID

			if errHr := workflow.HookRegistration(db, store, oldW, newW, proj); errHr != nil {
				return nil, sdk.WrapError(errHr, "migratePipeline> Cannot register hook 2")
			}

			if err := workflow.Update(db, store, &newW, oldW, proj, u); err != nil {
				return nil, sdk.WrapError(err, "migratePipeline> Unable to update workflow 2")
			}

		}

		if withCurrentVersion {
			opts := []pipeline.ExecOptionFunc{
				pipeline.LoadPipelineBuildOpts.WithStatus(sdk.StatusSuccess.String()),
			}
			pbs, errPB := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, oldW.Application.ID, oldW.Pipeline.ID, oldW.Environment.ID, 1, opts...)
			if errPB != nil && errPB != sql.ErrNoRows {
				return nil, sdk.WrapError(err, "migratePipeline> Cannot load pipeline")
			}
			if len(pbs) == 1 {
				if err := workflow.InsertRunNum(db, &newW, pbs[0].Version); err != nil {
					return nil, sdk.WrapError(err, "migratePipeline> Cannot set the version %d", pbs[0].Version)
				}
			}
		}

		for _, g := range newW.Groups {
			if err := workflow.AddGroup(db, &newW, g); err != nil {
				return nil, sdk.WrapError(err, "MigrateToWorkflow> Cannot add group")
			}
		}
		workflows[i] = newW
	}
	return workflows, nil
}

func addGroupOnWorkflow(db gorp.SqlExecutor, w *sdk.Workflow, app *sdk.Application) error {
	if err := application.LoadGroupByApplication(db, app); err != nil {
		return sdk.WrapError(err, "addGroupOnWorkflow> error while LoadGroupByApplication on application %s", app.ID)
	}

	for _, ag := range app.ApplicationGroups {
		if ag.Permission == permission.PermissionReadWriteExecute || ag.Permission == permission.PermissionReadExecute {
			w.Groups = append(w.Groups, ag)
		}
	}
	return nil
}

func migratePipeline(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, oldPipeline sdk.CDPipeline, appID int64, u *sdk.User) (*sdk.WorkflowNode, error) {
	newNode := &sdk.WorkflowNode{
		PipelineID: oldPipeline.Pipeline.ID,
		Context:    &sdk.WorkflowNodeContext{},
	}

	// Check if pipeline use application & env variable
	pip, err := pipeline.LoadPipelineByID(db, oldPipeline.Pipeline.ID, true)
	if err != nil {
		return nil, sdk.WrapError(err, "migratePipeline> Cannot load pipeline")
	}
	foundApp := false
bigloop:
	for _, s := range pip.Stages {
		for _, pre := range s.Prerequisites {
			if strings.Contains(pre.ExpectedValue, "cds.app") {
				foundApp = true
				break bigloop
			}
		}

		for _, j := range s.Jobs {
			for _, r := range j.Action.Requirements {
				if strings.Contains(r.Value, "cds.app") || strings.Contains(r.Value, "git.") {
					foundApp = true
					break bigloop
				}
			}
			for _, step := range j.Action.Actions {
				for _, param := range step.Parameters {
					if strings.Contains(param.Value, "cds.app") || strings.Contains(param.Value, "git.") {
						foundApp = true
						break bigloop
					}
				}
			}
		}
	}
	if foundApp {
		app, err := application.LoadByName(db, store, p.Key, oldPipeline.Application.Name, u, application.LoadOptions.WithClearDeploymentStrategies)
		if err != nil {
			return nil, sdk.WrapError(err, "migratePipeline> Cannot load application")
		}
		newNode.Context.Application = app
	}

	if oldPipeline.Environment.ID != 0 && oldPipeline.Environment.ID != sdk.DefaultEnv.ID {
		newNode.Context.Environment = &oldPipeline.Environment
	}

	// Add trigger
	if len(oldPipeline.SubPipelines) > 0 {
		for _, childPip := range oldPipeline.SubPipelines {
			// Create new trigger
			t := sdk.WorkflowNodeTrigger{}

			// Migrate child pipeline
			n, err := migratePipeline(db, store, p, childPip, appID, u)
			if err != nil {
				return nil, err
			}

			// Migrate pipeline parameter
			n.Context.DefaultPipelineParameters = childPip.Trigger.Parameters

			t.WorkflowDestNode = *n

			// Add Condition on trigger
			for _, c := range childPip.Trigger.Prerequisites {
				t.WorkflowDestNode.Context.Conditions.PlainConditions = append(t.WorkflowDestNode.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
					Variable: c.Parameter,
					Value:    c.ExpectedValue,
					Operator: sdk.WorkflowConditionsOperatorRegex,
				})
			}
			t.WorkflowDestNode.Context.Conditions.PlainConditions = append(t.WorkflowDestNode.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
				Variable: "cds.status",
				Value:    "Success",
				Operator: "eq",
			})
			if childPip.Trigger.Manual {
				t.WorkflowDestNode.Context.Conditions.PlainConditions = append(t.WorkflowDestNode.Context.Conditions.PlainConditions, sdk.WorkflowNodeCondition{
					Variable: "cds.manual",
					Value:    "true",
					Operator: "eq",
				})
			}

			// is sub App
			if childPip.Application.ID != 0 && childPip.Application.ID != appID {
				childPip.Application.WorkflowMigration = STATUS_CLEANING
				childPip.Application.ProjectID = p.ID
				if errA := application.Update(db, store, &childPip.Application, u); errA != nil {
					return nil, sdk.WrapError(errA, "Cannot update subapplication %s", childPip.Application.Name)
				}
			}

			// Add trigger
			newNode.Triggers = append(newNode.Triggers, t)
		}

	}

	return newNode, nil
}
