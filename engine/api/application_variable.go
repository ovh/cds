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
			log.Warning("getVariablesAuditInApplicationHandler: Cannot get variable audit for application %s: %s\n", appName, err)
			return err

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
			log.Warning("restoreAuditHandler: Cannot parse auditID %s: %s\n", auditIDString, err)
			return sdk.ErrInvalidID
		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			log.Warning("restoreAuditHandler: Cannot load %s: %s\n", key, err)
			return err
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			log.Warning("restoreAuditHandler: Cannot load application %s : %s\n", appName, err)
			return sdk.ErrApplicationNotFound
		}

		variables, err := application.GetAudit(api.mustDB(), key, appName, auditID)
		if err != nil {
			log.Warning("restoreAuditHandler: Cannot get variable audit for application %s: %s\n", appName, err)
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("restoreAuditHandler: Cannot start transaction : %s\n", err)
			return sdk.ErrUnknownError
		}
		defer tx.Rollback()

		err = application.DeleteAllVariable(tx, app.ID)
		if err != nil {
			log.Warning("restoreAuditHandler: Cannot delete variables for application %s:  %s\n", appName, err)
			return sdk.ErrUnknownError
		}

		for _, v := range variables {
			if sdk.NeedPlaceholder(v.Type) {
				value, err := secret.Decrypt([]byte(v.Value))
				if err != nil {
					log.Warning("restoreAuditHandler: Cannot decrypt variable %s for application %s:  %s\n", v.Name, appName, err)
					return err
				}
				v.Value = string(value)
			}
			err := application.InsertVariable(tx, api.Cache, app, v, getUser(ctx))
			if err != nil {
				log.Warning("restoreAuditHandler: Cannot insert variable %s for application %s:  %s\n", v.Name, appName, err)
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			log.Warning("restoreAuditHandler: Cannot commit transaction:  %s\n", err)
			return sdk.ErrUnknownError
		}

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Warning("restoreAuditHandler: Cannot check warnings: %s", err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("restoreAuditHandler: Cannot check application sanity: %s")
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
			log.Warning("getVariableInApplicationHandler: Cannot load application %s: %s\n", appName, err)
			return err
		}

		variable, err := application.LoadVariable(api.mustDB(), app.ID, varName)
		if err != nil {
			log.Warning("getVariableInApplicationHandler: Cannot get variable %s for application %s: %s\n", varName, appName, err)
			return err
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
			log.Warning("getVariablesInApplicationHandler: Cannot get variables for application %s: %s\n", appName, err)
			return err
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

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableInApplicationHandler: Cannot load project %s", key)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableInApplicationHandler: Cannot load application: %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		// Clear password for audit
		varToDelete, errV := application.LoadVariable(api.mustDB(), app.ID, varName, application.WithClearPassword())
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromApplicationHandler> Cannot load variable %s", varName)
		}

		if err := application.DeleteVariable(tx, api.Cache, app, varToDelete, getUser(ctx)); err != nil {
			log.Warning("deleteVariableFromApplicationHandler: Cannot delete %s: %s\n", varName, err)
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot delete %s", varName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot commit transaction")
		}

		go func() {
			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("restoreAuditHandler: Cannot check application sanity: %s", err)
			}
		}()

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot load variables")
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

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
		if err != nil {
			log.Warning("updateVariablesInApplicationHandler: Cannot load %s: %s\n", key, err)
			return err
		}

		var varsToUpdate []sdk.Variable
		if err := UnmarshalBody(r, &varsToUpdate); err != nil {
			return err
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			log.Warning("updateVariablesInApplicationHandler: Cannot load application %s : %s\n", appName, err)
			return sdk.ErrApplicationNotFound
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("updateVariablesInApplicationHandler: Cannot unmarshal body : %s\n", err)
			return sdk.ErrUnknownError
		}
		defer tx.Rollback()

		// Preload values, if one password variable has a password placeholder, we can't just insert
		// the placeholder !
		preload, err := application.GetAllVariable(tx, key, appName, application.WithClearPassword())
		if err != nil {
			log.Warning("updateVariablesInProjectHandler: Cannot preload variables values: %s\n", err)
			return err
		}

		err = application.DeleteAllVariable(tx, app.ID)
		if err != nil {
			log.Warning("updateVariablesInApplicationHandler: Cannot delete variables for application %s:  %s\n", appName, err)
			return sdk.ErrUnknownError
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
					log.Warning("updateVariablesInApplicationHandler: Cannot insert variable %s for application %s:  %s\n", v.Name, appName, err)
					return err
				}
				break
			case sdk.KeyVariable:
				if v.Value == "" {
					if err := application.AddKeyPairToApplication(tx, api.Cache, app, v.Name, getUser(ctx)); err != nil {
						log.Warning("updateVariablesInApplicationHandler> cannot generate keypair: %s\n", err)
						return err
					}
				} else if v.Value == sdk.PasswordPlaceholder {
					for _, p := range preload {
						if p.ID == v.ID {
							v.Value = p.Value
						}
					}
					if err := application.InsertVariable(tx, api.Cache, app, v, getUser(ctx)); err != nil {
						log.Warning("updateVariablesInApplication: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
						return err
					}
				}
				break
			default:
				if err := application.InsertVariable(tx, api.Cache, app, v, getUser(ctx)); err != nil {
					log.Warning("updateVariablesInApplicationHandler: Cannot insert variable %s for application %s:  %s\n", v.Name, appName, err)
					return err
				}
			}
		}

		if err := tx.Commit(); err != nil {
			log.Warning("updateVariablesInApplicationHandler: Cannot commit transaction:  %s\n", err)
			return sdk.ErrUnknownError
		}

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Warning("updateVariableInApplicationHandler: Cannot check warnings: %s\n", err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("updateVariableInApplicationHandler: Cannot check application sanity: %s", err)
			}
		}()

		return nil
	}
}

func (api *API) updateVariableInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		varName := vars["name"]

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot load project %s", key)
		}

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName {
			return sdk.ErrWrongRequest
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot load application: %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot create transaction")
		}
		defer tx.Rollback()

		if err := application.UpdateVariable(tx, api.Cache, app, &newVar, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot update variable %s for application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot commit transaction")
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot load variables")
		}
		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Warning("updateVariableInApplicationHandler: Cannot check warnings: %v", err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("updateVariableInApplicationHandler: Cannot check application sanity: %s", err)
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

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
		if err != nil {
			log.Warning("addVariableInApplicationHandler: Cannot load %s: %s\n", key, err)
			return err
		}

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}

		if newVar.Name != varName {
			return sdk.ErrWrongRequest
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.Default)
		if err != nil {
			log.Warning("addVariableInApplicationHandler: Cannot load application %s :  %s\n", appName, err)
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			log.Warning("addVariableInApplicationHandler: Cannot start transaction:  %s\n", err)
			return err
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
			log.Warning("addVariableInApplicationHandler: Cannot add variable %s in application %s:  %s\n", varName, appName, err)
			return err
		}

		if err := tx.Commit(); err != nil {
			log.Warning("addVariableInApplicationHandler: Cannot commit transaction:  %s\n", err)
			return err
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			log.Warning("addVariableInApplicationHandler: Cannot get variables: %s\n", err)
			return err
		}

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Warning("addVariableInApplicationHandler: Cannot check warnings: %s\n", err)
			}

			if err := sanity.CheckApplication(api.mustDB(), p, app); err != nil {
				log.Warning("addVariableInApplicationHandler: Cannot check application sanity: %s", err)
			}
		}()

		return WriteJSON(w, r, app, http.StatusOK)
	}
}
