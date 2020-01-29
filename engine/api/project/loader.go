package project

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc is used as options to loadProject functions
type LoadOptionFunc func(gorp.SqlExecutor, cache.Store, *sdk.Project) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	Default                                 LoadOptionFunc
	WithIcon                                LoadOptionFunc
	WithApplications                        LoadOptionFunc
	WithApplicationNames                    LoadOptionFunc
	WithVariables                           LoadOptionFunc
	WithVariablesWithClearPassword          LoadOptionFunc
	WithPipelines                           LoadOptionFunc
	WithPipelineNames                       LoadOptionFunc
	WithEnvironments                        LoadOptionFunc
	WithEnvironmentNames                    LoadOptionFunc
	WithGroups                              LoadOptionFunc
	WithPermission                          LoadOptionFunc
	WithApplicationVariables                LoadOptionFunc
	WithApplicationWithDeploymentStrategies LoadOptionFunc
	WithKeys                                LoadOptionFunc
	WithWorkflows                           LoadOptionFunc
	WithWorkflowNames                       LoadOptionFunc
	WithClearKeys                           LoadOptionFunc
	WithIntegrations                        LoadOptionFunc
	WithClearIntegrations                   LoadOptionFunc
	WithFavorites                           func(uID string) LoadOptionFunc
	WithFeatures                            LoadOptionFunc
	WithLabels                              LoadOptionFunc
}{
	Default:                                 loadDefault,
	WithIcon:                                loadIcon,
	WithPipelines:                           loadPipelines,
	WithPipelineNames:                       loadPipelineNames,
	WithEnvironments:                        loadEnvironments,
	WithEnvironmentNames:                    loadEnvironmentNames,
	WithGroups:                              loadGroups,
	WithApplications:                        loadApplications,
	WithApplicationNames:                    loadApplicationNames,
	WithVariables:                           loadVariables,
	WithVariablesWithClearPassword:          loadVariablesWithClearPassword,
	WithApplicationVariables:                loadApplicationVariables,
	WithKeys:                                loadKeys,
	WithWorkflows:                           loadWorkflows,
	WithWorkflowNames:                       loadWorkflowNames,
	WithClearKeys:                           loadClearKeys,
	WithIntegrations:                        loadIntegrations,
	WithClearIntegrations:                   loadClearIntegrations,
	WithFavorites:                           loadFavorites,
	WithFeatures:                            loadFeatures,
	WithApplicationWithDeploymentStrategies: loadApplicationWithDeploymentStrategies,
	WithLabels:                              loadLabels,
}

func loadDefault(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	if err := loadVariables(db, store, proj); err != nil {
		return sdk.WithStack(err)
	}
	if err := loadApplications(db, store, proj); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func loadApplications(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	if err := loadApplicationsWithOpts(db, store, proj); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func loadApplicationNames(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	apps, err := application.LoadAllNames(db, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.ApplicationNames = apps
	return nil
}

func loadApplicationWithDeploymentStrategies(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	if proj.Applications == nil {
		if err := loadApplications(db, store, proj); err != nil {
			return sdk.WithStack(err)
		}
	}
	for i := range proj.Applications {
		a := &proj.Applications[i]
		if err := (*application.LoadOptions.WithDeploymentStrategies)(db, store, a); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}

func loadVariables(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	return loadAllVariables(db, store, proj)
}

func loadVariablesWithClearPassword(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	return loadAllVariables(db, store, proj, WithClearPassword())
}

func loadApplicationVariables(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	if proj.Applications == nil {
		if err := loadApplications(db, store, proj); err != nil {
			return sdk.WithStack(err)
		}
	}

	for _, a := range proj.Applications {
		if err := (*application.LoadOptions.WithVariables)(db, store, &a); err != nil {
			return sdk.WithStack(err)
		}
	}

	return nil
}

func loadKeys(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	return LoadAllKeys(db, proj)
}

func loadClearKeys(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	return LoadAllDecryptedKeys(db, proj)
}

func loadIntegrations(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	pf, err := integration.LoadIntegrationsByProjectID(db, proj.ID, false)
	if err != nil {
		return sdk.WrapError(err, "cannot load integrations")
	}
	proj.Integrations = pf
	return nil
}

func loadFeatures(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	// Loads features into a project from the feature flipping provider
	proj.Features = feature.GetFeatures(context.TODO(), store, proj.Key)
	return nil
}

func loadClearIntegrations(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	pf, err := integration.LoadIntegrationsByProjectID(db, proj.ID, true)
	if err != nil {
		return sdk.WrapError(err, "cannot load integrations")
	}
	proj.Integrations = pf
	return nil
}

func loadWorkflows(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	workflows, err := workflow.LoadAll(db, proj.Key)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.Workflows = workflows
	return nil
}

func loadWorkflowNames(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	wfs, err := workflow.LoadAllNames(db, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.WorkflowNames = wfs
	return nil
}

func loadAllVariables(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, args ...GetAllVariableFuncArg) error {
	vars, err := GetAllVariableInProject(db, proj.ID, args...)
	if err != nil && sdk.Cause(err) != sql.ErrNoRows {
		return sdk.WithStack(err)
	}
	proj.Variable = vars
	return nil
}

func loadApplicationsWithOpts(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, opts ...application.LoadOptionFunc) error {
	apps, err := application.LoadAll(db, store, proj.Key, opts...)
	if err != nil && sdk.Cause(err) != sql.ErrNoRows && !sdk.ErrorIs(err, sdk.ErrApplicationNotFound) {
		return sdk.WithStack(err)
	}
	proj.Applications = apps
	return nil
}

func loadIcon(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	icon, err := db.SelectStr("SELECT icon FROM project WHERE id = $1", proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.Icon = icon
	return nil
}

func loadPipelines(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	pipelines, err := pipeline.LoadPipelines(db, proj.ID, false)
	if err != nil && sdk.Cause(err) != sql.ErrNoRows && !sdk.ErrorIs(err, sdk.ErrPipelineNotFound) && !sdk.ErrorIs(err, sdk.ErrPipelineNotAttached) {
		return sdk.WithStack(err)
	}
	proj.Pipelines = append(proj.Pipelines, pipelines...)
	return nil
}

func loadPipelineNames(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	pips, err := pipeline.LoadAllNames(db, store, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.PipelineNames = pips
	return nil
}

func loadEnvironments(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	envs, err := environment.LoadEnvironments(db, proj.Key)
	if err != nil && sdk.Cause(err) != sql.ErrNoRows && !sdk.ErrorIs(err, sdk.ErrNoEnvironment) {
		return sdk.WithStack(err)
	}
	proj.Environments = envs
	return nil
}

func loadEnvironmentNames(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	var err error
	var envs sdk.IDNames

	if envs, err = environment.LoadAllNames(db, proj.ID); err != nil {
		return sdk.WrapError(err, "cannot load environment names")
	}
	proj.EnvironmentNames = envs

	return nil
}

func loadGroups(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
	if err := group.LoadGroupByProject(db, proj); err != nil && sdk.Cause(err) != sql.ErrNoRows {
		return sdk.WithStack(err)
	}
	return nil
}

func loadLabels(db gorp.SqlExecutor, _ cache.Store, proj *sdk.Project) error {
	labels, err := Labels(db, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.Labels = labels
	return nil
}

func loadFavorites(uID string) LoadOptionFunc {
	return func(db gorp.SqlExecutor, store cache.Store, proj *sdk.Project) error {
		count, err := db.SelectInt("SELECT COUNT(1) FROM project_favorite WHERE project_id = $1 AND authentified_user_id = $2", proj.ID, uID)
		if err != nil {
			return sdk.WithStack(err)
		}
		proj.Favorite = count > 0
		return nil
	}
}
