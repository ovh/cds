package queue

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// RunPipeline  the given pipeline with the given parameters
func RunPipeline(db gorp.SqlExecutor, projectKey string, app *sdk.Application, pipelineName string, environmentName string, params []sdk.Parameter, version int64, trigger sdk.PipelineBuildTrigger, user *sdk.User) (*sdk.PipelineBuild, error) {
	// Load pipeline + Args + stage + action
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load pipeline %s: %s\n", pipelineName, err)
		return nil, err
	}
	parameters, err := pipeline.GetAllParametersInPipeline(db, p.ID)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load pipeline %s parameters: %s\n", pipelineName, err)
		return nil, err
	}
	p.Parameter = parameters

	// Pipeline type check
	if p.Type == sdk.BuildPipeline && environmentName != "" && environmentName != sdk.DefaultEnv.Name {
		log.Warning("scheduler.Run> Pipeline %s/%s/%s is a %s pipeline, but environment '%s' was provided\n", projectKey, app.Name, pipelineName, p.Type, environmentName)
		return nil, sdk.ErrEnvironmentProvided
	}
	if p.Type != sdk.BuildPipeline && (environmentName == "" || environmentName == sdk.DefaultEnv.Name) {
		log.Warning("scheduler.Run> Pipeline %s/%s/%s is a %s pipeline, but no environment was provided\n", projectKey, app.Name, pipelineName, p.Type)
		return nil, sdk.ErrNoEnvironmentProvided
	}

	applicationPipelineParams, err := application.GetAllPipelineParam(db, app.ID, p.ID)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load application pipeline args: %s\n", err)
		return nil, err
	}

	// Load project + var
	projectData, err := project.Load(db, projectKey, user)
	if err != nil {
		log.Warning("scheduler.Run> Cannot load project %s: %s\n", projectKey, err)
		return nil, err
	}
	projectsVar, err := project.GetAllVariableInProject(db, projectData.ID, project.WithClearPassword())
	if err != nil {
		log.Warning("scheduler.Run> Cannot load project variable: %s\n", err)
		return nil, err
	}
	projectData.Variable = projectsVar
	var env *sdk.Environment
	if environmentName != "" && environmentName != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, projectKey, environmentName)
		if err != nil {
			log.Warning("scheduler.Run> Cannot load environment %s for project %s: %s\n", environmentName, projectKey, err)
			return nil, err
		}
	} else {
		env = &sdk.DefaultEnv
	}

	pb, err := pipeline.InsertPipelineBuild(db, projectData, p, app, applicationPipelineParams, params, env, version, trigger)
	if err != nil {
		log.Warning("scheduler.Run> Cannot start pipeline %s: %s\n", pipelineName, err)
		return nil, err
	}

	return pb, nil
}
