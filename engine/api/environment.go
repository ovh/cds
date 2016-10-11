package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"io/ioutil"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getEnvironmentsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	environments, err := environment.LoadEnvironments(db, projectKey, true, c.User)
	if err != nil {
		log.Warning("getEnvironmentsHandler: Cannot load environments from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, environments, http.StatusOK)
}

func getEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	environment, err := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if err != nil {
		log.Warning("getEnvironmentHandler: Cannot load environment %s for project %s from db: %s\n", environmentName, projectKey, err)
		WriteError(w, r, err)
		return
	}

	environment.Permission = permission.EnvironmentPermission(environment.ID, c.User)

	WriteJSON(w, r, environment, http.StatusOK)
}

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
					err := keys.AddKeyPairToEnvironment(tx, env.ID, varEnv.Name)
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

	projectData, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("addEnvironmentHandler: Cannot load %s: %s\n", key, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var env sdk.Environment
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(data, &env)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	env.ProjectID = projectData.ID

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addEnvironmentHandler> Cannot start transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer tx.Rollback()

	err = environment.InsertEnvironment(tx, &env)
	if err != nil {
		log.Warning("addEnvironmentHandler> Cannot insert environment: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = group.LoadGroupByProject(tx, projectData)
	if err != nil {
		log.Warning("addEnvironmentHandler> Cannot load group from project: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, g := range projectData.ProjectGroups {
		err = group.InsertGroupInEnvironment(tx, env.ID, g.Group.ID, g.Permission)
		if err != nil {
			log.Warning("addEnvironmentHandler> Cannot add group on environment: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addEnvironmentHandler> Cannot commit transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	p, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("deleteEnvironmentHandler> Cannot load project %s: %s\n", projectKey, err)
		WriteError(w, r, err)
		return
	}

	env, err := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if err != nil {
		if err != sdk.ErrNoEnvironment {
			log.Warning("deleteEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, err)
		}
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteEnvironmentHandler> Cannot begin transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = environment.DeleteEnvironment(tx, env.ID)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	lastModified, err := project.UpdateProjectDB(tx, projectKey, p.Name)
	if err != nil {
		log.Warning("deleteEnvironmentHandler> Cannot update project last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}
	p.LastModified = lastModified.Unix()

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteEnvironmentHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	log.Notice("Environment %s deleted.\n", environmentName)
	WriteJSON(w, r, p, http.StatusOK)
}

func updateEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	environmentName := vars["permEnvironmentName"]

	env, err := environment.LoadEnvironmentByName(db, projectKey, environmentName)
	if err != nil {
		log.Warning("updateEnvironmentHandler> Cannot load environment %s: %s\n", environmentName, err)
		WriteError(w, r, err)
		return
	}

	p, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("updateEnvironmentHandler> Cannot load project %s: %s\n", projectKey, err)
		WriteError(w, r, err)
		return
	}

	var envPost sdk.Environment
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	err = json.Unmarshal(data, &envPost)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	env.Name = envPost.Name

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateEnvironmentHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = environment.CreateAudit(tx, projectKey, env, c.User)
	if err != nil {
		log.Warning("updateEnvironmentHandler> Cannot create audit for env %s: %s\n", env.Name, err)
		WriteError(w, r, err)
		return
	}

	err = environment.UpdateEnvironment(tx, env)
	if err != nil {
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

	lastModified, err := project.UpdateProjectDB(tx, projectKey, p.Name)
	if err != nil {
		log.Warning("updateEnvironmentHandler> Cannot update project last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}
	p.LastModified = lastModified.Unix()
	p.Environments = append(p.Environments, *env)

	err = tx.Commit()
	if err != nil {
		log.Warning("updateEnvironmentHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, p, http.StatusOK)
}
