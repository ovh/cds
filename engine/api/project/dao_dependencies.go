package project

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	loadDefault = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		if err := loadVariables(db, store, proj, u); err != nil {
			return sdk.WrapError(err, "application.loadDefault")
		}
		if err := loadApplications(db, store, proj, u); err != nil {
			return sdk.WrapError(err, "application.loadDefault")
		}
		if err := loadPermission(db, store, proj, u); err != nil {
			return sdk.WrapError(err, "application.loadDefault")
		}
		return nil
	}

	loadApplications = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		if err := loadApplicationsWithOpts(db, store, proj); err != nil {
			return sdk.WrapError(err, "application.loadApplications")
		}
		return nil
	}

	loadApplicationNames = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		var err error
		var apps sdk.IDNames

		if apps, err = application.LoadAllNames(db, proj.ID); err != nil {
			return sdk.WrapError(err, "application.loadApplications")
		}
		proj.ApplicationNames = apps

		return nil
	}

	loadApplicationWithDeploymentStrategies = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		if proj.Applications == nil {
			if err := loadApplications(db, store, proj, u); err != nil {
				return sdk.WrapError(err, "application.loadApplicationWithDeploymentStrategies")
			}
		}
		for i := range proj.Applications {
			a := &proj.Applications[i]
			if err := (*application.LoadOptions.WithDeploymentStrategies)(db, store, a); err != nil {
				return sdk.WrapError(err, "application.loadApplicationWithDeploymentStrategies")
			}
		}
		return nil
	}

	loadVariables = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		return loadAllVariables(db, store, proj)
	}

	loadVariablesWithClearPassword = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		return loadAllVariables(db, store, proj, WithClearPassword())
	}

	loadApplicationVariables = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		if proj.Applications == nil {
			if err := loadApplications(db, store, proj, u); err != nil {
				return sdk.WrapError(err, "application.loadApplicationVariables")
			}
		}

		for _, a := range proj.Applications {
			if err := (*application.LoadOptions.WithVariables)(db, store, &a); err != nil {
				return sdk.WrapError(err, "application.loadApplicationVariables")
			}
		}
		return nil
	}

	loadKeys = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		return LoadAllKeys(db, proj)
	}

	loadClearKeys = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		return LoadAllDecryptedKeys(db, proj)
	}

	loadIntegrations = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		pf, err := integration.LoadIntegrationsByProjectID(db, proj.ID, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load integrations")
		}
		proj.Integrations = pf
		return nil
	}

	loadFeatures = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		LoadFeatures(store, proj)
		return nil
	}

	loadClearIntegrations = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		pf, err := integration.LoadIntegrationsByProjectID(db, proj.ID, true)
		if err != nil {
			return sdk.WrapError(err, "Cannot load integrations")
		}
		proj.Integrations = pf
		return nil
	}

	loadWorkflows = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		workflows, errW := workflow.LoadAll(db, proj.Key)
		if errW != nil {
			log.Error("Unable to load workflows for project %s: %v", proj.Key, errW)
		}
		proj.Workflows = workflows
		return nil
	}

	loadWorkflowNames = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		var err error
		var wfs sdk.IDNames

		if wfs, err = workflow.LoadAllNames(db, proj.ID, u); err != nil {
			return sdk.WrapError(err, "workflow.loadworkflownames")
		}
		proj.WorkflowNames = wfs

		return nil
	}

	lockAndWaitProject = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		return nil
	}

	loadAllVariables = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, args ...GetAllVariableFuncArg) error {
		vars, err := GetAllVariableInProject(db, proj.ID, args...)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadAllVariables")
		}
		proj.Variable = vars
		return nil
	}

	loadApplicationsWithOpts = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, opts ...application.LoadOptionFunc) error {
		var err error
		proj.Applications, err = application.LoadAll(db, store, proj.Key, opts...)
		if err != nil && sdk.Cause(err) != sql.ErrNoRows && !sdk.ErrorIs(err, sdk.ErrApplicationNotFound) {
			return sdk.WrapError(err, "application.loadApplicationsWithOpts")
		}
		return nil
	}

	loadIcon = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		icon, err := db.SelectStr("SELECT icon FROM project WHERE id = $1", proj.ID)
		if err != nil {
			return sdk.WrapError(err, "project.loadIcon")
		}
		proj.Icon = icon
		return nil
	}

	loadPipelines = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		pipelines, errPip := pipeline.LoadPipelines(db, proj.ID, false)
		if errPip != nil && sdk.Cause(errPip) != sql.ErrNoRows && !sdk.ErrorIs(errPip, sdk.ErrPipelineNotFound) && !sdk.ErrorIs(errPip, sdk.ErrPipelineNotAttached) {
			return sdk.WrapError(errPip, "application.loadPipelines")
		}
		proj.Pipelines = append(proj.Pipelines, pipelines...)
		return nil
	}

	loadPipelineNames = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		var err error
		var pips sdk.IDNames

		if pips, err = pipeline.LoadAllNames(db, store, proj.ID); err != nil {
			return sdk.WrapError(err, "pipeline.loadpipelinenames")
		}
		proj.PipelineNames = pips

		return nil
	}

	loadEnvironments = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		envs, errEnv := environment.LoadEnvironments(db, proj.Key, true, u)
		if errEnv != nil && sdk.Cause(errEnv) != sql.ErrNoRows && !sdk.ErrorIs(errEnv, sdk.ErrNoEnvironment) {
			return sdk.WrapError(errEnv, "application.loadEnvironments")
		}
		proj.Environments = envs
		return nil
	}

	loadEnvironmentNames = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		var err error
		var envs sdk.IDNames

		if envs, err = environment.LoadAllNames(db, proj.ID); err != nil {
			return sdk.WrapError(err, "cannot load environment names")
		}
		proj.EnvironmentNames = envs

		return nil
	}

	loadGroups = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		if err := group.LoadGroupByProject(db, proj); err != nil && sdk.Cause(err) != sql.ErrNoRows {
			return sdk.WrapError(err, "application.loadGroups")
		}
		return nil
	}

	loadPermission = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		proj.Permission = permission.ProjectPermission(proj.Key, u)
		return nil
	}

	loadLabels = func(db gorp.SqlExecutor, _ cache.Store, proj *sdk.Project, _ *sdk.User) error {
		labels, err := Labels(db, proj.ID)
		if err != nil {
			return sdk.WithStack(err)
		}
		proj.Labels = labels
		return nil
	}

	loadFavorites = func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, u *sdk.User) error {
		count, err := db.SelectInt("SELECT COUNT(1) FROM project_favorite WHERE project_id = $1 AND user_id = $2", proj.ID, u.ID)
		if err != nil {
			return sdk.WithStack(err)
		}
		proj.Favorite = count > 0

		return nil
	}
)
