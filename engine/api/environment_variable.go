package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Deprecated
func (api *API) getEnvironmentsAuditHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		audits, errAudit := environment.GetEnvironmentAudit(api.mustDB(), key, envName)
		if errAudit != nil {
			return sdk.WrapError(errAudit, "getEnvironmentsAuditHandler: Cannot get environment audit for project %s", key)
		}
		return WriteJSON(w, r, audits, http.StatusOK)
	}
}

// Deprecated
func (api *API) restoreEnvironmentAuditHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		auditIDString := vars["auditID"]

		auditID, errAudit := strconv.ParseInt(auditIDString, 10, 64)
		if errAudit != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "restoreEnvironmentAuditHandler: Cannot parse auditID %s", auditIDString)
		}

		p, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if errProj != nil {
			return sdk.WrapError(errProj, "restoreEnvironmentAuditHandler: Cannot load project %s", key)
		}

		env, errEnv := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errEnv != nil {
			return sdk.WrapError(errEnv, "restoreEnvironmentAuditHandler: Cannot load environment %s", envName)
		}

		auditVars, errGetAudit := environment.GetAudit(api.mustDB(), auditID)
		if errGetAudit != nil {
			return sdk.WrapError(errGetAudit, "restoreEnvironmentAuditHandler: Cannot get environment audit for project %s", key)
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "restoreEnvironmentAuditHandler: Cannot start transaction ")
		}
		defer tx.Rollback()

		if err := environment.DeleteAllVariable(tx, env.ID); err != nil {
			return sdk.WrapError(err, "restoreEnvironmentAuditHandler> Cannot delete variables on environments for update")
		}

		for varIndex := range auditVars {
			varEnv := &auditVars[varIndex]
			if sdk.NeedPlaceholder(varEnv.Type) {
				value, errDecrypt := secret.Decrypt([]byte(varEnv.Value))
				if errDecrypt != nil {
					return sdk.WrapError(errDecrypt, "restoreEnvironmentAuditHandler> Cannot decrypt variable %s on environment %s", varEnv.Name, envName)
				}
				varEnv.Value = string(value)
			}
			if err := environment.InsertVariable(tx, env.ID, varEnv, getUser(ctx)); err != nil {
				return sdk.WrapError(err, "restoreEnvironmentAuditHandler> Cannot insert variables on environments")
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectEnvironmentLastModificationType); err != nil {
			return sdk.WrapError(err, "restoreEnvironmentAuditHandler> Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "restoreEnvironmentAuditHandler: Cannot commit transaction")
		}

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Error("restoreEnvironmentAuditHandler: Cannot check warnings: %s", err)
			}
		}()

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), p.Key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "restoreEnvironmentAuditHandler: Cannot load environments")
		}

		apps, errApps := application.LoadAll(api.mustDB(), api.Cache, p.Key, getUser(ctx), application.LoadOptions.WithVariables)
		if errApps != nil {
			return sdk.WrapError(errApps, "updateVariableInEnvironmentHandler: Cannot load applications")
		}
		for _, a := range apps {
			if err := sanity.CheckApplication(api.mustDB(), p, &a); err != nil {
				return sdk.WrapError(err, "restoreAuditHandler: Cannot check application sanity")
			}
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}

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
		return WriteJSON(w, r, audits, http.StatusOK)
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

		return WriteJSON(w, r, v, http.StatusOK)
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

		return WriteJSON(w, r, variables, http.StatusOK)
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

		apps, errApps := application.LoadAll(api.mustDB(), api.Cache, p.Key, getUser(ctx), application.LoadOptions.WithVariables)
		if errApps != nil {
			return sdk.WrapError(errApps, "deleteVariableFromEnvironmentHandler: Cannot load applications")
		}
		for _, a := range apps {
			if err := sanity.CheckApplication(api.mustDB(), p, &a); err != nil {
				return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot check application sanity")
			}
		}

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "deleteVariableFromEnvironmentHandler: Cannot load environments")
		}

		return WriteJSON(w, r, p, http.StatusOK)
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

		if err := environment.UpdateVariable(api.mustDB(), env.ID, &newVar, getUser(ctx)); err != nil {
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

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Error("updateVariableInEnvironmentHandler: Cannot check warnings: %s", err)
			}
		}()

		apps, errApps := application.LoadAll(api.mustDB(), api.Cache, p.Key, getUser(ctx), application.LoadOptions.WithVariables)
		if errApps != nil {
			return sdk.WrapError(errApps, "updateVariableInEnvironmentHandler: Cannot load applications")
		}
		for _, a := range apps {
			if err := sanity.CheckApplication(api.mustDB(), p, &a); err != nil {
				return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot check application sanity")
			}
		}

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "updateVariableInEnvironmentHandler: Cannot load environments")
		}

		return WriteJSON(w, r, p, http.StatusOK)
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

		go func() {
			if err := sanity.CheckProjectPipelines(api.mustDB(), api.Cache, p); err != nil {
				log.Error("addVariableInEnvironmentHandler: Cannot check warnings: %s", err)
			}
		}()

		apps, errApps := application.LoadAll(api.mustDB(), api.Cache, p.Key, getUser(ctx), application.LoadOptions.WithVariables)
		if errApps != nil {
			return sdk.WrapError(errApps, "addVariableInEnvironmentHandler: Cannot load applications")
		}
		go func() {
			for _, a := range apps {
				if err := sanity.CheckApplication(api.mustDB(), p, &a); err != nil {
					log.Warning("addVariableInEnvironmentHandler: Cannot check application sanity: %s", err)
				}
			}
		}()

		var errEnvs error
		p.Environments, errEnvs = environment.LoadEnvironments(api.mustDB(), key, true, getUser(ctx))
		if errEnvs != nil {
			return sdk.WrapError(errEnvs, "addVariableInEnvironmentHandler: Cannot load environments")
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}
