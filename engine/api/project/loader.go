package project

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc is used as options to loadProject functions
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, *sdk.Project) error

// LoadOptions provides all options on project loads functions
var LoadOptions = struct {
	Default                                 LoadOptionFunc
	WithApplications                        LoadOptionFunc
	WithApplicationNames                    LoadOptionFunc
	WithVariables                           LoadOptionFunc
	WithVariablesWithClearPassword          LoadOptionFunc
	WithPipelines                           LoadOptionFunc
	WithPipelineNames                       LoadOptionFunc
	WithEnvironments                        LoadOptionFunc
	WithEnvironmentNames                    LoadOptionFunc
	WithGroups                              LoadOptionFunc
	WithApplicationVariables                LoadOptionFunc
	WithApplicationKeys                     LoadOptionFunc
	WithApplicationWithDeploymentStrategies LoadOptionFunc
	WithKeys                                LoadOptionFunc
	WithWorkflows                           LoadOptionFunc
	WithWorkflowNames                       LoadOptionFunc
	WithClearKeys                           LoadOptionFunc
	WithIntegrations                        LoadOptionFunc
	WithClearIntegrations                   LoadOptionFunc
	WithLabels                              LoadOptionFunc
}{
	Default:                                 loadDefault,
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
	WithApplicationKeys:                     loadApplicationKeys,
	WithKeys:                                loadKeys,
	WithWorkflows:                           loadWorkflows,
	WithWorkflowNames:                       loadWorkflowNames,
	WithClearKeys:                           loadClearKeys,
	WithIntegrations:                        loadIntegrations,
	WithClearIntegrations:                   loadClearIntegrations,
	WithApplicationWithDeploymentStrategies: loadApplicationWithDeploymentStrategies,
	WithLabels:                              loadLabels,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	if err := loadVariables(ctx, db, proj); err != nil {
		return sdk.WithStack(err)
	}
	if err := loadApplications(ctx, db, proj); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func loadApplications(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	if err := loadApplicationsWithOpts(ctx, db, proj); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func loadApplicationNames(_ context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	apps, err := application.LoadAllNames(db, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.ApplicationNames = apps
	return nil
}

func loadApplicationWithDeploymentStrategies(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	if proj.Applications == nil {
		if err := loadApplications(ctx, db, proj); err != nil {
			return sdk.WithStack(err)
		}
	}
	for i := range proj.Applications {
		a := &proj.Applications[i]
		if err := application.LoadOptions.WithDeploymentStrategies(ctx, db, a); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}

func loadVariables(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	vars, err := LoadAllVariables(ctx, db, proj.ID)
	if err != nil {
		return err
	}
	proj.Variables = vars
	return nil
}

func loadVariablesWithClearPassword(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	vars, err := LoadAllVariablesWithDecrytion(ctx, db, proj.ID)
	if err != nil {
		return err
	}
	proj.Variables = vars
	return nil
}

func loadApplicationVariables(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	if proj.Applications == nil {
		if err := loadApplications(ctx, db, proj); err != nil {
			return sdk.WithStack(err)
		}
	}

	for _, a := range proj.Applications {
		if err := application.LoadOptions.WithVariables(ctx, db, &a); err != nil {
			return sdk.WithStack(err)
		}
	}

	return nil
}

func loadApplicationKeys(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	if proj.Applications == nil {
		if err := loadApplications(ctx, db, proj); err != nil {
			return sdk.WithStack(err)
		}
	}

	for _, a := range proj.Applications {
		if err := application.LoadOptions.WithKeys(ctx, db, &a); err != nil {
			return sdk.WithStack(err)
		}
	}

	return nil
}

func loadKeys(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	keys, err := LoadAllKeys(ctx, db, proj.ID)
	if err != nil {
		return err
	}
	proj.Keys = keys
	return nil
}

func loadClearKeys(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	keys, err := LoadAllKeysWithPrivateContent(ctx, db, proj.ID)
	if err != nil {
		return err
	}
	proj.Keys = keys
	return nil
}

func loadIntegrations(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	pf, err := integration.LoadIntegrationsByProjectID(ctx, db, proj.ID)
	if err != nil {
		return sdk.WrapError(err, "cannot load integrations")
	}
	proj.Integrations = pf
	return nil
}

func loadClearIntegrations(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	pf, err := integration.LoadIntegrationsByProjectIDWithClearPassword(ctx, db, proj.ID)
	if err != nil {
		return sdk.WrapError(err, "cannot load integrations")
	}
	proj.Integrations = pf
	return nil
}

func loadWorkflows(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	workflows, err := workflow.LoadAll(ctx, db, proj.Key)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.Workflows = workflows
	return nil
}

func loadWorkflowNames(_ context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	wfs, err := workflow.LoadAllNames(db, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.WorkflowNames = wfs
	return nil
}

func loadApplicationsWithOpts(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, opts ...application.LoadOptionFunc) error {
	apps, err := application.LoadAll(ctx, db, proj.Key, opts...)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.Applications = apps
	return nil
}

func loadPipelines(_ context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	pipelines, err := pipeline.LoadPipelines(db, proj.ID, false)
	if err != nil && sdk.Cause(err) != sql.ErrNoRows && !sdk.ErrorIs(err, sdk.ErrPipelineNotFound) && !sdk.ErrorIs(err, sdk.ErrPipelineNotAttached) {
		return sdk.WithStack(err)
	}
	proj.Pipelines = append(proj.Pipelines, pipelines...)
	return nil
}

func loadPipelineNames(_ context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	pips, err := pipeline.LoadAllNames(db, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.PipelineNames = pips
	return nil
}

func loadEnvironments(_ context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	envs, err := environment.LoadEnvironments(db, proj.Key)
	if err != nil && sdk.Cause(err) != sql.ErrNoRows && !sdk.ErrorIs(err, sdk.ErrNoEnvironment) {
		return sdk.WithStack(err)
	}
	proj.Environments = envs
	return nil
}

func loadEnvironmentNames(_ context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	var err error
	var envs sdk.IDNames

	if envs, err = environment.LoadAllNames(db, proj.ID); err != nil {
		return sdk.WrapError(err, "cannot load environment names")
	}
	proj.EnvironmentNames = envs

	return nil
}

func loadGroups(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	if err := group.LoadGroupsIntoProject(ctx, db, proj); err != nil && sdk.Cause(err) != sql.ErrNoRows {
		return err
	}
	return nil
}

func loadLabels(_ context.Context, db gorp.SqlExecutor, proj *sdk.Project) error {
	labels, err := Labels(db, proj.ID)
	if err != nil {
		return sdk.WithStack(err)
	}
	proj.Labels = labels
	return nil
}
