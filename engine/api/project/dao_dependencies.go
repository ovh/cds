package project

import (
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

func loadAllVariables(db gorp.SqlExecutor, proj *sdk.Project, args ...GetAllVariableFuncArg) error {
	vars, err := GetAllVariableInProject(db, proj.ID, args...)
	if err != nil {
		return err
	}
	proj.Variable = vars
	return nil
}

func loadApplications(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	var err error
	proj.Applications, err = application.LoadApplications(db, proj.Key, true, true, u)
	if err != nil {
		return err
	}
	return nil
}

func loadPipelines(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	pipelines, errPip := pipeline.LoadPipelines(db, proj.ID, false, u)
	if errPip != nil {
		log.Warning("getProject: Cannot load pipelines from db: %s\n", errPip)
		return errPip
	}
	proj.Pipelines = append(proj.Pipelines, pipelines...)
	return nil
}

func loadEnvironments(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	envs, errEnv := environment.LoadEnvironments(db, proj.Key, true, u)
	if errEnv != nil {
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

func loadGroups(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	return group.LoadGroupByProject(db, proj)
}

func loadPermission(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	proj.Permission = permission.ProjectPermission(proj.Key, u)
	return nil
}

func loadRepositoriesManagers(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	var errRepos error
	proj.ReposManager, errRepos = repositoriesmanager.LoadAllForProject(db, proj.Key)
	if errRepos != nil {
		log.Warning("loadRepositoriesManagers> Cannot load repos manager for project %s: %s\n", proj.Key, errRepos)
		return errRepos
	}
	return nil
}
