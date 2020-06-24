package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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

		app, errA := application.LoadByName(api.mustDB(), key, appName)
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

		app, err := application.LoadByName(api.mustDB(), key, appName)
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

		app, err := application.LoadByName(api.mustDB(), key, appName, application.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s", appName)
		}

		return service.WriteJSON(w, app.Variables, http.StatusOK)
	}
}

func (api *API) deleteVariableFromApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		app, err := application.LoadByName(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application: %s", appName)
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		varToDelete, errV := application.LoadVariable(api.mustDB(), app.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromApplicationHandler> Cannot load variable %s", varName)
		}

		if err := application.DeleteVariable(tx, app.ID, varToDelete, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot delete %s", varName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishDeleteVariableApplication(ctx, key, *app, *varToDelete, getAPIConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) updateVariableInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		var newVar sdk.ApplicationVariable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Type == sdk.KeyVariable {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		app, err := application.LoadByName(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application: %s", appName)
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		variableBefore, err := application.LoadVariableWithDecryption(api.mustDB(), app.ID, newVar.ID, varName)
		if err != nil {
			return sdk.WrapError(err, "cannot load variable with id %d", newVar.ID)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot create transaction")
		}
		defer tx.Rollback() // nolint

		if err := application.UpdateVariable(tx, app.ID, &newVar, variableBefore, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot update variable %s for application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishUpdateVariableApplication(ctx, key, *app, newVar, *variableBefore, getAPIConsumer(ctx))

		return service.WriteJSON(w, newVar, http.StatusOK)
	}
}

func (api *API) addVariableInApplicationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		varName := vars["name"]

		var newVar sdk.ApplicationVariable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return err
		}

		if newVar.Name != varName {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		app, err := application.LoadByName(api.mustDB(), key, appName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s ", appName)
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if !sdk.IsInArray(newVar.Type, sdk.AvailableVariableType) {
			return sdk.WithStack(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid variable type %s", newVar.Type))
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err = application.InsertVariable(tx, app.ID, &newVar, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot add variable %s in application %s", varName, appName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishAddVariableApplication(ctx, key, *app, newVar, getAPIConsumer(ctx))

		return service.WriteJSON(w, newVar, http.StatusOK)
	}
}
