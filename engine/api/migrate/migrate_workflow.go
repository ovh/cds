package migrate

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	STATUS_START    = "STARTED"
	STATUS_CLEANING = "CLEANING"
	STATUS_DONE     = "DONE"
)

func MigrateToWorkflow(db gorp.SqlExecutor, store cache.Store, cdTree []sdk.CDPipeline, proj *sdk.Project, u *sdk.User, force bool) error {
	for i := range cdTree {
		oldW := cdTree[i]
		name := "w" + oldW.Application.Name
		if len(cdTree) > 1 {
			name = fmt.Sprintf("%s_%d", name, i)
		}
		newW := sdk.Workflow{
			Name:       name,
			ProjectID:  proj.ID,
			ProjectKey: proj.Key,
		}

		if err := addGroupOnWorkflow(db, &newW, &oldW.Application); err != nil {
			return sdk.WrapError(err, "MigrateToWorkflow")
		}

		currentApplicationID := oldW.Application.ID

		n, err := migratePipeline(db, store, proj, oldW, currentApplicationID, u)
		if err != nil {
			return sdk.WrapError(err, "MigrateToWorkflow")
		}
		newW.Root = n

		if force {
			w, err := workflow.Load(db, store, proj, newW.Name, u, workflow.LoadOptions{})
			if err == nil {
				if errD := workflow.Delete(db, store, proj, w, u); errD != nil {
					return sdk.WrapError(errD, "MigrateToWorkflow")
				}
			}
		}

		if errW := workflow.Insert(db, store, &newW, proj, u); errW != nil {
			return sdk.WrapError(errW, "MigrateToWorkflow")
		}

		for _, g := range newW.Groups {
			if err := workflow.AddGroup(db, &newW, g); err != nil {
				return sdk.WrapError(err, "MigrateToWorkflow> Cannot add group")
			}
		}
	}
	return nil
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
		newNode.Context.Application = &oldPipeline.Application
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
			log.Warning("%+v", childPip.Trigger.Parameters)
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
