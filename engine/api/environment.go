package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getEnvironmentsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		withUsage := FormBool(r, "withUsage")

		tx, errTx := api.mustDB().Begin()
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

		return WriteJSON(w, r, environments, http.StatusOK)
	}
}

func (api *API) getEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]
		withWorkflows := FormBool(r, "withWorkflows")

		tx, errTx := api.mustDB().Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "getEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		environment, errEnv := environment.LoadEnvironmentByName(tx, projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "getEnvironmentHandler> Cannot load environment %s for project %s from db", environmentName, projectKey)
		}

		if withWorkflows {
			environment.Usage = &sdk.Usage{}
			wf, errW := workflow.LoadByEnvName(tx, projectKey, environmentName)
			if errW != nil {
				return sdk.WrapError(errW, "getEnvironmentHandler> Cannot load workflows linked to environments %s in project %s", environmentName, projectKey)
			}
			environment.Usage.Workflows = wf
		}

		environment.Permission = permission.EnvironmentPermission(environment.ID, getUser(ctx))

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "getEnvironmentHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, environment, http.StatusOK)
	}
}

func (api *API) getEnvironmentUsageHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]
		usage := sdk.Usage{}

		wf, errW := workflow.LoadByEnvName(api.mustDB(), projectKey, environmentName)
		if errW != nil {
			return sdk.WrapError(errW, "getEnvironmentHandler> Cannot load workflows linked to environments %s in project %s", environmentName, projectKey)
		}
		usage.Workflows = wf

		return WriteJSON(w, r, usage, http.StatusOK)
	}
}

// Deprecated
func (api *API) updateEnvironmentsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "updateEnvironmentsHandler: Cannot load %s", key)
		}

		var envs []sdk.Environment
		if err := UnmarshalBody(r, &envs); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot start transaction")
		}
		defer tx.Rollback()
		for i := range envs {
			env := &envs[i]
			env.ProjectID = proj.ID

			if env.ID != 0 {
				err = environment.UpdateEnvironment(tx, env)
				if err != nil {
					return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot update environment")
				}
			} else {
				err = environment.InsertEnvironment(tx, env)
				if err != nil {
					return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot insert environment")
				}
				env.Permission = permission.PermissionReadWriteExecute
			}

			if len(env.EnvironmentGroups) == 0 {
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateEnvironmentsHandler> Cannot have an environment (%s) without group", env.Name)
			}
			found := false
			for _, eg := range env.EnvironmentGroups {
				if eg.Permission == permission.PermissionReadWriteExecute {
					found = true
				}
			}
			if !found {
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateEnvironmentsHandler> Cannot have an environment (%s) without group with write permission", env.Name)
			}

			if err := group.DeleteAllGroupFromEnvironment(tx, env.ID); err != nil {
				return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot delete groups from environment %s for update", env.Name)
			}
			for groupIndex := range env.EnvironmentGroups {
				groupEnv := &env.EnvironmentGroups[groupIndex]
				g, err := group.LoadGroup(tx, groupEnv.Group.Name)
				if err != nil {
					log.Warning("updateEnvironmentsHandler> Cannot load group %s: %s\n", groupEnv.Group.Name, err)
				}

				err = group.InsertGroupInEnvironment(tx, env.ID, g.ID, groupEnv.Permission)
				if err != nil {
					return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot insert group %s on environments %s", groupEnv.Group.Name, env.Name)
				}

				// Update group ID
				groupEnv.Group.ID = g.ID
			}

			preload, err := environment.GetAllVariable(tx, key, env.Name, environment.WithClearPassword())
			if err != nil {
				return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot preload variable value")
			}

			err = environment.DeleteAllVariable(tx, env.ID)
			if err != nil {
				return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot delete variables on environments for update")
			}

			for varIndex := range env.Variable {
				varEnv := &env.Variable[varIndex]
				switch varEnv.Type {
				case sdk.SecretVariable:
					found := false
					if sdk.NeedPlaceholder(varEnv.Type) && varEnv.Value == sdk.PasswordPlaceholder {
						for _, p := range preload {
							if p.ID == varEnv.ID {
								found = true
								varEnv.Value = p.Value
								break
							}
						}
						if !found {
							log.Warning("UpdateEnvironments> Previous value of %s/%s.%s not found, set to empty\n", key, env.Name, varEnv.Name)
							varEnv.Value = ""
						}
					}
					if varEnv.Value == "" {
						errMsg := fmt.Sprintf("Variable %s on environment %s on project %s cannot be empty", varEnv.Name, env.Name, key)
						log.Warning("updateEnvironmentsHandler> %s (%s)\n", errMsg, getUser(ctx).Username)
						return sdk.NewError(sdk.ErrInvalidSecretValue, fmt.Errorf("%s", errMsg))
					}
					err = environment.InsertVariable(tx, env.ID, varEnv, getUser(ctx))
					if err != nil {
						return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot insert variables on environments")
					}

					// put placeholder because env.Variable will be in the handler response
					varEnv.Value = sdk.PasswordPlaceholder
					break
				case sdk.KeyVariable:
					if varEnv.Value == "" {
						err := environment.AddKeyPairToEnvironment(tx, env.ID, varEnv.Name, getUser(ctx))
						if err != nil {
							return sdk.WrapError(err, "updateEnvironmentsHandler> cannot generate keypair")
						}
					} else if varEnv.Value == sdk.PasswordPlaceholder {
						for _, p := range preload {
							if p.ID == varEnv.ID {
								varEnv.Value = p.Value
							}
						}
						err = environment.InsertVariable(tx, env.ID, varEnv, getUser(ctx))
						if err != nil {
							return sdk.WrapError(err, "updateEnvironments> Cannot insert variable %s", varEnv.Name)
						}
					}
					// put placeholder because env.Variable will be in the handler response
					varEnv.Value = sdk.PasswordPlaceholder
					break
				default:
					err = environment.InsertVariable(tx, env.ID, varEnv, getUser(ctx))
					if err != nil {
						return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot insert variables on environments")
					}
				}
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot update last modified date")
		}
		proj.Environments = envs

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateEnvironmentsHandler> Cannot commit transaction")
		}

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, proj); err != nil {
				log.Warning("updateVariablesInApplicationHandler> Cannot check warnings: %s", err)
			}
		}()

		return WriteJSON(w, r, proj, http.StatusOK)
	}
}

func (api *API) addEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "addEnvironmentHandler> Cannot load %s", key)
		}

		var env sdk.Environment
		if err := UnmarshalBody(r, &env); err != nil {
			return err
		}
		env.ProjectID = proj.ID

		tx, errBegin := api.mustDB().Begin()
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
		proj.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), proj.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "addEnvironmentHandler> Cannot load all environments")
		}

		return WriteJSON(w, r, proj, http.StatusOK)
	}
}

func (api *API) deleteEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]

		p, errProj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "deleteEnvironmentHandler> Cannot load project %s", projectKey)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			if errEnv != sdk.ErrNoEnvironment {
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

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "deleteEnvironmentHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteEnvironmentHandler> Cannot commit transaction")
		}

		log.Info("Environment %s deleted.\n", environmentName)
		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "deleteEnvironmentHandler> Cannot load environments")
		}
		return WriteJSON(w, r, p, http.StatusOK)
	}
}

func (api *API) updateEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "updateEnvironmentHandler> Cannot load environment %s", environmentName)
		}

		p, errProj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "updateEnvironmentHandler> Cannot load project %s", projectKey)
		}

		var envPost sdk.Environment
		if err := UnmarshalBody(r, &envPost); err != nil {
			return err
		}

		env.Name = envPost.Name

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updateEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := environment.UpdateEnvironment(tx, env); err != nil {
			return sdk.WrapError(err, "updateEnvironmentHandler> Cannot update environment %s", environmentName)
		}

		if len(envPost.Variable) > 0 {
			preload, err := environment.GetAllVariable(tx, projectKey, env.Name, environment.WithClearPassword())
			if err != nil {
				return sdk.WrapError(err, "updateEnvironmentHandler> Cannot preload variable value")
			}

			err = environment.DeleteAllVariable(tx, env.ID)
			if err != nil {
				return sdk.WrapError(err, "updateEnvironmentHandler> Cannot delete variables on environments for update")
			}

			for varIndex := range envPost.Variable {
				varEnv := &envPost.Variable[varIndex]
				found := false
				if sdk.NeedPlaceholder(varEnv.Type) && varEnv.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.Name == varEnv.Name {
							found = true
							varEnv.Value = p.Value
							break
						}
					}
					if !found {
						log.Warning("updateEnvironmentHandler> Previous value of %s/%s.%s not found, set to empty\n", projectKey, env.Name, varEnv.Name)
						varEnv.Value = ""
					}
				}
				err = environment.InsertVariable(tx, env.ID, varEnv, getUser(ctx))
				if err != nil {
					return sdk.WrapError(err, "updateEnvironmentHandler> Cannot insert variables on environments")
				}
			}
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

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "updateEnvironmentHandler> Cannot load environments")
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}

func (api *API) cloneEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		environmentName := vars["permEnvironmentName"]
		cloneName := vars["cloneName"]

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), projectKey, environmentName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "cloneEnvironmentHandler> Cannot load environment %s: %s", environmentName, errEnv)
		}

		p, errProj := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx))
		if errProj != nil {
			return sdk.WrapError(errProj, "cloneEnvironmentHandler> Cannot load project %s: %s", projectKey, errProj)
		}

		//Load all environments to check if there is another environment with the same name
		envs, err := environment.LoadEnvironments(api.mustDB(), projectKey, false, getUser(ctx))
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

		tx, err := api.mustDB().Begin()
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
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "cloneEnvironmentHandler> Cannot load environments: %s", errEnvs)
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}
