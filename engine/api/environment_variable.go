package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getEnvironmentsAuditHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	audits, err := environment.GetEnvironmentAudit(db, key, envName)
	if err != nil {
		log.Warning("getEnvironmentsAuditHandler: Cannot get environment audit for project %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, audits, http.StatusOK)
}

func restoreEnvironmentAuditHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	auditIDString := vars["auditID"]

	auditID, err := strconv.ParseInt(auditIDString, 10, 64)
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot parse auditID %s: %s\n", auditIDString, err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot load project %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot load environment %s: %s\n", envName, err)
		WriteError(w, r, err)
		return
	}

	auditVars, err := environment.GetAudit(db, auditID)
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot get environment audit for project %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot start transaction : %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = environment.CreateAudit(tx, key, env, c.User)
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot create audit: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = environment.DeleteAllVariable(tx, env.ID)
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler> Cannot delete variables on environments for update: %s\n", err)
		WriteError(w, r, err)
		return
	}

	for varIndex := range auditVars {
		varEnv := &auditVars[varIndex]
		if sdk.NeedPlaceholder(varEnv.Type) {
			value, err := secret.Decrypt([]byte(varEnv.Value))
			if err != nil {
				log.Warning("restoreEnvironmentAuditHandler> Cannot decrypt variable %s on environment %s: %s\n", varEnv.Name, envName, err)
				WriteError(w, r, err)
				return
			}
			varEnv.Value = string(value)
		}
		err = environment.InsertVariable(tx, env.ID, varEnv)
		if err != nil {
			log.Warning("restoreEnvironmentAuditHandler> Cannot insert variables on environments: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("restoreEnvironmentAuditHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getVariablesInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	variables, err := environment.GetAllVariable(db, key, envName)
	if err != nil {
		log.Warning("getVariablesInEnvironmentHandler: Cannot get variables for environment %s: %s\n", envName, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, variables, http.StatusOK)
}

func deleteVariableFromEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	varName := vars["name"]

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("deleteVariableFromEnvironmentHandler: Cannot load environment %s :  %s\n", envName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteVariableFromEnvironmentHandler: Cannot start transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = environment.CreateAudit(tx, key, env, c.User)
	if err != nil {
		log.Warning("deleteVariableFromEnvironmentHandler: Cannot create audit for env %s:  %s\n", envName, err)
		WriteError(w, r, err)
		return
	}

	err = environment.DeleteVariable(db, env.ID, varName)
	if err != nil {
		log.Warning("deleteVariableFromEnvironmentHandler: Cannot delete %s: %s\n", varName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteVariableFromEnvironmentHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func updateVariableInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	varName := vars["name"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateVariableInEnvironment: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	var newVar sdk.Variable
	err = json.Unmarshal(data, &newVar)
	if err != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot unmarshal body : %s\n", err)
		WriteError(w, r, err)
		return
	}
	if newVar.Name != varName {
		WriteError(w, r, sdk.ErrNoVariable)
		return
	}

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("updateVariableInEnvironmentHandler: cannot load environment %s: %s\n", envName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot start transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = environment.CreateAudit(tx, key, env, c.User)
	if err != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot create audit for env %s:  %s\n", envName, err)
		WriteError(w, r, err)
		return
	}

	err = environment.UpdateVariable(db, env.ID, newVar)
	if err != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot update variable %s for environment %s:  %s\n", varName, envName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateVariableInEnvironmentHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func addVariableInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	varName := vars["name"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateVariableInEnvironment: Cannot load %s: %s\n", key, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var newVar sdk.Variable
	err = json.Unmarshal(data, &newVar)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if newVar.Name != varName {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("addVariableInEnvironmentHandler: Cannot load environment %s :  %s\n", envName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addVariableInEnvironmentHandler: cannot begin tx: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = environment.CreateAudit(tx, key, env, c.User)
	if err != nil {
		log.Warning("addVariableInEnvironmentHandler: Cannot create audit for env %s:  %s\n", envName, err)
		WriteError(w, r, err)
		return
	}

	switch newVar.Type {
	case sdk.KeyVariable:
		err = keys.AddKeyPairToEnvironment(tx, env.ID, newVar.Name)
		break
	default:
		err = environment.InsertVariable(tx, env.ID, &newVar)
		break
	}
	if err != nil {
		log.Warning("addVariableInEnvironmentHandler: Cannot add variable %s in environment %s:  %s\n", varName, envName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addVariableInEnvironmentHandler: cannot commit tx: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
