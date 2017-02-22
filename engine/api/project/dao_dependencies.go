package project

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var (
	dontLoadApplications = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		return nil
	}

	dontLoadVariables = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		return nil
	}

	dontLoadApplicationPipelines = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		return nil
	}

	dontLoadApplicationVariables = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		return nil
	}

	loadAllVariables = func(db gorp.SqlExecutor, proj *sdk.Project, args ...GetAllVariableFuncArg) error {
		vars, err := GetAllVariableInProject(db, proj.ID, args...)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		proj.Variable = vars
		return nil
	}

	loadApplications = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User, withPipelines, withVariables bool) error {
		var err error
		proj.Applications, err = application.LoadApplications(db, proj.Key, withPipelines, withVariables, u)
		if err != nil && err != sql.ErrNoRows && err != sdk.ErrApplicationNotFound {
			return err
		}
		return nil
	}

	loadPipelines = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		pipelines, errPip := pipeline.LoadPipelines(db, proj.ID, false, u)
		if errPip != nil && errPip != sql.ErrNoRows && errPip != sdk.ErrPipelineNotFound && errPip != sdk.ErrPipelineNotAttached {
			log.Warning("getProject: Cannot load pipelines from db: %s\n", errPip)
			return errPip
		}
		proj.Pipelines = append(proj.Pipelines, pipelines...)
		return nil
	}

	loadEnvironments = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		envs, errEnv := environment.LoadEnvironments(db, proj.Key, true, u)
		if errEnv != nil && errEnv != sql.ErrNoRows && errEnv != sdk.ErrNoEnvironment {
			log.Warning("loadEnvironments> Cannot load environments from db: %s\n", errEnv)
			return errEnv

		}
		proj.Environments = append(proj.Environments, envs...)

		for i := range proj.Environments {
			env := &proj.Environments[i]
			env.Permission = permission.EnvironmentPermission(env.ID, u)
		}

		return nil
	}

	loadGroups = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		return group.LoadGroupByProject(db, proj)
	}

	loadPermission = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		proj.Permission = permission.ProjectPermission(proj.Key, u)
		return nil
	}

	loadRepositoriesManagers = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		var errRepos error
		proj.ReposManager, errRepos = repositoriesmanager.LoadAllForProject(db, proj.Key)
		if errRepos != nil && errRepos != sql.ErrNoRows && errRepos != sdk.ErrNoReposManager {
			log.Warning("loadRepositoriesManagers> Cannot load repos manager for project %s: %s\n", proj.Key, errRepos)
			return errRepos
		}
		return nil
	}
)
