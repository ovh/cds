package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getVariablesAuditInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	audits, err := application.GetVariableAudit(db, key, appName)
	if err != nil {
		log.Warning("getVariablesAuditInApplicationHandler: Cannot get variable audit for application %s: %s\n", appName, err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, audits, http.StatusOK)
}

func restoreAuditHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	auditIDString := vars["auditID"]

	auditID, err := strconv.ParseInt(auditIDString, 10, 64)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot parse auditID %s: %s\n", auditIDString, err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot load application %s : %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	variables, err := application.GetAudit(db, key, appName, auditID)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot get variable audit for application %s: %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot start transaction : %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	err = application.CreateAudit(tx, key, app, c.User)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot create audit: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = application.DeleteAllVariable(tx, app.ID)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot delete variables for application %s:  %s\n", appName, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	for _, v := range variables {
		if sdk.NeedPlaceholder(v.Type) {
			value, err := secret.Decrypt([]byte(v.Value))
			if err != nil {
				log.Warning("restoreAuditHandler: Cannot decrypt variable %s for application %s:  %s\n", v.Name, appName, err)
				WriteError(w, r, err)
				return
			}
			v.Value = string(value)
		}
		err := application.InsertVariable(tx, app, v)
		if err != nil {
			log.Warning("restoreAuditHandler: Cannot insert variable %s for application %s:  %s\n", v.Name, appName, err)
			WriteError(w, r, err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}
	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))
}

func getVariableInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("getVariableInApplicationHandler: Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	variable, err := application.LoadVariable(db, app.ID, varName)
	if err != nil {
		log.Warning("getVariableInApplicationHandler: Cannot get variable %s for application %s: %s\n", varName, appName, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, variable, http.StatusOK)
}

func getVariablesInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	variables, err := application.GetAllVariable(db, key, appName)
	if err != nil {
		log.Warning("getVariablesInApplicationHandler: Cannot get variables for application %s: %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, variables, http.StatusOK)
}

func deleteVariableFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("deleteVariableInApplicationHandler: Cannot load application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteVariableFromApplicationHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = application.CreateAudit(tx, key, app, c.User)
	if err != nil {
		log.Warning("deleteVariableFromApplicationHandler: Cannot create variable audit for application %s: %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	err = application.DeleteVariable(tx, app, varName)
	if err != nil {
		log.Warning("deleteVariableFromApplicationHandler: Cannot delete %s: %s\n", varName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteVariableFromApplicationHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	app.Variable, err = application.GetAllVariableByID(db, app.ID)
	if err != nil {
		log.Warning("deleteVariableFromApplicationHandler: Cannot load variables: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	WriteJSON(w, r, app, http.StatusOK)
}

func updateVariablesInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot read body: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var varsToUpdate []sdk.Variable
	err = json.Unmarshal(data, &varsToUpdate)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot unmarshal body : %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot load application %s : %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot unmarshal body : %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	err = application.CreateAudit(tx, key, app, c.User)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot create audit: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Preload values, if one password variable has a password placeholder, we can't just insert
	// the placeholder !
	preload, err := application.GetAllVariable(tx, key, appName, application.WithClearPassword())
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot preload variables values: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = application.DeleteAllVariable(tx, app.ID)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot delete variables for application %s:  %s\n", appName, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
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
			err := application.InsertVariable(tx, app, v)
			if err != nil {
				log.Warning("updateVariablesInApplicationHandler: Cannot insert variable %s for application %s:  %s\n", v.Name, appName, err)
				WriteError(w, r, err)
				return
			}
			break
		case sdk.KeyVariable:
			if v.Value == "" {
				err := keys.AddKeyPairToApplication(tx, app, v.Name)
				if err != nil {
					log.Warning("updateVariablesInApplicationHandler> cannot generate keypair: %s\n", err)
					WriteError(w, r, err)
					return
				}
			} else if v.Value == sdk.PasswordPlaceholder {
				for _, p := range preload {
					if p.ID == v.ID {
						v.Value = p.Value
					}
				}
				err = application.InsertVariable(tx, app, v)
				if err != nil {
					log.Warning("updateVariablesInApplication: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
					WriteError(w, r, err)
					return
				}
			}
			break
		default:
			err := application.InsertVariable(tx, app, v)
			if err != nil {
				log.Warning("updateVariablesInApplicationHandler: Cannot insert variable %s for application %s:  %s\n", v.Name, appName, err)
				WriteError(w, r, err)
				return
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))
}

func updateVariableInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot read body: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var newVar sdk.Variable
	err = json.Unmarshal(data, &newVar)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot unmarshal body : %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	if newVar.Name != varName {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot load application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot create transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = application.CreateAudit(tx, key, app, c.User)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot create audit: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = application.UpdateVariable(tx, app, newVar)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot update variable %s for application %s:  %s\n", varName, appName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	app.Variable, err = application.GetAllVariableByID(db, app.ID)
	if err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot load variables: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	WriteJSON(w, r, app, http.StatusOK)
}

func addVariableInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var newVar sdk.Variable
	err = json.Unmarshal(data, &newVar)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if newVar.Name != varName {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot load application %s :  %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot start transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = application.CreateAudit(tx, key, app, c.User)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot create variable audit for application %s:  %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	switch newVar.Type {
	case sdk.KeyVariable:
		err = keys.AddKeyPairToApplication(tx, app, newVar.Name)
		break
	default:
		err = application.InsertVariable(tx, app, newVar)
		break
	}
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot add variable %s in application %s:  %s\n", varName, appName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	app.Variable, err = application.GetAllVariableByID(db, app.ID)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot get variables: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	WriteJSON(w, r, app, http.StatusOK)
}
