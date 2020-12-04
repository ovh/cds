package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getEnvironmentsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		withUsage := service.FormBool(r, "withUsage")

		tx, errTx := api.mustDB().Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "getEnvironmentsHandler> Cannot start transaction from db")
		}
		defer tx.Rollback() // nolint

		environments, errEnv := environment.LoadEnvironments(tx, projectKey)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "getEnvironmentsHandler> Cannot load environments from db")
		}

		if withUsage {
			for iEnv := range environments {
				environments[iEnv].Usage = &sdk.Usage{}
				wf, errW := workflow.LoadByEnvName(ctx, tx, projectKey, environments[iEnv].Name)
				if errW != nil {
					return sdk.WrapError(errW, "getEnvironmentsHandler> Cannot load workflows linked to environment %s from db", environments[iEnv].Name)
				}
				environments[iEnv].Usage.Workflows = append(environments[iEnv].Usage.Workflows, wf...)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction from db")
		}

		return service.WriteJSON(w, environments, http.StatusOK)
	}
}

func (api *API) getEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		environmentName := vars["environmentName"]
		withUsage := service.FormBool(r, "withUsage")

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "getEnvironmentHandler> Cannot load environment %s for project %s from db", environmentName, projectKey)
		}
		env.Usage = &sdk.Usage{}

		if withUsage {
			wf, errW := workflow.LoadByEnvName(ctx, api.mustDB(), projectKey, environmentName)
			if errW != nil {
				return sdk.WrapError(errW, "getEnvironmentHandler> Cannot load workflows linked to environment %s in project %s", environmentName, projectKey)
			}
			env.Usage.Workflows = wf
		}

		if env.FromRepository != "" {
			proj, err := project.Load(ctx, api.mustDB(), projectKey,
				project.LoadOptions.WithApplicationWithDeploymentStrategies,
				project.LoadOptions.WithPipelines,
				project.LoadOptions.WithEnvironments,
				project.LoadOptions.WithIntegrations)
			if err != nil {
				return err
			}

			wkAscodeHolder, err := workflow.LoadByRepo(ctx, api.mustDB(), *proj, env.FromRepository, workflow.LoadOptions{
				WithTemplate: true,
			})
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.NewErrorFrom(err, "cannot found workflow holder of the environment")
			}
			env.WorkflowAscodeHolder = wkAscodeHolder

			// FIXME from_repository should never be set if the workflow holder was deleted
			if env.WorkflowAscodeHolder == nil {
				env.FromRepository = ""
			}
		}

		return service.WriteJSON(w, env, http.StatusOK)
	}
}

func (api *API) getEnvironmentUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		environmentName := vars["environmentName"]
		usage, err := loadEnvironmentUsage(ctx, api.mustDB(), projectKey, environmentName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load usage for environment %s in project %s", environmentName, projectKey)
		}

		return service.WriteJSON(w, usage, http.StatusOK)
	}
}

func loadEnvironmentUsage(ctx context.Context, db gorp.SqlExecutor, projectKey, envName string) (sdk.Usage, error) {
	usage := sdk.Usage{}

	wf, errW := workflow.LoadByEnvName(ctx, db, projectKey, envName)
	if errW != nil {
		return usage, sdk.WrapError(errW, "loadEnvironmentUsage> Cannot load workflows linked to environment %s in project %s", envName, projectKey)
	}
	usage.Workflows = wf

	// TODO: add usage for envs, apps, pips

	return usage, nil
}

func (api *API) addEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		proj, errProj := project.Load(ctx, api.mustDB(), key, project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "addEnvironmentHandler> Cannot load %s", key)
		}

		var env sdk.Environment
		if err := service.UnmarshalBody(r, &env); err != nil {
			return err
		}
		env.ProjectID = proj.ID

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addEnvironmentHandler> Cannot start transaction")
		}

		defer tx.Rollback() // nolint

		if err := environment.InsertEnvironment(tx, &env); err != nil {
			return sdk.WrapError(err, "Cannot insert environment")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		var errEnvs error
		proj.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), proj.Key)
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "addEnvironmentHandler> Cannot load all environments")
		}

		event.PublishEnvironmentAdd(ctx, key, env, getAPIConsumer(ctx))

		return service.WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) deleteEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		environmentName := vars["environmentName"]

		p, errProj := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "deleteEnvironmentHandler> Cannot load project %s", projectKey)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			if !sdk.ErrorIs(errEnv, sdk.ErrEnvironmentNotFound) {
				log.Warning(ctx, "deleteEnvironmentHandler> Cannot load environment %s: %v", environmentName, errEnv)
			}
			return errEnv
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "deleteEnvironmentHandler> Cannot begin transaction")
		}
		defer tx.Rollback() // nolint

		if err := environment.DeleteEnvironment(tx, env.ID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishEnvironmentDelete(ctx, p.Key, *env, getAPIConsumer(ctx))

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key)
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "deleteEnvironmentHandler> Cannot load environments")
		}
		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) updateAsCodeEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		environmentName := vars["environmentName"]

		branch := FormString(r, "branch")
		message := FormString(r, "message")

		if branch == "" || message == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing branch or message data")
		}

		var env sdk.Environment
		if err := service.UnmarshalBody(r, &env); err != nil {
			return err
		}

		// check application name pattern
		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(env.Name) {
			return sdk.WrapError(sdk.ErrInvalidApplicationPattern, "Environment name %s do not respect pattern", env.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(ctx, tx, key, project.LoadOptions.WithClearKeys)
		if err != nil {
			return err
		}

		envDB, err := environment.LoadEnvironmentByName(tx, key, environmentName)
		if err != nil {
			return sdk.WrapError(err, "cannot load environment %s", environmentName)
		}

		if envDB.FromRepository == "" {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "current environment is not ascode")
		}

		wkHolder, err := workflow.LoadByRepo(ctx, tx, *proj, envDB.FromRepository, workflow.LoadOptions{
			WithTemplate: true,
		})
		if err != nil {
			return err
		}
		if wkHolder.TemplateInstance != nil {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot edit an application that was generated by a template")
		}

		var rootApp *sdk.Application
		if wkHolder.WorkflowData.Node.Context != nil && wkHolder.WorkflowData.Node.Context.ApplicationID != 0 {
			rootApp, err = application.LoadByIDWithClearVCSStrategyPassword(ctx, tx, wkHolder.WorkflowData.Node.Context.ApplicationID)
			if err != nil {
				return err
			}
		}
		if rootApp == nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot find the root application of the workflow %s that hold the pipeline", wkHolder.Name)
		}

		// create keys
		for i := range env.Keys {
			k := &env.Keys[i]
			newKey, err := keys.GenerateKey(k.Name, k.Type)
			if err != nil {
				return err
			}
			k.Public = newKey.Public
			k.Private = newKey.Private
			k.KeyID = newKey.KeyID
		}

		u := getAPIConsumer(ctx)
		env.ProjectID = proj.ID
		envExported, err := environment.ExportEnvironment(tx, env, project.EncryptWithBuiltinKey, fmt.Sprintf("env:%d:%s", envDB.ID, branch))
		if err != nil {
			return err
		}
		wp := exportentities.WorkflowComponents{
			Environments: []exportentities.Environment{envExported},
		}

		ope, err := operation.PushOperationUpdate(ctx, tx, api.Cache, *proj, wp, rootApp.VCSServer, rootApp.RepositoryFullname, branch, message, rootApp.RepositoryStrategy, u)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		api.GoRoutines.Exec(context.Background(), fmt.Sprintf("UpdateAsCodeEnvironmentHandler-%s", ope.UUID), func(ctx context.Context) {
			ed := ascode.EntityData{
				FromRepo:      envDB.FromRepository,
				Type:          ascode.EnvironmentEvent,
				ID:            envDB.ID,
				Name:          envDB.Name,
				OperationUUID: ope.UUID,
			}
			ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, api.GoRoutines, *proj, *wkHolder, *rootApp, ed, u)
		}, api.PanicDump())

		return service.WriteJSON(w, sdk.Operation{
			UUID:   ope.UUID,
			Status: ope.Status,
		}, http.StatusOK)
	}
}

func (api *API) updateEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		environmentName := vars["environmentName"]

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "updateEnvironmentHandler> Cannot load environment %s", environmentName)
		}

		if env.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		p, errProj := project.Load(ctx, api.mustDB(), projectKey)
		if errProj != nil {
			return sdk.WrapError(errProj, "updateEnvironmentHandler> Cannot load project %s", projectKey)
		}

		var envPost sdk.Environment
		if err := service.UnmarshalBody(r, &envPost); err != nil {
			return err
		}

		oldEnv := env
		env.Name = envPost.Name

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updateEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := environment.UpdateEnvironment(tx, env); err != nil {
			return sdk.WrapError(err, "Cannot update environment %s", environmentName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishEnvironmentUpdate(ctx, p.Key, *env, *oldEnv, getAPIConsumer(ctx))

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key)
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "updateEnvironmentHandler> Cannot load environments")
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) cloneEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		environmentName := vars["environmentName"]
		cloneName := vars["cloneName"]

		env, err := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if err != nil {
			return sdk.WrapError(err, "cannot load environment %s", environmentName)
		}

		p, err := project.Load(ctx, api.mustDB(), projectKey)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", projectKey)
		}

		//Load all environments to check if there is another environment with the same name
		envs, err := environment.LoadEnvironments(api.mustDB(), projectKey)
		if err != nil {
			return err
		}

		for _, e := range envs {
			if e.Name == cloneName {
				return sdk.WrapError(sdk.ErrEnvironmentExist, "an environment was found with the same name: %s", cloneName)
			}
		}

		variables := []sdk.EnvironmentVariable{}
		for _, v := range env.Variables {
			// do not clone secret variable to avoid 'secret value not specified'
			if v.Type != sdk.SecretVariable {
				variables = append(variables, v)
			}
		}
		//Set all the data of the environment we want to clone
		envPost := sdk.Environment{
			Name:       cloneName,
			ProjectID:  p.ID,
			ProjectKey: p.Key,
			Variables:  variables,
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start a transaction")
		}

		defer tx.Rollback() // nolint

		//Insert environment
		if err := environment.InsertEnvironment(tx, &envPost); err != nil {
			return sdk.WrapError(err, "Unable to insert environment %s", envPost.Name)
		}

		//Insert variables
		for _, v := range envPost.Variables {
			if err := environment.InsertVariable(tx, envPost.ID, &v, getAPIConsumer(ctx)); err != nil {
				return sdk.WrapError(err, "Unable to insert variable")
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		//return the project with all environments
		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key)
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "cloneEnvironmentHandler> Cannot load environments: %s", errEnvs)
		}

		event.PublishEnvironmentAdd(ctx, p.Key, envPost, getAPIConsumer(ctx))

		return service.WriteJSON(w, p, http.StatusOK)
	}
}
