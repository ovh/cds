package main

import (
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getEnvironmentsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	environments, errEnv := environment.LoadEnvironments(db, projectKey, true, c.User)
	if errEnv != nil {
		log.Warning("getEnvironmentsHandler: Cannot load environments from db: %s\n", errEnv)
		return errEnv
	}

	return WriteJSON(w, r, environments, http.StatusOK)
}

func getEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	environment, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		log.Warning("getEnvironmentHandler: Cannot load environment %s for project %s from db: %s\n", environmentName, projectKey, errEnv)
		return errEnv
	}

	environment.Permission = permission.EnvironmentPermission(environment.ID, c.User)

	return WriteJSON(w, r, environment, http.StatusOK)
}

// Deprecated
func updateEnvironmentsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	proj, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		log.Warning("updateEnvironmentsHandler: Cannot load %s: %s\n", key, err)
		return err
	}

	var envs []sdk.Environment
	if err := UnmarshalBody(r, &envs); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateEnvironmentsHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()
	for i := range envs {
		env := &envs[i]
		env.ProjectID = proj.ID

		if env.ID != 0 {
			err = environment.UpdateEnvironment(tx, env)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot update environment: %s\n", err)
				return err
			}
		} else {
			err = environment.InsertEnvironment(tx, env)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot insert environment: %s\n", err)
				return err
			}
			env.Permission = permission.PermissionReadWriteExecute
		}

		if len(env.EnvironmentGroups) == 0 {
			log.Warning("updateEnvironmentsHandler> Cannot have an environment (%s) without group\n", env.Name)
			return sdk.ErrGroupNeedWrite
		}
		found := false
		for _, eg := range env.EnvironmentGroups {
			if eg.Permission == permission.PermissionReadWriteExecute {
				found = true
			}
		}
		if !found {
			log.Warning("updateEnvironmentsHandler> Cannot have an environment (%s) without group with write permission\n", env.Name)
			return sdk.ErrGroupNeedWrite
		}

		if err := group.DeleteAllGroupFromEnvironment(tx, env.ID); err != nil {
			log.Warning("updateEnvironmentsHandler> Cannot delete groups from environment %s for update: %s\n", env.Name, err)
			return err
		}
		for groupIndex := range env.EnvironmentGroups {
			groupEnv := &env.EnvironmentGroups[groupIndex]
			g, err := group.LoadGroup(tx, groupEnv.Group.Name)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot load group %s: %s\n", groupEnv.Group.Name, err)
			}

			err = group.InsertGroupInEnvironment(tx, env.ID, g.ID, groupEnv.Permission)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot insert group %s on environments %s: %s\n", groupEnv.Group.Name, env.Name, err)
				return err
			}

			// Update group ID
			groupEnv.Group.ID = g.ID
		}

		preload, err := environment.GetAllVariable(tx, key, env.Name, environment.WithClearPassword())
		if err != nil {
			log.Warning("updateEnvironmentsHandler> Cannot preload variable value: %s\n", err)
			return err
		}

		err = environment.DeleteAllVariable(tx, env.ID)
		if err != nil {
			log.Warning("updateEnvironmentsHandler> Cannot delete variables on environments for update: %s\n", err)
			return err
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
					log.Warning("updateEnvironmentsHandler> %s (%s)\n", errMsg, c.User.Username)
					return sdk.NewError(sdk.ErrInvalidSecretValue, fmt.Errorf("%s", errMsg))
				}
				err = environment.InsertVariable(tx, env.ID, varEnv, c.User)
				if err != nil {
					log.Warning("updateEnvironmentsHandler> Cannot insert variables on environments: %s\n", err)
					return err
				}

				// put placeholder because env.Variable will be in the handler response
				varEnv.Value = sdk.PasswordPlaceholder
				break
			case sdk.KeyVariable:
				if varEnv.Value == "" {
					err := environment.AddKeyPairToEnvironment(tx, env.ID, varEnv.Name, c.User)
					if err != nil {
						log.Warning("updateEnvironmentsHandler> cannot generate keypair: %s\n", err)
						return err
					}
				} else if varEnv.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == varEnv.ID {
							varEnv.Value = p.Value
						}
					}
					err = environment.InsertVariable(tx, env.ID, varEnv, c.User)
					if err != nil {
						log.Warning("updateEnvironments: Cannot insert variable %s:  %s\n", varEnv.Name, err)
						return err
					}
				}
				// put placeholder because env.Variable will be in the handler response
				varEnv.Value = sdk.PasswordPlaceholder
				break
			default:
				err = environment.InsertVariable(tx, env.ID, varEnv, c.User)
				if err != nil {
					log.Warning("updateEnvironmentsHandler> Cannot insert variables on environments: %s\n", err)
					return err
				}
			}
		}
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		log.Warning("updateEnvironmentsHandler> Cannot update last modified date: %s\n", err)
		return err
	}
	proj.Environments = envs

	if err := tx.Commit(); err != nil {
		log.Warning("updateEnvironmentsHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	if err := sanity.CheckProjectPipelines(db, proj); err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot check warnings: %s\n", err)
		return err
	}

	return WriteJSON(w, r, proj, http.StatusOK)
}

func addEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	proj, errProj := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errProj != nil {
		log.Warning("addEnvironmentHandler: Cannot load %s: %s\n", key, errProj)
		return errProj
	}

	var env sdk.Environment
	if err := UnmarshalBody(r, &env); err != nil {
		return err
	}
	env.ProjectID = proj.ID

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("addEnvironmentHandler> Cannot start transaction: %s\n", errBegin)
		return errBegin
	}

	defer tx.Rollback()

	if err := environment.InsertEnvironment(tx, &env); err != nil {
		log.Warning("addEnvironmentHandler> Cannot insert environment: %s\n", err)
		return err
	}
	if err := group.LoadGroupByProject(tx, proj); err != nil {
		log.Warning("addEnvironmentHandler> Cannot load group from project: %s\n", err)
		return err
	}
	for _, g := range proj.ProjectGroups {
		if err := group.InsertGroupInEnvironment(tx, env.ID, g.Group.ID, g.Permission); err != nil {
			log.Warning("addEnvironmentHandler> Cannot add group on environment: %s\n", err)
			return err
		}
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		log.Warning("addEnvironmentHandler> Cannot update last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addEnvironmentHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	var errEnvs error
	proj.Environments, errEnvs = environment.LoadEnvironments(db, proj.Key, true, c.User)
	if errEnvs != nil {
		log.Warning("addEnvironmentHandler> Cannot load all environments: %s\n", errEnvs)
		return errEnvs
	}

	return WriteJSON(w, r, proj, http.StatusOK)
}

func deleteEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	p, errProj := project.Load(db, projectKey, c.User, project.LoadOptions.Default)
	if errProj != nil {
		log.Warning("deleteEnvironmentHandler> Cannot load project %s: %s\n", projectKey, errProj)
		return errProj
	}

	env, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		if errEnv != sdk.ErrNoEnvironment {
			log.Warning("deleteEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, errEnv)
		}
		return errEnv
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("deleteEnvironmentHandler> Cannot begin transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if err := environment.DeleteEnvironment(tx, env.ID); err != nil {
		return err
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		log.Warning("deleteEnvironmentHandler> Cannot update last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteEnvironmentHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	log.Info("Environment %s deleted.\n", environmentName)
	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, p.Key, true, c.User)
	if errEnvs != nil {
		log.Warning("deleteEnvironmentHandler> Cannot load environments: %s\n", errEnvs)
		return errEnvs
	}
	return WriteJSON(w, r, p, http.StatusOK)
}

func updateEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	env, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		log.Warning("updateEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, errEnv)
		return errEnv
	}

	p, errProj := project.Load(db, projectKey, c.User, project.LoadOptions.Default)
	if errProj != nil {
		log.Warning("updateEnvironmentHandler> Cannot load project %s: %s\n", projectKey, errProj)
		return errProj
	}

	var envPost sdk.Environment
	if err := UnmarshalBody(r, &envPost); err != nil {
		return err
	}

	env.Name = envPost.Name

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("updateEnvironmentHandler> Cannot start transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if err := environment.UpdateEnvironment(tx, env); err != nil {
		log.Warning("updateEnvironmentHandler> Cannot update environment %s: %s\n", environmentName, err)
		return err
	}

	if len(envPost.Variable) > 0 {
		preload, err := environment.GetAllVariable(tx, projectKey, env.Name, environment.WithClearPassword())
		if err != nil {
			log.Warning("updateEnvironmentHandler> Cannot preload variable value: %s\n", err)
			return err
		}

		err = environment.DeleteAllVariable(tx, env.ID)
		if err != nil {
			log.Warning("updateEnvironmentHandler> Cannot delete variables on environments for update: %s\n", err)
			return err
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
			err = environment.InsertVariable(tx, env.ID, varEnv, c.User)
			if err != nil {
				log.Warning("updateEnvironmentHandler> Cannot insert variables on environments: %s\n", err)
				return err
			}
		}
	}

	if err := environment.UpdateLastModified(tx, c.User, env); err != nil {
		return sdk.WrapError(err, "updateEnvironmentHandler> Cannot update environment last modified date")
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		log.Warning("updateEnvironmentHandler> Cannot update last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateEnvironmentHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, p.Key, true, c.User)
	if errEnvs != nil {
		log.Warning("updateEnvironmentHandler> Cannot load environments: %s\n", errEnvs)
		return errEnvs
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func cloneEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]
	cloneName := vars["cloneName"]

	env, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		return sdk.WrapError(errEnv, "cloneEnvironmentHandler> Cannot load environment %s: %s", environmentName, errEnv)
	}

	p, errProj := project.Load(db, projectKey, c.User)
	if errProj != nil {
		return sdk.WrapError(errProj, "cloneEnvironmentHandler> Cannot load project %s: %s", projectKey, errProj)
	}

	//Load all environments to check if there is another environment with the same name
	envs, err := environment.LoadEnvironments(db, projectKey, false, c.User)
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

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to start a transaction: %s", err)
	}

	defer tx.Rollback()

	//Insert environment
	if err := environment.InsertEnvironment(tx, &envPost); err != nil {
		return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to insert environment %s: %s", envPost.Name, err)
	}

	//Insert variables
	for _, v := range envPost.Variable {
		if environment.InsertVariable(tx, envPost.ID, &v, c.User); err != nil {
			return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to insert variable: %s", err)
		}
	}

	//Insert environment
	for _, e := range envPost.EnvironmentGroups {
		if err := group.InsertGroupInEnvironment(tx, envPost.ID, e.Group.ID, e.Permission); err != nil {
			return sdk.WrapError(err, "cloneEnvironmentHandler> Unable to insert group in environment: %s", err)
		}
	}

	//Update the poroject
	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "cloneEnvironmentHandler> Cannot update last modified date: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	//return the project with all environments
	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, p.Key, true, c.User)
	if errEnvs != nil {
		return sdk.WrapError(errEnvs, "cloneEnvironmentHandler> Cannot load environments: %s", errEnvs)
	}

	return WriteJSON(w, r, p, http.StatusOK)
}
