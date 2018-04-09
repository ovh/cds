package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) getVariableAuditInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		varName := vars["name"]

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "getVariableAuditInEnvironmentHandler> Cannot load environment %s on project %s", envName, key)
		}

		variable, errV := environment.GetVariable(api.mustDB(), key, envName, varName)
		if errV != nil {
			return sdk.WrapError(errV, "getVariableAuditInEnvironmentHandler> Cannot load variable %s", varName)
		}

		audits, errA := environment.LoadVariableAudits(api.mustDB(), env.ID, variable.ID)
		if errA != nil {
			return sdk.WrapError(errA, "getVariableAuditInEnvironmentHandler> Cannot load audit for variable %s", varName)
		}
		return WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getVariableInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		name := vars["name"]

		v, errVar := environment.GetVariable(api.mustDB(), key, envName, name)
		if errVar != nil {
			return sdk.WrapError(errVar, "getVariableInEnvironmentHandler: Cannot get variable %s for environment %s", name, envName)
		}

		return WriteJSON(w, v, http.StatusOK)
	}
}

func (api *API) getVariablesInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		variables, errVar := environment.GetAllVariable(api.mustDB(), key, envName)
		if errVar != nil {
			return sdk.WrapError(errVar, "getVariablesInEnvironmentHandler: Cannot get variables for environment %s", envName)
		}

		return WriteJSON(w, variables, http.StatusOK)
	}
}

func (api *API) deleteVariableFromEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		varName := vars["name"]

		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "deleteVariableFromEnvironmentHandler: Cannot load project %s", key)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "deleteVariableFromEnvironmentHandler: Cannot load environment %s", envName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "deleteVariableFromEnvironmentHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		// clear passwordfor audit
		varToDelete, errV := environment.GetVariable(tx, key, envName, varName, environment.WithClearPassword())
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromEnvironmentHandler> Cannot load variable %s", varName)
		}

		if err := environment.DeleteVariable(tx, env.ID, varToDelete, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot delete %s", varName)
		}

		if err := environment.UpdateLastModified(tx, api.Cache, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler> Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot commit transaction")
		}

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "deleteVariableFromEnvironmentHandler: Cannot load environments")
		}

		event.PublishEnvironmentVariableDelete(key, *env, *varToDelete, getUser(ctx))

		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) updateVariableInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		varName := vars["name"]

		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "updateVariableInEnvironment: Cannot load %s", key)
		}

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return sdk.ErrWrongRequest
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "updateVariableInEnvironmentHandler: cannot load environment %s", envName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "updateVariableInEnvironmentHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		varBefore, errV := environment.GetVariableByID(api.mustDB(), env.ID, newVar.ID, environment.WithClearPassword())
		if errV != nil {
			return sdk.WrapError(errV, "updateVariableInEnvironmentHandler> Cannot load variable %d", newVar.ID)
		}

		if err := environment.UpdateVariable(api.mustDB(), env.ID, &newVar, varBefore, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot update variable %s for environment %s", varName, envName)
		}

		if err := environment.UpdateLastModified(tx, api.Cache, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot commit transaction")
		}

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "updateVariableInEnvironmentHandler: Cannot load environments")
		}

		event.PublishEnvironmentVariableUpdate(key, *env, newVar, varBefore, getUser(ctx))

		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) addVariableInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		varName := vars["name"]

		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "addVariableInEnvironmentHandler: Cannot load project %s", key)
		}

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return sdk.ErrWrongRequest
		}

		if newVar.Name != varName {
			return sdk.ErrWrongRequest
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "addVariableInEnvironmentHandler: Cannot load environment %s", envName)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "addVariableInEnvironmentHandler: cannot begin tx")
		}
		defer tx.Rollback()

		var errInsert error
		switch newVar.Type {
		case sdk.KeyVariable:
			errInsert = environment.AddKeyPairToEnvironment(tx, env.ID, newVar.Name, getUser(ctx))
		default:
			errInsert = environment.InsertVariable(tx, env.ID, &newVar, getUser(ctx))
		}
		if errInsert != nil {
			return sdk.WrapError(errInsert, "addVariableInEnvironmentHandler: Cannot add variable %s in environment %s", varName, envName)
		}

		if err := environment.UpdateLastModified(tx, api.Cache, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "addVariableInEnvironmentHandler> Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "addVariableInEnvironmentHandler: Cannot update last modified date")
		}
		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addVariableInEnvironmentHandler: cannot commit tx")
		}

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "addVariableInEnvironmentHandler: Cannot load environments")
		}

		event.PublishEnvironmentVariableAdd(key, *env, newVar, getUser(ctx))

		return WriteJSON(w, p, http.StatusOK)
	}
}
