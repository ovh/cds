package api

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getEnvironmentsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		withUsage := FormBool(r, "withUsage")

		tx, errTx := api.mustDB().Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "getEnvironmentsHandler> Cannot start transaction from db")
		}
		defer tx.Rollback()

		environments, errEnv := environment.LoadEnvironments(tx, projectKey, true, deprecatedGetUser(ctx))
		if errEnv != nil {
			return sdk.WrapError(errEnv, "getEnvironmentsHandler> Cannot load environments from db")
		}

		if withUsage {
			for iEnv := range environments {
				environments[iEnv].Usage = &sdk.Usage{}
				wf, errW := workflow.LoadByEnvName(tx, projectKey, environments[iEnv].Name)
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
		withUsage := FormBool(r, "withUsage")

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "getEnvironmentHandler> Cannot load environment %s for project %s from db", environmentName, projectKey)
		}
		env.Usage = &sdk.Usage{}

		if withUsage {
			wf, errW := workflow.LoadByEnvName(api.mustDB(), projectKey, environmentName)
			if errW != nil {
				return sdk.WrapError(errW, "getEnvironmentHandler> Cannot load workflows linked to environment %s in project %s", environmentName, projectKey)
			}
			env.Usage.Workflows = wf
		}

		env.Permission = permission.ProjectPermission(projectKey, deprecatedGetUser(ctx))

		return service.WriteJSON(w, env, http.StatusOK)
	}
}

func (api *API) getEnvironmentUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		environmentName := vars["environmentName"]
		usage, err := loadEnvironmentUsage(api.mustDB(), projectKey, environmentName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load usage for environment %s in project %s", environmentName, projectKey)
		}

		return service.WriteJSON(w, usage, http.StatusOK)
	}
}

func loadEnvironmentUsage(db gorp.SqlExecutor, projectKey, envName string) (sdk.Usage, error) {
	usage := sdk.Usage{}

	wf, errW := workflow.LoadByEnvName(db, projectKey, envName)
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

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.Default)
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

		defer tx.Rollback()

		if err := environment.InsertEnvironment(tx, &env); err != nil {
			return sdk.WrapError(err, "Cannot insert environment")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		var errEnvs error
		proj.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), proj.Key, true, deprecatedGetUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "addEnvironmentHandler> Cannot load all environments")
		}

		event.PublishEnvironmentAdd(key, env, deprecatedGetUser(ctx))

		return service.WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) deleteEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		environmentName := vars["environmentName"]

		p, errProj := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "deleteEnvironmentHandler> Cannot load project %s", projectKey)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			if !sdk.ErrorIs(errEnv, sdk.ErrEnvironmentNotFound) {
				log.Warning("deleteEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, errEnv)
			}
			return errEnv
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "deleteEnvironmentHandler> Cannot begin transaction")
		}
		defer tx.Rollback()

		if err := environment.DeleteEnvironment(tx, env.ID); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishEnvironmentDelete(p.Key, *env, deprecatedGetUser(ctx))

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key, true, deprecatedGetUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "deleteEnvironmentHandler> Cannot load environments")
		}
		return service.WriteJSON(w, p, http.StatusOK)
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

		p, errProj := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
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
		defer tx.Rollback()

		if err := environment.UpdateEnvironment(tx, env); err != nil {
			return sdk.WrapError(err, "Cannot update environment %s", environmentName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishEnvironmentUpdate(p.Key, *env, *oldEnv, deprecatedGetUser(ctx))

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key, true, deprecatedGetUser(ctx))
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

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "cloneEnvironmentHandler> Cannot load environment %s: %s", environmentName, errEnv)
		}

		p, errProj := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if errProj != nil {
			return sdk.WrapError(errProj, "cloneEnvironmentHandler> Cannot load project %s: %s", projectKey, errProj)
		}

		//Load all environments to check if there is another environment with the same name
		envs, err := environment.LoadEnvironments(api.mustDB(), projectKey, false, deprecatedGetUser(ctx))
		if err != nil {
			return err
		}

		for _, e := range envs {
			if e.Name == cloneName {
				return sdk.WrapError(sdk.ErrConflict, "cloneEnvironmentHandler> an environment was found with the same name: %s", cloneName)
			}
		}

		//Set all the data of the environment we want to clone
		envPost := sdk.Environment{
			Name:       cloneName,
			ProjectID:  p.ID,
			ProjectKey: p.Key,
			Variable:   env.Variable,
			Permission: env.Permission,
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Unable to start a transaction")
		}

		defer tx.Rollback()

		//Insert environment
		if err := environment.InsertEnvironment(tx, &envPost); err != nil {
			return sdk.WrapError(err, "Unable to insert environment %s", envPost.Name)
		}

		//Insert variables
		for _, v := range envPost.Variable {
			if err := environment.InsertVariable(tx, envPost.ID, &v, deprecatedGetUser(ctx)); err != nil {
				return sdk.WrapError(err, "Unable to insert variable")
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		//return the project with all environments
		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key, true, deprecatedGetUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "cloneEnvironmentHandler> Cannot load environments: %s", errEnvs)
		}

		event.PublishEnvironmentAdd(p.Key, envPost, deprecatedGetUser(ctx))

		return service.WriteJSON(w, p, http.StatusOK)
	}
}
