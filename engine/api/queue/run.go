package queue

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RunPipeline  the given pipeline with the given parameters
func RunPipeline(DBFunc func(context.Context) *gorp.DbMap, store cache.Store, db gorp.SqlExecutor, projectKey string, app *sdk.Application, pipelineName string, environmentName string, params []sdk.Parameter, version int64, trigger sdk.PipelineBuildTrigger, user *sdk.User) (*sdk.PipelineBuild, error) {
	// Load pipeline + Args + stage + action
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		return nil, sdk.WrapError(err, "queue.Run> Cannot load pipeline %s", pipelineName)
	}
	parameters, err := pipeline.GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "queue.Run> Cannot load pipeline %s parameters", pipelineName)
	}
	p.Parameter = parameters

	// Pipeline type check
	if p.Type == sdk.BuildPipeline && environmentName != "" && environmentName != sdk.DefaultEnv.Name {
		return nil, sdk.WrapError(sdk.ErrEnvironmentProvided, "queue.Run> Pipeline %s/%s/%s is a %s pipeline, but environment '%s' was provided", projectKey, app.Name, pipelineName, p.Type, environmentName)
	}
	if p.Type != sdk.BuildPipeline && (environmentName == "" || environmentName == sdk.DefaultEnv.Name) {
		return nil, sdk.WrapError(sdk.ErrNoEnvironmentProvided, "queue.Run> Pipeline %s/%s/%s is a %s pipeline, but no environment was provided", projectKey, app.Name, pipelineName, p.Type)
	}

	applicationPipelineParams, err := application.GetAllPipelineParam(db, app.ID, p.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "queue.Run> Cannot load application pipeline args")
	}

	// Load project + var
	projectData, err := project.Load(db, store, projectKey, user)
	if err != nil {
		return nil, sdk.WrapError(err, "queue.Run> Cannot load project %s", projectKey)
	}
	projectsVar, err := project.GetAllVariableInProject(db, projectData.ID, project.WithClearPassword())
	if err != nil {
		return nil, sdk.WrapError(err, "queue.Run> Cannot load project variable")
	}
	projectData.Variable = projectsVar
	var env *sdk.Environment
	if environmentName != "" && environmentName != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, projectKey, environmentName)
		if err != nil {
			return nil, sdk.WrapError(err, "queue.Run> Cannot load environment %s for project %s", environmentName, projectKey)
		}
	} else {
		env = &sdk.DefaultEnv
	}

	pb, err := pipeline.InsertPipelineBuild(db, store, projectData, p, app, applicationPipelineParams, params, env, version, trigger)
	if err != nil {
		return nil, sdk.WrapError(err, "queue.Run> Cannot start pipeline %s", pipelineName)
	}

	go func() {
		db := DBFunc(context.Background())
		if _, err := pipeline.UpdatePipelineBuildCommits(db, store, projectData, p, app, env, pb); err != nil {
			log.Warning("queue.Run> Unable to update pipeline build commits : %s", err)
		}
	}()

	return pb, nil
}
