package api

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getEnvironmentsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		withUsage := FormBool(r, "withUsage")

		tx, errTx := api.mustDB(ctx).Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "getEnvironmentsHandler> Cannot start transaction from db")
		}
		defer tx.Rollback()

		environments, errEnv := environment.LoadEnvironments(tx, projectKey, true, getUser(ctx))
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
			return sdk.WrapError(err, "getEnvironmentsHandler> Cannot commit transaction from db")
		}

		return WriteJSON(w, environments, http.StatusOK)
	}
}

func (api *API) getEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]
		withUsage := FormBool(r, "withUsage")

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(ctx), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "getEnvironmentHandler> Cannot load environment %s for project %s from db", environmentName, projectKey)
		}
		env.Usage = &sdk.Usage{}

		if withUsage {
			wf, errW := workflow.LoadByEnvName(api.mustDB(ctx), projectKey, environmentName)
			if errW != nil {
				return sdk.WrapError(errW, "getEnvironmentHandler> Cannot load workflows linked to environment %s in project %s", environmentName, projectKey)
			}
			env.Usage.Workflows = wf

			apps, errApps := application.LoadByEnvName(api.mustDB(ctx), projectKey, environmentName)
			if errApps != nil {
				return sdk.WrapError(errApps, "getEnvironmentHandler> Cannot load applications linked to environment %s in project %s", environmentName, projectKey)
			}
			env.Usage.Applications = apps

			pips, errPips := pipeline.LoadByEnvName(api.mustDB(ctx), projectKey, environmentName)
			if errPips != nil {
				return sdk.WrapError(errApps, "getEnvironmentHandler> Cannot load pipelines linked to environment %s in project %s", environmentName, projectKey)
			}
			env.Usage.Pipelines = pips
		}

		env.Permission = permission.EnvironmentPermission(projectKey, env.Name, getUser(ctx))

		return WriteJSON(w, env, http.StatusOK)
	}
}

func (api *API) getEnvironmentUsageHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]
		usage, err := loadEnvironmentUsage(api.mustDB(ctx), projectKey, environmentName)
		if err != nil {
			return sdk.WrapError(err, "getEnvironmentHandler> Cannot load usage for environment %s in project %s", environmentName, projectKey)
		}

		return WriteJSON(w, usage, http.StatusOK)
	}
}

func loadEnvironmentUsage(db gorp.SqlExecutor, projectKey, envName string) (sdk.Usage, error) {
	usage := sdk.Usage{}

	wf, errW := workflow.LoadByEnvName(db, projectKey, envName)
	if errW != nil {
		return usage, sdk.WrapError(errW, "loadEnvironmentUsage> Cannot load workflows linked to environment %s in project %s", envName, projectKey)
	}
	usage.Workflows = wf

	apps, errApps := application.LoadByEnvName(db, projectKey, envName)
	if errApps != nil {
		return usage, sdk.WrapError(errApps, "loadEnvironmentUsage> Cannot load applications linked to environment %s in project %s", envName, projectKey)
	}
	usage.Applications = apps

	pips, errPips := pipeline.LoadByEnvName(db, projectKey, envName)
	if errPips != nil {
		return usage, sdk.WrapError(errApps, "loadEnvironmentUsage> Cannot load pipelines linked to environment %s in project %s", envName, projectKey)
	}
	usage.Pipelines = pips

	return usage, nil
}

func (api *API) addEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, errProj := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "addEnvironmentHandler> Cannot load %s", key)
		}

		var env sdk.Environment
		if err := UnmarshalBody(r, &env); err != nil {
			return err
		}
		env.ProjectID = proj.ID

		tx, errBegin := api.mustDB(ctx).Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addEnvironmentHandler> Cannot start transaction")
		}

		defer tx.Rollback()

		if err := environment.InsertEnvironment(tx, &env); err != nil {
			return sdk.WrapError(err, "addEnvironmentHandler> Cannot insert environment")
		}
		if err := group.LoadGroupByProject(tx, proj); err != nil {
			return sdk.WrapError(err, "addEnvironmentHandler> Cannot load group from project")
		}
		for _, g := range proj.ProjectGroups {
			if err := group.InsertGroupInEnvironment(tx, env.ID, g.Group.ID, g.Permission); err != nil {
				return sdk.WrapError(err, "addEnvironmentHandler> Cannot add group on environment")
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "addEnvironmentHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addEnvironmentHandler> Cannot commit transaction")
		}

		var errEnvs error
		proj.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(ctx), proj.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "addEnvironmentHandler> Cannot load all environments")
		}

		event.PublishEnvironmentAdd(key, env, getUser(ctx))

		return WriteJSON(w, proj, http.StatusOK)
	}
}

func (api *API) deleteEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]

		p, errProj := project.Load(api.mustDB(ctx), api.Cache, projectKey, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "deleteEnvironmentHandler> Cannot load project %s", projectKey)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(ctx), projectKey, environmentName)
		if errEnv != nil {
			if errEnv != sdk.ErrNoEnvironment {
				log.Warning("deleteEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, errEnv)
			}
			return errEnv
		}

		tx, errBegin := api.mustDB(ctx).Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "deleteEnvironmentHandler> Cannot begin transaction")
		}
		defer tx.Rollback()

		if err := environment.DeleteEnvironment(tx, env.ID); err != nil {
			return err
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "deleteEnvironmentHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteEnvironmentHandler> Cannot commit transaction")
		}

		event.PublishEnvironmentDelete(p.Key, *env, getUser(ctx))

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(ctx), p.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "deleteEnvironmentHandler> Cannot load environments")
		}
		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) updateEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(ctx), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "updateEnvironmentHandler> Cannot load environment %s", environmentName)
		}

		p, errProj := project.Load(api.mustDB(ctx), api.Cache, projectKey, getUser(ctx))
		if errProj != nil {
			return sdk.WrapError(errProj, "updateEnvironmentHandler> Cannot load project %s", projectKey)
		}

		var envPost sdk.Environment
		if err := UnmarshalBody(r, &envPost); err != nil {
			return err
		}

		oldEnv := env
		env.Name = envPost.Name

		tx, errBegin := api.mustDB(ctx).Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updateEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := environment.UpdateEnvironment(tx, env); err != nil {
			return sdk.WrapError(err, "updateEnvironmentHandler> Cannot update environment %s", environmentName)
		}

		if err := environment.UpdateLastModified(tx, api.Cache, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "updateEnvironmentHandler> Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "updateEnvironmentHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateEnvironmentHandler> Cannot commit transaction")
		}

		event.PublishEnvironmentUpdate(p.Key, *env, *oldEnv, getUser(ctx))

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(ctx), p.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "updateEnvironmentHandler> Cannot load environments")
		}

		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) cloneEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]
		cloneName := vars["cloneName"]

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(ctx), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "cloneEnvironmentHandler> Cannot load environment %s: %s", environmentName, errEnv)
		}

		p, errProj := project.Load(api.mustDB(ctx), api.Cache, projectKey, getUser(ctx))
		if errProj != nil {
			return sdk.WrapError(errProj, "cloneEnvironmentHandler> Cannot load project %s: %s", projectKey, errProj)
		}

		//Load all environments to check if there is another environment with the same name
		envs, err := environment.LoadEnvironments(api.mustDB(ctx), projectKey, false, getUser(ctx))
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
			Name:              cloneName,
			ProjectID:         p.ID,
			ProjectKey:        p.Key,
			Variable:          env.Variable,
			EnvironmentGroups: env.EnvironmentGroups,
			Permission:        env.Permission,
		}

		tx, err := api.mustDB(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to start a transaction")
		}

		defer tx.Rollback()

		//Insert environment
		if err := environment.InsertEnvironment(tx, &envPost); err != nil {
			return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to insert environment %s", envPost.Name)
		}

		//Insert variables
		for _, v := range envPost.Variable {
			if environment.InsertVariable(tx, envPost.ID, &v, getUser(ctx)); err != nil {
				return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to insert variable")
			}
		}

		//Insert environment
		for _, e := range envPost.EnvironmentGroups {
			if err := group.InsertGroupInEnvironment(tx, envPost.ID, e.Group.ID, e.Permission); err != nil {
				return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to insert group in environment")
			}
		}

		//Update the poroject
		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "cloneEnvironmentHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		//return the project with all environments
		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(ctx), p.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "cloneEnvironmentHandler> Cannot load environments: %s", errEnvs)
		}

		event.PublishEnvironmentAdd(p.Key, envPost, getUser(ctx))

		return WriteJSON(w, p, http.StatusOK)
	}
}
