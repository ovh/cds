package migrate

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const DEPRECATEDGitClone = "DEPRECATED_GitClone"

// MigrateActionDEPRECATEDGitClone is temporary code
func MigrateActionDEPRECATEDGitClone(DBFunc func() *gorp.DbMap, store cache.Store) error {
	log.Info("MigrateActionDEPRECATEDGitClone> Begin")
	defer log.Info("MigrateActionDEPRECATEDGitClone> End")

	pipelines, err := action.GetPipelineUsingAction(DBFunc(), DEPRECATEDGitClone)

	if err != nil {
		return err
	}

	for _, p := range pipelines {
		log.Info("MigrateActionDEPRECATEDGitClone> Migrate %s/%s", p.ProjKey, p.PipName)

		tx, err := DBFunc().Begin()
		if err != nil {
			return sdk.WrapError(err, "MigrateActionDEPRECATEDGitClone> Cannot start transaction")
		}
		var id int64
		// Lock the job (action)
		if err := tx.QueryRow("select id from action where id = $1 for update nowait", p.ActionID).Scan(&id); err != nil {
			log.Info("MigrateActionDEPRECATEDGitClone> unable to take lock on action table: %v", err)
			tx.Rollback()
			continue
		}
		_ = id // we don't care about it
		if err := MigrateActionDEPRECATEDGitClonePipeline(tx, store, p); err != nil {
			log.Error("MigrateActionDEPRECATEDGitClone> %v", err)
			tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "MigrateActionDEPRECATEDGitClone> Cannot commit transaction")
		}

		log.Info("MigrateActionDEPRECATEDGitClone> Migrate %s/%s DONE", p.ProjKey, p.PipName)
	}

	return nil
}

// MigrateActionDEPRECATEDGitClonePipeline is the unitary function
func MigrateActionDEPRECATEDGitClonePipeline(db gorp.SqlExecutor, store cache.Store, p action.PipelineUsingAction) error {
	//Override the appname with the application in workflow node context if needed
	if p.AppName == "" && p.WorkflowName != "" {
		proj, err := project.Load(db, store, p.ProjKey, nil, project.LoadOptions.WithPlatforms)
		if err != nil {
			return err
		}
		w, err := workflow.Load(context.TODO(), db, store, proj, p.WorkflowName, nil, workflow.LoadOptions{})
		if err != nil {
			return err
		}
		node := w.GetNodeByName(p.WorkflowNodeName)
		if node == nil {
			return sdk.ErrWorkflowNodeNotFound
		}
		if node.Context != nil && node.Context.Application != nil {
			p.AppName = node.Context.Application.Name
		}
	}

	pip, err := pipeline.LoadPipeline(db, p.ProjKey, p.PipName, true)
	if err != nil {
		return sdk.WrapError(err, "unable to load pipeline")
	}

	for _, s := range pip.Stages {
		for _, j := range s.Jobs {
			var migrateJob bool
			for _, a := range j.Action.Actions {
				if a.Name == DEPRECATEDGitClone {
					log.Info("MigrateActionDEPRECATEDGitClone> Migrate %s/%s/%s(%d)", p.ProjKey, p.PipName, j.Action.Name, j.Action.ID)
					migrateJob = true
					break
				}
			}
			if migrateJob {
				if err := MigrateActionDEPRECATEDGitCloneJob(db, store, p.ProjKey, p.PipName, p.AppName, j); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

var (
	originalGitClone *sdk.Action
	anAdminID        int64
)

// MigrateActionDEPRECATEDGitCloneJob is the unitary function
func MigrateActionDEPRECATEDGitCloneJob(db gorp.SqlExecutor, store cache.Store, pkey, pipName, appName string, j sdk.Job) error {
	mapReplacement := make(map[int]sdk.Action)

	var err error
	//Load the builtin gitclone action is needed
	if originalGitClone == nil {
		originalGitClone, err = action.LoadPublicAction(db, sdk.GitCloneAction)
		if err != nil {
			return err
		}
	}

	//Load the first admin we can
	if anAdminID == 0 {
		users, err := user.LoadUsers(db)
		if err != nil {
			return err
		}
		for _, u := range users {
			if u.Admin {
				anAdminID = u.ID
				break
			}
		}
	}

	//Load the project
	proj, err := project.Load(db, store, pkey, nil, project.LoadOptions.WithVariables)
	if err != nil {
		return err
	}
	projKeys := []sdk.Variable{}
	for _, v := range proj.Variable {
		if v.Type == sdk.KeyVariable {
			projKeys = append(projKeys, v)
		}
	}

	//Load the application
	log.Debug("load application %s", appName)
	app, err := application.LoadByName(db, store, pkey, appName, nil, application.LoadOptions.WithVariables)
	if err != nil {
		log.Warning("MigrateActionDEPRECATEDGitCloneJob> application.LoadByName> %v", err)
	}

	appKeys := []sdk.Variable{}
	if app != nil {
		for _, v := range app.Variable {
			if v.Type == sdk.KeyVariable {
				appKeys = append(appKeys, v)
			}
		}
	}

	//Check all the steps of the job
	for i := range j.Action.Actions {
		step := &j.Action.Actions[i]
		log.Debug("MigrateActionDEPRECATEDGitCloneJob>CheckJob> Checking step %s", step.Name)

		if step.Name == DEPRECATEDGitClone {
			//Migrate this step
			url := sdk.ParameterFind(&step.Parameters, "url")
			directory := sdk.ParameterFind(&step.Parameters, "directory")
			branch := sdk.ParameterFind(&step.Parameters, "branch")
			commit := sdk.ParameterFind(&step.Parameters, "commit")

			newGitClone := sdk.Action{
				Name:       sdk.GitCloneAction,
				Enabled:    true,
				Type:       sdk.DefaultAction,
				Parameters: originalGitClone.Parameters,
			}

			//Keep the old parameters
			sdk.ParameterFind(&newGitClone.Parameters, "url").Value = url.Value
			sdk.ParameterFind(&newGitClone.Parameters, "directory").Value = directory.Value
			sdk.ParameterFind(&newGitClone.Parameters, "branch").Value = branch.Value
			sdk.ParameterFind(&newGitClone.Parameters, "commit").Value = commit.Value
			sdk.ParameterFind(&newGitClone.Parameters, "user").Value = ""
			sdk.ParameterFind(&newGitClone.Parameters, "password").Value = ""

			//If there is an application key or a project key, use it
			switch {
			case len(appKeys) > 0:
				sdk.ParameterFind(&newGitClone.Parameters, "privateKey").Value = fmt.Sprintf("{{.cds.app.%s}}", appKeys[0].Name)
			case len(projKeys) > 0:
				sdk.ParameterFind(&newGitClone.Parameters, "privateKey").Value = fmt.Sprintf("{{.cds.proj.%s}}", projKeys[0].Name)
			default:
				log.Warning("MigrateActionDEPRECATEDGitCloneJob> Skipping [%s] %s/%s (%s) : can't find suitable key", proj.Key, proj.Name, pipName, j.Action.Name)
				continue
			}

			mapReplacement[i] = newGitClone
			continue
		}
	}

	//Just replace DEPRECATED_GitClone steps by builtin GitClone
	for i, a := range mapReplacement {
		j.Action.Actions[i] = a
	}

	//Updte in database
	return action.UpdateActionDB(db, &j.Action, anAdminID)
}
