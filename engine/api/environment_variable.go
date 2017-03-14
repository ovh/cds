package main

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getEnvironmentsAuditHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	audits, errAudit := environment.GetEnvironmentAudit(db, key, envName)
	if errAudit != nil {
		log.Warning("getEnvironmentsAuditHandler: Cannot get environment audit for project %s: %s\n", key, errAudit)
		return errAudit
	}
	return WriteJSON(w, r, audits, http.StatusOK)
}

func restoreEnvironmentAuditHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	auditIDString := vars["auditID"]

	auditID, errAudit := strconv.ParseInt(auditIDString, 10, 64)
	if errAudit != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot parse auditID %s: %s\n", auditIDString, errAudit)
		return sdk.ErrInvalidID
	}

	p, errProj := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errProj != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot load project %s: %s\n", key, errProj)
		return errProj
	}

	env, errEnv := environment.LoadEnvironmentByName(db, key, envName)
	if errEnv != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot load environment %s: %s\n", envName, errEnv)
		return errEnv
	}

	auditVars, errGetAudit := environment.GetAudit(db, auditID)
	if errGetAudit != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot get environment audit for project %s: %s\n", key, errGetAudit)
		return errGetAudit
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot start transaction : %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if err := environment.CreateAudit(tx, key, env, c.User); err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot create audit: %s\n", err)
		return err
	}

	if err := environment.DeleteAllVariable(tx, env.ID); err != nil {
		log.Warning("restoreEnvironmentAuditHandler> Cannot delete variables on environments for update: %s\n", err)
		return err
	}

	for varIndex := range auditVars {
		varEnv := &auditVars[varIndex]
		if sdk.NeedPlaceholder(varEnv.Type) {
			value, errDecrypt := secret.Decrypt([]byte(varEnv.Value))
			if errDecrypt != nil {
				log.Warning("restoreEnvironmentAuditHandler> Cannot decrypt variable %s on environment %s: %s\n", varEnv.Name, envName, errDecrypt)
				return errDecrypt
			}
			varEnv.Value = string(value)
		}
		if err := environment.InsertVariable(tx, env.ID, varEnv, c.User); err != nil {
			log.Warning("restoreEnvironmentAuditHandler> Cannot insert variables on environments: %s\n", err)
			return err
		}
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		log.Warning("restoreEnvironmentAuditHandler> Cannot update last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot commit transaction:  %s\n", err)
		return err
	}

	if err := sanity.CheckProjectPipelines(db, p); err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot check warnings: %s\n", err)
		return err
	}

	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, p.Key, true, c.User)
	if errEnvs != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot load environments: %s\n", errEnvs)
		return errEnvs
	}

	apps, errApps := application.LoadAll(db, p.Key, c.User, application.LoadOptions.WithVariables)
	if errApps != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot load applications: %s\n", errApps)
		return errApps
	}
	for _, a := range apps {
		if err := sanity.CheckApplication(db, p, &a); err != nil {
			log.Warning("restoreAuditHandler: Cannot check application sanity: %s\n", err)
			return err
		}
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func getVariableAuditInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	varName := vars["name"]

	env, errE := environment.LoadEnvironmentByName(db, key, envName)
	if errE != nil {
		return sdk.WrapError(errE, "getVariableAuditInEnvironmentHandler> Cannot load environment %s on project %s", envName, key)
	}

	variable, errV := environment.GetVariable(db, key, envName, varName)
	if errV != nil {
		return sdk.WrapError(errV, "getVariableAuditInEnvironmentHandler> Cannot load variable %s", varName)
	}

	audits, errA := environment.LoadVariableAudits(db, env.ID, variable.ID)
	if errA != nil {
		return sdk.WrapError(errA, "getVariableAuditInEnvironmentHandler> Cannot load audit for variable %s", varName)
	}
	return WriteJSON(w, r, audits, http.StatusOK)
}

func getVariableInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	name := vars["name"]

	v, errVar := environment.GetVariable(db, key, envName, name)
	if errVar != nil {
		log.Warning("getVariableInEnvironmentHandler: Cannot get variable %s for environment %s: %s\n", name, envName, errVar)
		return errVar
	}

	return WriteJSON(w, r, v, http.StatusOK)
}

func getVariablesInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	variables, errVar := environment.GetAllVariable(db, key, envName)
	if errVar != nil {
		log.Warning("getVariablesInEnvironmentHandler: Cannot get variables for environment %s: %s\n", envName, errVar)
		return errVar
	}

	return WriteJSON(w, r, variables, http.StatusOK)
}

func deleteVariableFromEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	varName := vars["name"]

	p, errProj := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errProj != nil {
		return sdk.WrapError(errProj, "deleteVariableFromEnvironmentHandler: Cannot load project %s", key)
	}

	env, errEnv := environment.LoadEnvironmentByName(db, key, envName)
	if errEnv != nil {
		return sdk.WrapError(errEnv, "deleteVariableFromEnvironmentHandler: Cannot load environment %s", envName)
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "deleteVariableFromEnvironmentHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	// DEPRECATED
	if err := environment.CreateAudit(tx, key, env, c.User); err != nil {
		log.Warning("deleteVariableFromEnvironmentHandler: Cannot create audit for env %s:  %s\n", envName, err)
		return err
	}

	// clear passwordfor audit
	varToDelete, errV := environment.GetVariable(db, key, envName, varName, environment.WithClearPassword())
	if errV != nil {
		return sdk.WrapError(errV, "deleteVariableFromEnvironmentHandler> Cannot load variable %s", varName)
	}

	if err := environment.DeleteVariable(db, env.ID, varToDelete, c.User); err != nil {
		return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot delete %s", varName)
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler> Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot commit transaction")
	}

	apps, errApps := application.LoadAll(db, p.Key, c.User, application.LoadOptions.WithVariables)
	if errApps != nil {
		return sdk.WrapError(errApps, "deleteVariableFromEnvironmentHandler: Cannot load applications")
	}
	for _, a := range apps {
		if err := sanity.CheckApplication(db, p, &a); err != nil {
			return sdk.WrapError(err, "deleteVariableFromEnvironmentHandler: Cannot check application sanity")
		}
	}

	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, key, true, c.User)
	if errEnvs != nil {
		return sdk.WrapError(errEnvs, "deleteVariableFromEnvironmentHandler: Cannot load environments")
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func updateVariableInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	varName := vars["name"]

	p, errProj := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errProj != nil {
		return sdk.WrapError(errProj, "updateVariableInEnvironment: Cannot load %s", key)
	}

	var newVar sdk.Variable
	if err := UnmarshalBody(r, &newVar); err != nil {
		return sdk.ErrWrongRequest
	}

	env, errEnv := environment.LoadEnvironmentByName(db, key, envName)
	if errEnv != nil {
		return sdk.WrapError(errEnv, "updateVariableInEnvironmentHandler: cannot load environment %s", envName)
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "updateVariableInEnvironmentHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	// DEPRECATED
	if err := environment.CreateAudit(tx, key, env, c.User); err != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot create audit for env %s:  %s\n", envName, err)
		return err
	}

	if err := environment.UpdateVariable(db, env.ID, &newVar, c.User); err != nil {
		return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot update variable %s for environment %s", varName, envName)
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot commit transaction")
	}

	if err := sanity.CheckProjectPipelines(db, p); err != nil {
		return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot check warnings")
	}

	apps, errApps := application.LoadAll(db, p.Key, c.User, application.LoadOptions.WithVariables)
	if errApps != nil {
		return sdk.WrapError(errApps, "updateVariableInEnvironmentHandler: Cannot load applications")
	}
	for _, a := range apps {
		if err := sanity.CheckApplication(db, p, &a); err != nil {
			return sdk.WrapError(err, "updateVariableInEnvironmentHandler: Cannot check application sanity")
		}
	}

	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, key, true, c.User)
	if errEnvs != nil {
		return sdk.WrapError(errEnvs, "updateVariableInEnvironmentHandler: Cannot load environments")
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func addVariableInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	varName := vars["name"]

	p, errProj := project.Load(db, key, c.User, project.LoadOptions.Default)
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

	env, errEnv := environment.LoadEnvironmentByName(db, key, envName)
	if errEnv != nil {
		return sdk.WrapError(errEnv, "addVariableInEnvironmentHandler: Cannot load environment %s", envName)
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "addVariableInEnvironmentHandler: cannot begin tx")
	}
	defer tx.Rollback()

	// DEPRECATED
	if err := environment.CreateAudit(tx, key, env, c.User); err != nil {
		log.Warning("addVariableInEnvironmentHandler: Cannot create audit for env %s:  %s\n", envName, err)
		return sdk.WrapError(err, "addVariableInEnvironmentHandler: Cannot create audit for env %s", envName)
	}

	var errInsert error
	switch newVar.Type {
	case sdk.KeyVariable:
		errInsert = environment.AddKeyPairToEnvironment(tx, env.ID, newVar.Name, c.User)
	default:
		errInsert = environment.InsertVariable(tx, env.ID, &newVar, c.User)
	}
	if errInsert != nil {
		return sdk.WrapError(errInsert, "addVariableInEnvironmentHandler: Cannot add variable %s in environment %s", varName, envName)
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "addVariableInEnvironmentHandler: Cannot update last modified date")
	}
	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addVariableInEnvironmentHandler: cannot commit tx")
	}

	if err := sanity.CheckProjectPipelines(db, p); err != nil {
		return sdk.WrapError(err, "addVariableInEnvironmentHandler: Cannot check warnings")
	}

	apps, errApps := application.LoadAll(db, p.Key, c.User, application.LoadOptions.WithVariables)
	if errApps != nil {
		return sdk.WrapError(errApps, "addVariableInEnvironmentHandler: Cannot load applications")
	}
	for _, a := range apps {
		if err := sanity.CheckApplication(db, p, &a); err != nil {
			return sdk.WrapError(err, "addVariableInEnvironmentHandler: Cannot check application sanity")
		}
	}

	var errEnvs error
	p.Environments, errEnvs = environment.LoadEnvironments(db, key, true, c.User)
	if errEnvs != nil {
		return sdk.WrapError(errEnvs, "addVariableInEnvironmentHandler: Cannot load environments")
	}

	return WriteJSON(w, r, p, http.StatusOK)
}
