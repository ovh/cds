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
	"github.com/ovh/cds/sdk"
)

var (
	loadDefault = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		if err := loadVariables(db, proj, u); err != nil {
			return sdk.WrapError(err, "application.loadDefault")
		}
		if err := loadApplications(db, proj, u); err != nil {
			return sdk.WrapError(err, "application.loadDefault")
		}
		if err := loadApplicationPipelines(db, proj, u); err != nil {
			return sdk.WrapError(err, "application.loadDefault")
		}
		return nil
	}

	loadApplications = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		if err := loadApplicationsWithOpts(db, proj, u); err != nil {
			return sdk.WrapError(err, "application.loadApplications")
		}
		return nil
	}

	loadApplicationPipelines = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		if proj.Applications == nil {
			if err := loadApplications(db, proj, u); err != nil {
				return sdk.WrapError(err, "application.loadApplicationPipelines")
			}
		}

		for _, a := range proj.Applications {
			if err := (*application.LoadOptions.WithPipelines)(db, &a, u); err != nil {
				return sdk.WrapError(err, "application.loadApplicationPipelines")
			}
		}
		return nil
	}

	loadVariables = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		return loadAllVariables(db, proj)
	}

	loadApplicationVariables = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		if proj.Applications == nil {
			if err := loadApplications(db, proj, u); err != nil {
				return sdk.WrapError(err, "application.loadApplicationVariables")
			}
		}

		for _, a := range proj.Applications {
			if err := (*application.LoadOptions.WithVariables)(db, &a, u); err != nil {
				return sdk.WrapError(err, "application.loadApplicationVariables")
			}
		}
		return nil
	}

	loadAllVariables = func(db gorp.SqlExecutor, proj *sdk.Project, args ...GetAllVariableFuncArg) error {
		vars, err := GetAllVariableInProject(db, proj.ID, args...)
		if err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadAllVariables")
		}
		proj.Variable = vars
		return nil
	}

	loadApplicationsWithOpts = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User, opts ...application.LoadOptionFunc) error {
		var err error
		proj.Applications, err = application.LoadAll(db, proj.Key, u, opts...)
		if err != nil && err != sql.ErrNoRows && err != sdk.ErrApplicationNotFound {
			return sdk.WrapError(err, "application.loadApplicationsWithOpts")
		}
		return nil
	}

	loadPipelines = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		pipelines, errPip := pipeline.LoadPipelines(db, proj.ID, false, u)
		if errPip != nil && errPip != sql.ErrNoRows && errPip != sdk.ErrPipelineNotFound && errPip != sdk.ErrPipelineNotAttached {
			return sdk.WrapError(errPip, "application.loadPipelines")
		}
		proj.Pipelines = append(proj.Pipelines, pipelines...)
		return nil
	}

	loadEnvironments = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		envs, errEnv := environment.LoadEnvironments(db, proj.Key, true, u)
		if errEnv != nil && errEnv != sql.ErrNoRows && errEnv != sdk.ErrNoEnvironment {
			return sdk.WrapError(errEnv, "application.loadEnvironments")
		}
		proj.Environments = append(proj.Environments, envs...)

		for i := range proj.Environments {
			env := &proj.Environments[i]
			env.Permission = permission.EnvironmentPermission(env.ID, u)
		}

		return nil
	}

	loadGroups = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		if err := group.LoadGroupByProject(db, proj); err != nil && err != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadGroups")
		}
		return nil
	}

	loadPermission = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		proj.Permission = permission.ProjectPermission(proj.Key, u)
		return nil
	}

	loadRepositoriesManagers = func(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
		var errRepos error
		proj.ReposManager, errRepos = repositoriesmanager.LoadAllForProject(db, proj.Key)
		if errRepos != nil && errRepos != sql.ErrNoRows && errRepos != sdk.ErrNoReposManager {
			return sdk.WrapError(errRepos, "application.loadRepositoriesManagers")
		}
		return nil
	}
)
