package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getEnvironmentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	environments, errEnv := environment.LoadEnvironments(db, projectKey, true, c.User)
	if errEnv != nil {
		log.Warning("getEnvironmentsHandler: Cannot load environments from db: %s\n", errEnv)
		WriteError(w, r, errEnv)
		return
	}

	WriteJSON(w, r, environments, http.StatusOK)
}

func getEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	environment, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		log.Warning("getEnvironmentHandler: Cannot load environment %s for project %s from db: %s\n", environmentName, projectKey, errEnv)
		WriteError(w, r, errEnv)
		return
	}

	environment.Permission = permission.EnvironmentPermission(environment.ID, c.User)

	WriteJSON(w, r, environment, http.StatusOK)
}

// Deprecated
func updateEnvironmentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	projectData, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateEnvironmentsHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	var envs []sdk.Environment
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateEnvironmentsHandler: Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}
	err = json.Unmarshal(data, &envs)
	if err != nil {
		log.Warning("updateEnvironmentsHandler: Cannot unmarshal body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateEnvironmentsHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()
	for i := range envs {
		env := &envs[i]
		env.ProjectID = projectData.ID

		if env.ID != 0 {
			err = environment.CreateAudit(tx, key, env, c.User)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot create audit for env %s: %s\n", env.Name, err)
				WriteError(w, r, err)
				return
			}

			err = environment.UpdateEnvironment(tx, env)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot update environment: %s\n", err)
				WriteError(w, r, err)
				return
			}
		} else {
			err = environment.InsertEnvironment(tx, env)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot insert environment: %s\n", err)
				WriteError(w, r, err)
				return
			}
			env.Permission = permission.PermissionReadWriteExecute
		}

		if len(env.EnvironmentGroups) == 0 {
			log.Warning("updateEnvironmentsHandler> Cannot have an environment (%s) without group\n", env.Name)
			WriteError(w, r, sdk.ErrGroupNeedWrite)
			return
		}
		found := false
		for _, eg := range env.EnvironmentGroups {
			if eg.Permission == permission.PermissionReadWriteExecute {
				found = true
			}
		}
		if !found {
			log.Warning("updateEnvironmentsHandler> Cannot have an environment (%s) without group with write permission\n", env.Name)
			WriteError(w, r, sdk.ErrGroupNeedWrite)
			return
		}

		err = group.DeleteAllGroupFromEnvironment(tx, env.ID)
		if err != nil {
			log.Warning("updateEnvironmentsHandler> Cannot delete groups from environment %s for update: %s\n", env.Name, err)
			WriteError(w, r, err)
			return
		}
		for groupIndex := range env.EnvironmentGroups {
			groupEnv := &env.EnvironmentGroups[groupIndex]
			g, err := group.LoadGroup(tx, groupEnv.Group.Name)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot load group %s: %s\n", groupEnv.Group.Name, err)
				WriteError(w, r, err)
			}

			err = group.InsertGroupInEnvironment(tx, env.ID, g.ID, groupEnv.Permission)
			if err != nil {
				log.Warning("updateEnvironmentsHandler> Cannot insert group %s on environments %s: %s\n", groupEnv.Group.Name, env.Name, err)
				WriteError(w, r, err)
				return
			}

			// Update group ID
			groupEnv.Group.ID = g.ID
		}

		preload, err := environment.GetAllVariable(tx, key, env.Name, environment.WithClearPassword())
		if err != nil {
			log.Warning("updateEnvironmentsHandler> Cannot preload variable value: %s\n", err)
			WriteError(w, r, err)
			return
		}

		err = environment.DeleteAllVariable(tx, env.ID)
		if err != nil {
			log.Warning("updateEnvironmentsHandler> Cannot delete variables on environments for update: %s\n", err)
			WriteError(w, r, err)
			return
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
				err = environment.InsertVariable(tx, env.ID, varEnv)
				if err != nil {
					log.Warning("updateEnvironmentsHandler> Cannot insert variables on environments: %s\n", err)
					WriteError(w, r, err)
					return
				}

				// put placeholder because env.Variable will be in the handler response
				varEnv.Value = sdk.PasswordPlaceholder
				break
			case sdk.KeyVariable:
				if varEnv.Value == "" {
					err := environment.AddKeyPairToEnvironment(tx, env.ID, varEnv.Name)
					if err != nil {
						log.Warning("updateEnvironmentsHandler> cannot generate keypair: %s\n", err)
						WriteError(w, r, err)
						return
					}
				} else if varEnv.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == varEnv.ID {
							varEnv.Value = p.Value
						}
					}
					err = environment.InsertVariable(tx, env.ID, varEnv)
					if err != nil {
						log.Warning("updateEnvironments: Cannot insert variable %s:  %s\n", varEnv.Name, err)
						WriteError(w, r, err)
						return
					}
				}
				// put placeholder because env.Variable will be in the handler response
				varEnv.Value = sdk.PasswordPlaceholder
				break
			default:
				err = environment.InsertVariable(tx, env.ID, varEnv)
				if err != nil {
					log.Warning("updateEnvironmentsHandler> Cannot insert variables on environments: %s\n", err)
					WriteError(w, r, err)
					return
				}
			}

		}

	}

	lastModified, err := project.UpdateProjectDB(tx, projectData.Key, projectData.Name)
	if err != nil {
		log.Warning("updateEnvironmentsHandler> Cannot update project last modified date: %s\n", err)
		WriteError(w, r, err)
	}
	projectData.LastModified = lastModified.Unix()
	projectData.Environments = envs

	err = tx.Commit()
	if err != nil {
		log.Warning("updateEnvironmentsHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckProjectPipelines(db, projectData)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, projectData, http.StatusOK)
}

func addEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	projectData, errProj := project.LoadProject(db, key, c.User)
	if errProj != nil {
		log.Warning("addEnvironmentHandler: Cannot load %s: %s\n", key, errProj)
		WriteError(w, r, errProj)
		return
	}

	var env sdk.Environment
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	if err := json.Unmarshal(data, &env); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	env.ProjectID = projectData.ID

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("addEnvironmentHandler> Cannot start transaction: %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}

	defer tx.Rollback()

	if err := environment.InsertEnvironment(tx, &env); err != nil {
		log.Warning("addEnvironmentHandler> Cannot insert environment: %s\n", err)
		WriteError(w, r, err)
		return
	}
	if err := group.LoadGroupByProject(tx, projectData); err != nil {
		log.Warning("addEnvironmentHandler> Cannot load group from project: %s\n", err)
		WriteError(w, r, err)
		return
	}
	for _, g := range projectData.ProjectGroups {
		if err := group.InsertGroupInEnvironment(tx, env.ID, g.Group.ID, g.Permission); err != nil {
			log.Warning("addEnvironmentHandler> Cannot add group on environment: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	lastModified, errDate := project.UpdateProjectDB(tx, projectData.Key, projectData.Name)
	if errDate != nil {
		log.Warning("addEnvironmentHandler> Cannot update project last modified date: %s\n", errDate)
		WriteError(w, r, errDate)
		return
	}
	projectData.LastModified = lastModified.Unix()

	if err := tx.Commit(); err != nil {
		log.Warning("addEnvironmentHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	var errEnvs error
	projectData.Environments, errEnvs = environment.LoadEnvironments(db, projectData.Key, true, c.User)
	if errEnvs != nil {
		log.Warning("addEnvironmentHandler> Cannot load all environments: %s\n", errEnvs)
		WriteError(w, r, errEnvs)
		return
	}
	WriteJSON(w, r, projectData, http.StatusOK)
}

func deleteEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	p, errProj := project.LoadProject(db, projectKey, c.User)
	if errProj != nil {
		log.Warning("deleteEnvironmentHandler> Cannot load project %s: %s\n", projectKey, errProj)
		WriteError(w, r, errProj)
		return
	}

	env, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		if errEnv != sdk.ErrNoEnvironment {
			log.Warning("deleteEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, errEnv)
		}
		WriteError(w, r, errEnv)
		return
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("deleteEnvironmentHandler> Cannot begin transaction: %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	if err := environment.DeleteEnvironment(tx, env.ID); err != nil {
		WriteError(w, r, err)
		return
	}

	lastModified, errDate := project.UpdateProjectDB(tx, projectKey, p.Name)
	if errDate != nil {
		log.Warning("deleteEnvironmentHandler> Cannot update project last modified date: %s\n", errDate)
		WriteError(w, r, errDate)
		return
	}
	p.LastModified = lastModified.Unix()

	if err := tx.Commit(); err != nil {
		log.Warning("deleteEnvironmentHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	log.Notice("Environment %s deleted.\n", environmentName)
	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, p.Key, true, c.User)
	if errEnvs != nil {
		log.Warning("deleteEnvironmentHandler> Cannot load environments: %s\n", errEnvs)
		WriteError(w, r, errEnvs)
		return
	}
	WriteJSON(w, r, p, http.StatusOK)
}

func updateEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	env, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		log.Warning("updateEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, errEnv)
		WriteError(w, r, errEnv)
		return
	}

	p, errProj := project.LoadProject(db, projectKey, c.User)
	if errProj != nil {
		log.Warning("updateEnvironmentHandler> Cannot load project %s: %s\n", projectKey, errProj)
		WriteError(w, r, errProj)
		return
	}

	var envPost sdk.Environment
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if err := json.Unmarshal(data, &envPost); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	env.Name = envPost.Name

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("updateEnvironmentHandler> Cannot start transaction: %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	if err := environment.CreateAudit(tx, projectKey, env, c.User); err != nil {
		log.Warning("updateEnvironmentHandler> Cannot create audit for env %s: %s\n", env.Name, err)
		WriteError(w, r, err)
		return
	}

	if err := environment.UpdateEnvironment(tx, env); err != nil {
		log.Warning("updateEnvironmentHandler> Cannot update environment %s: %s\n", environmentName, err)
		WriteError(w, r, err)
		return
	}

	if len(envPost.Variable) > 0 {
		preload, err := environment.GetAllVariable(tx, projectKey, env.Name, environment.WithClearPassword())
		if err != nil {
			log.Warning("updateEnvironmentHandler> Cannot preload variable value: %s\n", err)
			WriteError(w, r, err)
			return
		}

		err = environment.DeleteAllVariable(tx, env.ID)
		if err != nil {
			log.Warning("updateEnvironmentHandler> Cannot delete variables on environments for update: %s\n", err)
			WriteError(w, r, err)
			return
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
			err = environment.InsertVariable(tx, env.ID, varEnv)
			if err != nil {
				log.Warning("updateEnvironmentHandler> Cannot insert variables on environments: %s\n", err)
				WriteError(w, r, err)
				return
			}
		}
	}

	lastModified, errDate := project.UpdateProjectDB(tx, projectKey, p.Name)
	if errDate != nil {
		log.Warning("updateEnvironmentHandler> Cannot update project last modified date: %s\n", errDate)
		WriteError(w, r, errDate)
		return
	}
	p.LastModified = lastModified.Unix()

	if err := tx.Commit(); err != nil {
		log.Warning("updateEnvironmentHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, p.Key, true, c.User)
	if errEnvs != nil {
		log.Warning("updateEnvironmentHandler> Cannot load environments: %s\n", errEnvs)
		WriteError(w, r, errEnvs)
		return
	}

	WriteJSON(w, r, p, http.StatusOK)
}

func cloneEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	env, errEnv := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if errEnv != nil {
		log.Warning("cloneEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, errEnv)
		WriteError(w, r, errEnv)
		return
	}

	p, errProj := project.LoadProject(db, projectKey, c.User)
	if errProj != nil {
		log.Warning("cloneEnvironmentHandler> Cannot load project %s: %s\n", projectKey, errProj)
		WriteError(w, r, errProj)
		return
	}

	var envPost sdk.Environment
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if err := json.Unmarshal(data, &envPost); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Check if the new environment has a name
	if envPost.Name == "" {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Load all environments to check if there is another environment with the same name
	envs, err := environment.LoadEnvironments(db, projectKey, false, c.User)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	for _, e := range envs {
		if e.Name == envPost.Name {
			WriteError(w, r, sdk.ErrConflict)
			return
		}
	}

	//Set all the data of the environment we want to clone
	envPost.ProjectID = p.ID
	envPost.ProjectKey = p.Key
	envPost.Variable = env.Variable
	envPost.EnvironmentGroups = env.EnvironmentGroups
	envPost.Permission = env.Permission

	tx, err := db.Begin()
	if err != nil {
		log.Warning("cloneEnvironmentHandler> Unable to start a transaction: %s", err)
		WriteError(w, r, err)
		return
	}

	defer tx.Rollback()

	//Insert environment
	if err := environment.InsertEnvironment(tx, &envPost); err != nil {
		log.Warning("cloneEnvironmentHandler> Unable to insert environment %s: %s", envPost.Name, err)
		WriteError(w, r, err)
		return
	}

	//Insert variables
	for _, v := range envPost.Variable {
		if environment.InsertVariable(tx, envPost.ID, &v); err != nil {
			log.Warning("cloneEnvironmentHandler> Unable to insert variable: %s", err)
			WriteError(w, r, err)
			return
		}
	}

	//Insert environment
	for _, e := range envPost.EnvironmentGroups {
		if err := group.InsertGroupInEnvironment(tx, envPost.ID, e.Group.ID, e.Permission); err != nil {
			log.Warning("cloneEnvironmentHandler> Unable to insert group in environment: %s", err)
			WriteError(w, r, err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, envPost, http.StatusCreated)
}
