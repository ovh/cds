package bootstrap

import (
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//InitiliazeDB inits the database
func InitiliazeDB(defaultValues sdk.DefaultValues, DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()

	if err := group.CreateDefaultGroup(dbGorp, sdk.SharedInfraGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group", sdk.SharedInfraGroupName)
	}

	if strings.TrimSpace(defaultValues.DefaultGroupName) != "" {
		if err := group.CreateDefaultGroup(dbGorp, defaultValues.DefaultGroupName); err != nil {
			return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group")
		}
	}

	if err := group.InitializeDefaultGroupName(dbGorp, defaultValues.DefaultGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	if err := token.Initialize(dbGorp, defaultValues.SharedInfraToken); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	if err := action.CreateBuiltinArtifactActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin Artifact actions")
	}

	if err := action.CreateBuiltinActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin actions")
	}

	if err := environment.CreateBuiltinEnvironments(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin environments")
	}

	return nil
}

const DEPRECATEDGitClone = "DEPRECATED_GitClone"

// MigrateActionDEPRECATEDGitClone is temporary code
func MigrateActionDEPRECATEDGitClone(DBFunc func() *gorp.DbMap, store cache.Store) error {
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

		if err := MigrateActionDEPRECATEDGitClonePipeline(tx, store, p); err != nil {
			log.Error("MigrateActionDEPRECATEDGitClone> %v", err)
			tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "MigrateActionDEPRECATEDGitClone> Cannot commit transaction")
		}
	}

	return nil
}

// MigrateActionDEPRECATEDGitClonePipeline is the unitary function
func MigrateActionDEPRECATEDGitClonePipeline(db gorp.SqlExecutor, store cache.Store, p action.PipelineUsingAction) error {
	//Override the appname with the application in workflow node context if needed
	if p.AppName == "" && p.WorkflowName != "" {
		w, err := workflow.Load(db, store, p.ProjKey, p.WorkflowName, nil, workflow.LoadOptions{})
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
				if err := MigrateActionDEPRECATEDGitCloneJob(db, store, p.ProjKey, p.AppName, j); err != nil {
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
func MigrateActionDEPRECATEDGitCloneJob(db gorp.SqlExecutor, store cache.Store, pkey, appName string, j sdk.Job) error {
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
	proj, err := project.Load(db, store, pkey, nil, project.LoadOptions.WithKeys)
	if err != nil {
		return err
	}
	projKeys := proj.SSHKeys()

	//Load the application
	log.Debug("load application %s", appName)
	app, err := application.LoadByName(db, store, pkey, appName, nil, application.LoadOptions.WithKeys)
	if err != nil {
		return err
	}
	appKeys := app.SSHKeys()

	//Check all the steps of the job
	for i := range j.Action.Actions {
		step := &j.Action.Actions[i]
		log.Debug("CheckJob> Checking step %s", step.Name)

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
				sdk.ParameterFind(&newGitClone.Parameters, "privateKey").Value = appKeys[0].Name
			case len(projKeys) > 0:
				sdk.ParameterFind(&newGitClone.Parameters, "privateKey").Value = projKeys[0].Name
			default:
				sdk.ParameterFind(&newGitClone.Parameters, "privateKey").Value = ""
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
