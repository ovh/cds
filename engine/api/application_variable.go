package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
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
		return WriteJSON(w, audits, http.StatusOK)
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
		return WriteJSON(w, audits, http.StatusOK)
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

		return WriteJSON(w, variable, http.StatusOK)
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

		return WriteJSON(w, variables, http.StatusOK)
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

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromApplicationHandler> Cannot load variables")
		}

		event.PublishDeleteVariableApplication(key, *app, *varToDelete, getUser(ctx))

		return WriteJSON(w, app, http.StatusOK)
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
		if newVar.Name != varName || newVar.Type == sdk.KeyVariable {
			return sdk.ErrWrongRequest
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot load application: %s", appName)
		}

		variableBefore, err := application.LoadVariableByID(api.mustDB(), app.ID, newVar.ID, application.WithClearPassword())
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> cannot load variable %d", variableBefore.ID)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot create transaction")
		}
		defer tx.Rollback()

		if err := application.UpdateVariable(tx, api.Cache, app, &newVar, variableBefore, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot update variable %s for application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot commit transaction")
		}

		app.Variable, err = application.GetAllVariableByID(api.mustDB(), app.ID)
		if err != nil {
			return sdk.WrapError(err, "updateVariableInApplicationHandler> Cannot load variables")
		}

		event.PublishUpdateVariableApplication(key, *app, newVar, *variableBefore, getUser(ctx))

		return WriteJSON(w, app, http.StatusOK)
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

		event.PublishAddVariableApplication(key, *app, newVar, getUser(ctx))

		return WriteJSON(w, app, http.StatusOK)
	}
}
