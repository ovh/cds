package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getVariableAuditInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]
		varName := vars["name"]

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "getVariableAuditInEnvironmentHandler> Cannot load environment %s on project %s", envName, key)
		}

		variable, err := environment.LoadVariable(api.mustDB(), env.ID, varName)
		if err != nil {
			return err
		}

		audits, errA := environment.LoadVariableAudits(api.mustDB(), env.ID, variable.ID)
		if errA != nil {
			return sdk.WrapError(errA, "getVariableAuditInEnvironmentHandler> Cannot load audit for variable %s", varName)
		}
		return service.WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getVariableInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]
		name := vars["name"]

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "getVariableInEnvironmentHandler> Cannot load environment %s on project %s", envName, key)
		}

		variable, err := environment.LoadVariable(api.mustDB(), env.ID, name)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, variable, http.StatusOK)
	}
}

func (api *API) getVariablesInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "getVariablesInEnvironmentHandler> Cannot load environment %s on project %s", envName, key)
		}

		variables, err := environment.LoadAllVariables(api.mustDB(), env.ID)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, variables, http.StatusOK)
	}
}

func (api *API) deleteVariableFromEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]
		varName := vars["name"]

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "deleteVariableFromEnvironmentHandler: Cannot load environment %s", envName)
		}
		if env.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "deleteVariableFromEnvironmentHandler: Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		// clear passwordfor audit
		varToDelete, err := environment.LoadVariable(tx, env.ID, varName)
		if err != nil {
			return err
		}

		if err := environment.DeleteVariable(tx, env.ID, varToDelete, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot delete %s", varName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot commit transaction")
		}
		event.PublishEnvironmentVariableDelete(ctx, key, *env, *varToDelete, getAPIConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) updateVariableInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}
		if newVar.Name != varName || newVar.Type == sdk.KeyVariable {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "updateVariableInEnvironmentHandler: cannot load environment %s", envName)
		}
		if env.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updateVariableInEnvironmentHandler: Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		varBefore, err := environment.LoadVariable(api.mustDB(), env.ID, varName)
		if err != nil {
			return err
		}

		if err := environment.UpdateVariable(api.mustDB(), env.ID, &newVar, varBefore, getAPIConsumer(ctx)); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot commit transaction")
		}

		event.PublishEnvironmentVariableUpdate(ctx, key, *env, newVar, *varBefore, getAPIConsumer(ctx))

		if sdk.NeedPlaceholder(newVar.Type) {
			newVar.Value = sdk.PasswordPlaceholder
		}

		return service.WriteJSON(w, newVar, http.StatusOK)
	}
}

func (api *API) addVariableInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		envName := vars["environmentName"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		if newVar.Name != varName {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "addVariableInEnvironmentHandler: Cannot load environment %s", envName)
		}
		if env.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addVariableInEnvironmentHandler: cannot begin tx")
		}
		defer tx.Rollback() // nolint

		if !sdk.IsInArray(newVar.Type, sdk.AvailableVariableType) {
			return sdk.WithStack(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid variable type %s", newVar.Type))
		}

		if err := environment.InsertVariable(tx, env.ID, &newVar, getAPIConsumer(ctx)); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addVariableInEnvironmentHandler: cannot commit tx")
		}

		event.PublishEnvironmentVariableAdd(ctx, key, *env, newVar, getAPIConsumer(ctx))

		if sdk.NeedPlaceholder(newVar.Type) {
			newVar.Value = sdk.PasswordPlaceholder
		}

		return service.WriteJSON(w, newVar, http.StatusOK)
	}
}
