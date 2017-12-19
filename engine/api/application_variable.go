package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getVariablesAuditInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		audits, err := application.GetVariableAudit(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(err, "getVariablesAuditInApplicationHandler> Cannot get variable audit for application %s", appName)

		}
		return WriteJSON(w, r, audits, http.StatusOK)
	}
}

// Deprecated
func (api *API) restoreAuditHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		auditIDString := vars["auditID"]

		auditID, err := strconv.ParseInt(auditIDString, 10, 64)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "restoreAuditHandler> Cannot parse auditID %s", auditIDString)
		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "restoreAuditHandler> Cannot load %s", key)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(sdk.ErrApplicationNotFound, "restoreAuditHandler> Cannot load application %s ", appName)
		}

		variables, err := application.GetAudit(api.mustDB(), key, appName, auditID)
		if err != nil {
			return sdk.WrapError(err, "restoreAuditHandler> Cannot get variable audit for application %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "restoreAuditHandler> Cannot start transaction ")
		}
		defer tx.Rollback()

		err = application.DeleteAllVariable(tx, app.ID)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "restoreAuditHandler> Cannot delete variables for application %s %s", appName, err)
		}

		for _, v := range variables {
			if sdk.NeedPlaceholder(v.Type) {
				value, err := secret.Decrypt([]byte(v.Value))
				if err != nil {
					return sdk.WrapError(err, "restoreAuditHandler> Cannot decrypt variable %s for application %s", v.Name, appName)
				}
				v.Value = string(value)
			}
			err := application.InsertVariable(tx, api.Cache, app, v, getUser(ctx))
			if err != nil {
				return sdk.WrapError(err, "restoreAuditHandler> Cannot insert variable %s for application %s", v.Name, appName)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "restoreAuditHandler> Cannot commit transaction: %s", err)
		}

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Warning("restoreAuditHandler> Cannot check warnings: %s", err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("restoreAuditHandler> Cannot check application sanity: %s")
			}
		}()

		return nil
	}
}

func (api *API) getVariableAuditInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		varName := vars["name"]

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "getVariableAuditInApplicationHandler> Cannot load application %s on project %s", appName, key)
		}

		variable, errV := application.LoadVariable(api.mustDB(), app.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "getVariableAuditInApplicationHandler> Cannot load variable %s", varName)
		}

		audits, errA := application.LoadVariableAudits(api.mustDB(), app.ID, variable.ID)
		if errA != nil {
			return sdk.WrapError(errA, "getVariableAuditInApplicationHandler> Cannot load audit for variable %s", varName)
		}
		return WriteJSON(w, r, audits, http.StatusOK)
	}
}

func (api *API) getVariableInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		varName := vars["name"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getVariableInApplicationHandler> Cannot load application %s", appName)
		}

		variable, err := application.LoadVariable(api.mustDB(), app.ID, varName)
		if err != nil {
			return sdk.WrapError(err, "getVariableInApplicationHandler> Cannot get variable %s for application %s", varName, appName)
		}

		return WriteJSON(w, r, variable, http.StatusOK)
	}
}

func (api *API) getVariablesInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		variables, err := application.GetAllVariable(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(err, "getVariablesInApplicationHandler> Cannot get variables for application %s", appName)
		}

		return WriteJSON(w, r, variables, http.StatusOK)
	}
}

func (api *API) deleteVariableFromApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		varName := vars["name"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "deleteVariableInApplicationHandler> Cannot load application: %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		// Clear password for audit
		varToDelete, errV := application.LoadVariable(api.mustDB(), app.ID, varName, application.WithClearPassword())
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromApplicationHandler> Cannot load variable %s", varName)
		}

		if err := application.DeleteVariable(tx, api.Cache, app, varToDelete, getUser(ctx)); err != nil {
			log.Warning("deleteVariableFromApplicationHandler: Cannot delete %s: %s\n", varName, err)
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler> Cannot delete %s", varName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler> Cannot commit transaction")
		}

		go func() {
			p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
			if err != nil {
				log.Warning("deleteVariableFromApplicationHandler> Cannot load project %s: %v", key, err)
			}

			app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
			if err != nil {
				log.Warning("deleteVariableInApplicationHandler> Cannot load application: %s: %v", appName, err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("restoreAuditHandler> Cannot check application sanity: %s", err)
			}
		}()

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler> Cannot load variables")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

// deprecated
func (api *API) updateVariablesInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var varsToUpdate []sdk.Variable
		if err := UnmarshalBody(r, &varsToUpdate); err != nil {
			return err
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "updateVariablesInApplicationHandler> Cannot load application %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariablesInApplicationHandler> Cannot unmarshal body ")
		}
		defer tx.Rollback()

		// Preload values, if one password variable has a password placeholder, we can't just insert
		// the placeholder !
		preload, err := application.GetAllVariable(tx, key, appName, application.WithClearPassword())
		if err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler: Cannot preload variables values")
		}

		if err := application.DeleteAllVariable(tx, app.ID); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updateVariablesInApplicationHandler> Cannot delete variables for application %s:  %s", appName, err)
		}

		for _, v := range varsToUpdate {
			switch v.Type {
			case sdk.SecretVariable:
				if sdk.NeedPlaceholder(v.Type) && v.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == v.ID {
							v.Value = p.Value
						}
					}
				}

				if err := application.InsertVariable(tx, api.Cache, app, v, getUser(ctx)); err != nil {
					return sdk.WrapError(err, "updateVariablesInApplicationHandler> Cannot insert variable %s for application %s", v.Name, appName)
				}
				break
			case sdk.KeyVariable:
				if v.Value == "" {
					if err := application.AddKeyPairToApplication(tx, api.Cache, app, v.Name, getUser(ctx)); err != nil {
						return sdk.WrapError(err, "updateVariablesInApplicationHandler> cannot generate keypair")
					}
				} else if v.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == v.ID {
							v.Value = p.Value
						}
					}
					if err := application.InsertVariable(tx, api.Cache, app, v, getUser(ctx)); err != nil {
						return sdk.WrapError(err, "updateVariablesInApplication> Cannot insert variable %s", v.Name)
					}
				}
				break
			default:
				if err := application.InsertVariable(tx, api.Cache, app, v, getUser(ctx)); err != nil {
					return sdk.WrapError(err, "updateVariablesInApplicationHandler> Cannot insert variable %s for application %s", v.Name, appName)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updateVariablesInApplicationHandler> Cannot commit transaction:  %s", err)
		}

		/*
			go func() {
				p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
				if err != nil {
					log.Warning("updateVariablesInApplicationHandler> Cannot load %s: %v", key, err)
				}

				app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
				if err != nil {
					log.Warning("updateVariablesInApplicationHandler> Cannot load application %s: %v", appName, err)
				}

				if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
					log.Warning("updateVariableInApplicationHandler> Cannot check warnings: %s", err)
				}

				if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
					log.Warning("updateVariableInApplicationHandler> Cannot check application sanity: %s", err)
				}
			}()
		*/

		return nil
	}
}

func (api *API) updateVariableInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName {
			return sdk.ErrWrongRequest
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot load application: %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot create transaction")
		}
		defer tx.Rollback()

		if err := application.UpdateVariable(tx, api.Cache, app, &newVar, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot update variable %s for application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot commit transaction")
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot load variables")
		}

		go func() {
			p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
			if err != nil {
				log.Warning("updateVariableInApplicationHandler> Cannot load project %s: %v", key, err)
			}

			app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
			if err != nil {
				log.Warning("updateVariableInApplicationHandler> Cannot load application: %s: %v", appName, err)
			}

			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Warning("updateVariableInApplicationHandler> Cannot check warnings: %v", err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("updateVariableInApplicationHandler> Cannot check application sanity: %s", err)
			}
		}()

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) addVariableInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}

		if newVar.Name != varName {
			return sdk.ErrWrongRequest
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "addVariableInApplicationHandler> Cannot load application %s ", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addVariableInApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		switch newVar.Type {
		case sdk.KeyVariable:
			err = application.AddKeyPairToApplication(tx, api.Cache, app, newVar.Name, getUser(ctx))
			break
		default:
			err = application.InsertVariable(tx, api.Cache, app, newVar, getUser(ctx))
			break
		}
		if err != nil {
			return sdk.WrapError(err, "addVariableInApplicationHandler> Cannot add variable %s in application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addVariableInApplicationHandler> Cannot commit transaction")
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "addVariableInApplicationHandler> Cannot get variables")
		}

		go func() {
			p, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
			if errp != nil {
				log.Warning("addVariableInApplicationHandler> Cannot load %s: %v", key, errp)
			}

			app, erra := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
			if erra != nil {
				log.Warning("addVariableInApplicationHandler> Cannot load application %s: %v", appName, erra)
			}

			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Warning("addVariableInApplicationHandler> Cannot check warnings: %s", err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("addVariableInApplicationHandler> Cannot check application sanity: %s", err)
			}
		}()

		return WriteJSON(w, r, app, http.StatusOK)
	}
}
