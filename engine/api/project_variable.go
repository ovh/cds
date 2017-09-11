package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getVariablesAuditInProjectnHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]

		audits, err := project.GetVariableAudit(api.mustDB(), key)
		if err != nil {
			log.Warning("getVariablesAuditInProjectnHandler: Cannot get variable audit for project %s: %s\n", key, err)
			return err

		}
		return WriteJSON(w, r, audits, http.StatusOK)
	}
}

// Deprecated
func (api *API) restoreProjectVariableAuditHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		auditIDString := vars["auditID"]

		auditID, err := strconv.ParseInt(auditIDString, 10, 64)
		if err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot parse auditID %s: %s\n", auditIDString, err)
			return sdk.ErrInvalidID

		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot load %s: %s\n", key, err)
			return err

		}

		variables, err := project.GetAudit(api.mustDB(), key, auditID)
		if err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot get variable audit for project %s: %s\n", key, err)
			return err

		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot start transaction : %s\n", err)
			return sdk.ErrUnknownError

		}
		defer tx.Rollback()

		if err := project.DeleteAllVariable(tx, p.ID); err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot delete variables for project %s:  %s\n", key, err)
			return sdk.ErrUnknownError

		}

		for _, v := range variables {
			if sdk.NeedPlaceholder(v.Type) {
				value, err := secret.Decrypt([]byte(v.Value))
				if err != nil {
					log.Warning("restoreProjectVariableAuditHandler: Cannot decrypt variable %s for project %s:  %s\n", v.Name, key, err)
					return err

				}
				v.Value = string(value)
			}
			if err := project.InsertVariable(tx, p, &v, getUser(ctx)); err != nil {
				log.Warning("restoreProjectVariableAuditHandler: Cannot insert variable %s for project %s:  %s\n", v.Name, key, err)
				return err
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot update last modified:  %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot commit transaction:  %s\n", err)
			return err
		}

		if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot check warnings: %s\n", err)
			return err
		}

		return nil
	}
}

func (api *API) getVariablesInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithVariables)
		if err != nil {
			log.Warning("deleteVariableFromProject: Cannot load %s: %s\n", key, err)
			return sdk.ErrNotFound
		}

		return WriteJSON(w, r, p.Variable, http.StatusOK)
	}
}

func (api *API) deleteVariableFromProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("deleteVariableFromProject: Cannot load %s: %s\n", key, err)
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("deleteVariableFromProject: Cannot start transaction: %s\n", err)
			return err
		}
		defer tx.Rollback()

		varToDelete, errV := project.GetVariableInProject(api.mustDB(), p.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromProject> Cannot load variable %s", varName)
		}

		if err := project.DeleteVariable(tx, p, varToDelete, getUser(ctx)); err != nil {
			log.Warning("deleteVariableFromProject: Cannot delete %s: %s\n", varName, err)
			return err
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			log.Warning("deleteVariableFromProject: Cannot update last modified date: %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("deleteVariableFromProject: Cannot commit transaction: %s\n", err)
			return err
		}

		return WriteJSON(w, r, nil, http.StatusOK)
	}
}

//DEPRECATED
func (api *API) updateVariablesInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		var projectVars []sdk.Variable
		if err := UnmarshalBody(r, &projectVars); err != nil {
			return err
		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot load %s: %s\n", key, err)
			return sdk.ErrNotFound

		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot start transaction: %s\n", err)
			return sdk.ErrNotFound

		}
		defer tx.Rollback()

		// Preload values, if one password variable has a password placeholder, we can't just insert
		// the placeholder !
		preload, err := project.GetAllVariableInProject(tx, p.ID, project.WithClearPassword())
		if err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot preload variables values: %s\n", err)
			return err

		}

		if err := project.DeleteAllVariable(tx, p.ID); err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot delete all variables for project %s: %s\n", p.Key, err)
			return sdk.ErrNotFound

		}

		for _, v := range projectVars {
			switch v.Type {
			case sdk.SecretVariable:
				if sdk.NeedPlaceholder(v.Type) && v.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == v.ID {
							v.Value = p.Value
						}
					}
				}
				err = project.InsertVariable(tx, p, &v, getUser(ctx))
				if err != nil {
					log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
					return err

				}
				break
			// In case of a key variable, if empty, generate a pair and add them as variable
			case sdk.KeyVariable:
				if v.Value == "" {
					err := project.AddKeyPair(tx, p, v.Name, getUser(ctx))
					if err != nil {
						log.Warning("updateVariablesInProjectHandler> cannot generate keypair: %s\n", err)
						return err

					}
				} else if v.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == v.ID {
							v.Value = p.Value
						}
					}
					err = project.InsertVariable(tx, p, &v, getUser(ctx))
					if err != nil {
						log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
						return err

					}
				}
				break
			default:
				err = project.InsertVariable(tx, p, &v, getUser(ctx))
				if err != nil {
					log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
					return err

				}
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot update last modified:  %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot commit transaction: %s\n", err)
			return sdk.ErrNotFound

		}

		if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
			log.Warning("updateVariablesInApplicationHandler: Cannot check warnings: %s\n", err)
			return err

		}

		return nil
	}
}

func (api *API) updateVariableInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName {
			return sdk.ErrWrongRequest

		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("updateVariableInProject: Cannot load %s: %s\n", key, err)
			return err

		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("updateVariableInProject: cannot start transaction: %s\n", err)
			return err

		}
		defer tx.Rollback()

		if err := project.UpdateVariable(tx, p, &newVar, getUser(ctx)); err != nil {
			log.Warning("updateVariableInProject: Cannot update variable %s in project %s:  %s\n", varName, p.Name, err)
			return err
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			log.Warning("updateVariableInProject: Cannot update last modified date: %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("updateVariableInProject: cannot commit transaction: %s\n", err)
			return err

		}

		if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
			log.Warning("updateVariableInProject: Cannot check warnings: %s\n", err)
			return err

		}
		return WriteJSON(w, r, newVar, http.StatusOK)
	}
}

//DEPRECATED
func (api *API) getVariableInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if err != nil {
			log.Warning("getVariableInProjectHandler: Cannot load project %s: %s\n", key, err)
			return err

		}

		v, err := project.GetVariableInProject(api.mustDB(), proj.ID, varName)
		if err != nil {
			log.Warning("getVariableInProjectHandler: Cannot get variable %s in project %s: %s\n", varName, key, err)
			return err

		}

		return WriteJSON(w, r, v, http.StatusOK)
	}
}

func (api *API) addVariableInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName {
			return sdk.ErrWrongRequest

		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("AddVariableInProject: Cannot load %s: %s\n", key, err)
			return err
		}

		varInProject, err := project.CheckVariableInProject(api.mustDB(), p.ID, varName)
		if err != nil {
			log.Warning("AddVariableInProject: Cannot check if variable %s is already in the project %s: %s\n", varName, p.Name, err)
			return err
		}

		if varInProject {
			return sdk.ErrVariableExists
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("addVariableInProjectHandler: cannot begin tx: %s\n", err)
			return err
		}
		defer tx.Rollback()

		switch newVar.Type {
		case sdk.KeyVariable:
			err = project.AddKeyPair(tx, p, newVar.Name, getUser(ctx))
			break
		default:
			err = project.InsertVariable(tx, p, &newVar, getUser(ctx))
			break
		}
		if err != nil {
			log.Warning("AddVariableInProject: Cannot add variable %s in project %s:  %s\n", varName, p.Name, err)
			return err

		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot update last modified:  %s\n", err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("addVariableInProjectHandler: cannot commit tx: %s\n", err)
			return err
		}

		err = sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p)
		if err != nil {
			log.Warning("AddVariableInProject: Cannot check warnings: %s\n", err)
			return err

		}

		p.Variable, err = project.GetAllVariableInProject(api.mustDB(), p.ID)
		if err != nil {
			log.Warning("AddVariableInProject: Cannot get variables: %s\n", err)
			return err

		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}

func (api *API) getVariableAuditInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "getVariableAuditInProjectHandler> Cannot load project %s", key)
		}

		variable, errV := project.GetVariableInProject(api.mustDB(), p.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "getVariableAuditInProjectHandler> Cannot load variable %s", varName)
		}

		audits, errA := project.LoadVariableAudits(api.mustDB(), p.ID, variable.ID)
		if errA != nil {
			return sdk.WrapError(errA, "getVariableAuditInProjectHandler> Cannot load audit for variable %s", varName)
		}
		return WriteJSON(w, r, audits, http.StatusOK)
	}
}
