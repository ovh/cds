package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

func (api *API) getVariablesAuditInProjectnHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]

		audits, err := project.GetVariableAudit(api.mustDB(), key)
		if err != nil {
			return sdk.WrapError(err, "getVariablesAuditInProjectnHandler: Cannot get variable audit for project %s", key)

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
			return sdk.WrapError(sdk.ErrInvalidID, "restoreProjectVariableAuditHandler: Cannot parse auditID %s", auditIDString)

		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "restoreProjectVariableAuditHandler: Cannot load %s", key)

		}

		variables, err := project.GetAudit(api.mustDB(), key, auditID)
		if err != nil {
			return sdk.WrapError(err, "restoreProjectVariableAuditHandler: Cannot get variable audit for project %s", key)

		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "restoreProjectVariableAuditHandler: Cannot start transaction ")

		}
		defer tx.Rollback()

		if err := project.DeleteAllVariable(tx, p.ID); err != nil {
			return sdk.WrapError(err, "restoreProjectVariableAuditHandler: Cannot delete variables for project %s", key)
		}

		for _, v := range variables {
			if sdk.NeedPlaceholder(v.Type) {
				value, err := secret.Decrypt([]byte(v.Value))
				if err != nil {
					return sdk.WrapError(err, "restoreProjectVariableAuditHandler: Cannot decrypt variable %s for project %s", v.Name, key)

				}
				v.Value = string(value)
			}
			if err := project.InsertVariable(tx, p, &v, getUser(ctx)); err != nil {
				return sdk.WrapError(err, "restoreProjectVariableAuditHandler: Cannot insert variable %s for project %s", v.Name, key)
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "restoreProjectVariableAuditHandler: Cannot update last modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "restoreProjectVariableAuditHandler: Cannot commit transaction")
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
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot load %s", key)
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
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot load %s", key)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot start transaction")
		}
		defer tx.Rollback()

		varToDelete, errV := project.GetVariableInProject(api.mustDB(), p.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromProject> Cannot load variable %s", varName)
		}

		if err := project.DeleteVariable(tx, p, varToDelete, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot delete %s", varName)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot commit transaction")
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
			return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot load %s", key)

		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot start transaction")

		}
		defer tx.Rollback()

		// Preload values, if one password variable has a password placeholder, we can't just insert
		// the placeholder !
		preload, err := project.GetAllVariableInProject(tx, p.ID, project.WithClearPassword())
		if err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot preload variables values")

		}

		if err := project.DeleteAllVariable(tx, p.ID); err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot delete all variables for project %s", p.Key)

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
					return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot insert variable %s in project %s", v.Name, p.Key)

				}
				break
			// In case of a key variable, if empty, generate a pair and add them as variable
			case sdk.KeyVariable:
				if v.Value == "" {
					err := project.AddKeyPair(tx, p, v.Name, getUser(ctx))
					if err != nil {
						return sdk.WrapError(err, "updateVariablesInProjectHandler> cannot generate keypair")

					}
				} else if v.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == v.ID {
							v.Value = p.Value
						}
					}
					err = project.InsertVariable(tx, p, &v, getUser(ctx))
					if err != nil {
						return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot insert variable %s in project %s", v.Name, p.Key)

					}
				}
				break
			default:
				err = project.InsertVariable(tx, p, &v, getUser(ctx))
				if err != nil {
					return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot insert variable %s in project %s", v.Name, p.Key)

				}
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot update last modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler> Cannot commit transaction")
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
			return sdk.WrapError(err, "updateVariableInProject: Cannot load %s", key)

		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariableInProject: cannot start transaction")

		}
		defer tx.Rollback()

		if err := project.UpdateVariable(tx, p, &newVar, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: Cannot update variable %s in project %s", varName, p.Name)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: cannot commit transaction")

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
			return sdk.WrapError(err, "getVariableInProjectHandler: Cannot load project %s", key)

		}

		v, err := project.GetVariableInProject(api.mustDB(), proj.ID, varName)
		if err != nil {
			return sdk.WrapError(err, "getVariableInProjectHandler: Cannot get variable %s in project %s", varName, key)

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
			return sdk.WrapError(err, "AddVariableInProject: Cannot load %s", key)
		}

		varInProject, err := project.CheckVariableInProject(api.mustDB(), p.ID, varName)
		if err != nil {
			return sdk.WrapError(err, "AddVariableInProject: Cannot check if variable %s is already in the project %s", varName, p.Name)
		}

		if varInProject {
			return sdk.ErrVariableExists
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addVariableInProjectHandler: cannot begin tx")
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
			return sdk.WrapError(err, "AddVariableInProject: Cannot add variable %s in project %s", varName, p.Name)

		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler: Cannot update last modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addVariableInProjectHandler: cannot commit tx")
		}

		p.Variable, err = project.GetAllVariableInProject(api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "AddVariableInProject: Cannot get variables")

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
