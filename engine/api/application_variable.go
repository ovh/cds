package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getVariablesAuditInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]

		audits, err := application.GetVariableAudit(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot get variable audit for application %s", appName)

		}
		return service.WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getVariableAuditInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		app, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName)
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
		return service.WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getVariableInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s", appName)
		}

		variable, err := application.LoadVariable(api.mustDB(), app.ID, varName)
		if err != nil {
			return sdk.WrapError(err, "Cannot get variable %s for application %s", varName, appName)
		}

		return service.WriteJSON(w, variable, http.StatusOK)
	}
}

func (api *API) getVariablesInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]

		variables, err := application.GetAllVariable(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot get variables for application %s", appName)
		}

		return service.WriteJSON(w, variables, http.StatusOK)
	}
}

func (api *API) deleteVariableFromApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application: %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback()

		// Clear password for audit
		varToDelete, errV := application.LoadVariable(api.mustDB(), app.ID, varName, application.WithClearPassword())
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromApplicationHandler> Cannot load variable %s", varName)
		}

		if err := application.DeleteVariable(tx, api.Cache, app, varToDelete, deprecatedGetUser(ctx)); err != nil {
			log.Warning("deleteVariableFromApplicationHandler: Cannot delete %s: %s\n", varName, err)
			return sdk.WrapError(err, "Cannot delete %s", varName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot load variables")
		}

		event.PublishDeleteVariableApplication(key, *app, *varToDelete, deprecatedGetUser(ctx))

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) updateVariableInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName || newVar.Type == sdk.KeyVariable {
			return sdk.ErrWrongRequest
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application: %s", appName)
		}

		variableBefore, err := application.LoadVariableByID(api.mustDB(), app.ID, newVar.ID, application.WithClearPassword())
		if err != nil {
			return sdk.WrapError(err, "cannot load variable %d", variableBefore.ID)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot create transaction")
		}
		defer tx.Rollback()

		if err := application.UpdateVariable(tx, api.Cache, app, &newVar, variableBefore, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot update variable %s for application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot load variables")
		}

		event.PublishUpdateVariableApplication(key, *app, newVar, *variableBefore, deprecatedGetUser(ctx))

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) addVariableInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return err
		}

		if newVar.Name != varName {
			return sdk.ErrWrongRequest
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s ", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback()

		switch newVar.Type {
		case sdk.KeyVariable:
			err = application.AddKeyPairToApplication(tx, api.Cache, app, newVar.Name, deprecatedGetUser(ctx))
			break
		default:
			err = application.InsertVariable(tx, api.Cache, app, newVar, deprecatedGetUser(ctx))
			break
		}
		if err != nil {
			return sdk.WrapError(err, "Cannot add variable %s in application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot get variables")
		}

		event.PublishAddVariableApplication(key, *app, newVar, deprecatedGetUser(ctx))

		return service.WriteJSON(w, app, http.StatusOK)
	}
}
