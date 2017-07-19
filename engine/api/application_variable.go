package main

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getVariablesAuditInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	audits, err := application.GetVariableAudit(db, key, appName)
	if err != nil {
		log.Warning("getVariablesAuditInApplicationHandler: Cannot get variable audit for application %s: %s", appName, err)
		return err

	}
	return WriteJSON(w, r, audits, http.StatusOK)
}

// Deprecated
func restoreAuditHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	auditIDString := vars["auditID"]

	auditID, err := strconv.ParseInt(auditIDString, 10, 64)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot parse auditID %s: %s", auditIDString, err)
		return sdk.ErrInvalidID
	}

	p, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot load %s: %s", key, err)
		return err
	}

	app, err := application.LoadByName(db, key, appName, c.User, application.LoadOptions.Default)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot load application %s : %s", appName, err)
		return sdk.ErrApplicationNotFound
	}

	variables, err := application.GetAudit(db, key, appName, auditID)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot get variable audit for application %s: %s", appName, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot start transaction : %s", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	err = application.DeleteAllVariable(tx, app.ID)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot delete variables for application %s:  %s", appName, err)
		return sdk.ErrUnknownError
	}

	for _, v := range variables {
		if sdk.NeedPlaceholder(v.Type) {
			value, err := secret.Decrypt([]byte(v.Value))
			if err != nil {
				log.Warning("restoreAuditHandler: Cannot decrypt variable %s for application %s:  %s", v.Name, appName, err)
				return err
			}
			v.Value = string(value)
		}
		err := application.InsertVariable(tx, app, v, c.User)
		if err != nil {
			log.Warning("restoreAuditHandler: Cannot insert variable %s for application %s:  %s", v.Name, appName, err)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot commit transaction:  %s", err)
		return sdk.ErrUnknownError
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("restoreAuditHandler: Cannot check warnings: %s", err)
		return err
	}

	if err := sanity.CheckApplication(tx, p, app); err != nil {
		log.Warning("restoreAuditHandler: Cannot check application sanity: %s", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))
	return nil
}

func getVariableAuditInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	app, errA := application.LoadByName(db, key, appName, c.User)
	if errA != nil {
		return sdk.WrapError(errA, "getVariableAuditInApplicationHandler> Cannot load application %s on project %s", appName, key)
	}

	variable, errV := application.LoadVariable(db, app.ID, varName)
	if errV != nil {
		return sdk.WrapError(errV, "getVariableAuditInApplicationHandler> Cannot load variable %s", varName)
	}

	audits, errA := application.LoadVariableAudits(db, app.ID, variable.ID)
	if errA != nil {
		return sdk.WrapError(errA, "getVariableAuditInApplicationHandler> Cannot load audit for variable %s", varName)
	}
	return WriteJSON(w, r, audits, http.StatusOK)
}

func getVariableInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	app, err := application.LoadByName(db, key, appName, c.User)
	if err != nil {
		log.Warning("getVariableInApplicationHandler: Cannot load application %s: %s", appName, err)
		return err
	}

	variable, err := application.LoadVariable(db, app.ID, varName)
	if err != nil {
		log.Warning("getVariableInApplicationHandler: Cannot get variable %s for application %s: %s", varName, appName, err)
		return err
	}

	return WriteJSON(w, r, variable, http.StatusOK)
}

func getVariablesInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return WriteJSON(w, r, c.Application.Variable, http.StatusOK)
}

func deleteVariableFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	varName := vars["name"]
	key := vars["key"]

	p, err := project.Load(db, key, c.User, project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
	if err != nil {
		return sdk.WrapError(err, "deleteVariableInApplicationHandler: Cannot load project %s", key)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	// Clear password for audit
	varToDelete, errV := application.LoadVariable(db, c.Application.ID, varName, application.WithClearPassword())
	if errV != nil {
		return sdk.WrapError(errV, "deleteVariableFromApplicationHandler> Cannot load variable %s", varName)
	}

	if err := application.DeleteVariable(tx, c.Application, varToDelete, c.User); err != nil {
		log.Warning("deleteVariableFromApplicationHandler: Cannot delete %s: %s", varName, err)
		return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot delete %s", varName)
	}

	if err := sanity.CheckApplication(tx, p, c.Application); err != nil {
		return sdk.WrapError(err, "restoreAuditHandler: Cannot check application sanity")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot commit transaction")
	}

	c.Application.Variable, err = application.GetAllVariableByID(db, c.Application.ID)
	if err != nil {
		return sdk.WrapError(err, "deleteVariableFromApplicationHandler: Cannot load variables")
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

// deprecated
func updateVariablesInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]

	p, err := project.Load(db, key, c.User, project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot load %s: %s", key, err)
		return err
	}

	var varsToUpdate []sdk.Variable
	if err := UnmarshalBody(r, &varsToUpdate); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot unmarshal body : %s", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	// Preload values, if one password variable has a password placeholder, we can't just insert
	// the placeholder !
	preload, err := application.GetAllVariable(tx, key, c.Application.Name, application.WithClearPassword())
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot preload variables values: %s", err)
		return err
	}

	err = application.DeleteAllVariable(tx, c.Application.ID)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot delete variables for application %s:  %s", c.Application.Name, err)
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

			if err := application.InsertVariable(tx, c.Application, v, c.User); err != nil {
				log.Warning("updateVariablesInApplicationHandler: Cannot insert variable %s for application %s:  %s", v.Name, c.Application.Name, err)
				return err
			}
			break
		case sdk.KeyVariable:
			if v.Value == "" {
				if err := application.AddKeyPairToApplication(tx, c.Application, v.Name, c.User); err != nil {
					log.Warning("updateVariablesInApplicationHandler> cannot generate keypair: %s", err)
					return err
				}
			} else if v.Value == sdk.PasswordPlaceholder {
				for _, p := range preload {
					if p.ID == v.ID {
						v.Value = p.Value
					}
				}
				if err := application.InsertVariable(tx, c.Application, v, c.User); err != nil {
					log.Warning("updateVariablesInApplication: Cannot insert variable %s in project %s: %s", v.Name, p.Key, err)
					return err
				}
			}
			break
		default:
			if err := application.InsertVariable(tx, c.Application, v, c.User); err != nil {
				log.Warning("updateVariablesInApplicationHandler: Cannot insert variable %s for application %s:  %s", v.Name, c.Application.Name, err)
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot commit transaction:  %s", err)
		return sdk.ErrUnknownError
	}

	if err := sanity.CheckProjectPipelines(db, p); err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot check warnings: %s", err)
		return err
	}

	if err := sanity.CheckApplication(db, p, c.Application); err != nil {
		log.Warning("updateVariableInApplicationHandler: Cannot check application sanity: %s", err)
		return err
	}

	return nil
}

func updateVariableInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	p, err := project.Load(db, key, c.User, project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
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

	app, err := application.LoadByName(db, key, appName, c.User, application.LoadOptions.Default)
	if err != nil {
		return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot load application: %s", appName)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot create transaction")
	}
	defer tx.Rollback()

	if err := application.UpdateVariable(tx, app, &newVar, c.User); err != nil {
		return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot update variable %s for application %s", varName, appName)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot commit transaction")
	}

	if err := sanity.CheckProjectPipelines(db, p); err != nil {
		return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot check warnings")
	}

	app.Variable, err = application.GetAllVariableByID(db, app.ID)
	if err != nil {
		return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot load variables")
	}

	if err := sanity.CheckApplication(db, p, app); err != nil {
		return sdk.WrapError(err, "updateVariableInApplicationHandler: Cannot check application sanity")
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	return WriteJSON(w, r, app, http.StatusOK)
}

func addVariableInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	varName := vars["name"]

	p, err := project.Load(db, key, c.User, project.LoadOptions.Default, project.LoadOptions.WithEnvironments)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot load %s: %s", key, err)
		return err
	}

	var newVar sdk.Variable
	if err := UnmarshalBody(r, &newVar); err != nil {
		return err
	}

	if newVar.Name != varName {
		return sdk.ErrWrongRequest
	}

	app, err := application.LoadByName(db, key, appName, c.User, application.LoadOptions.Default)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot load application %s :  %s", appName, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot start transaction:  %s", err)
		return err
	}
	defer tx.Rollback()

	switch newVar.Type {
	case sdk.KeyVariable:
		err = application.AddKeyPairToApplication(tx, app, newVar.Name, c.User)
		break
	default:
		err = application.InsertVariable(tx, app, newVar, c.User)
		break
	}
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot add variable %s in application %s:  %s", varName, appName, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot commit transaction:  %s", err)
		return err
	}

	if err := sanity.CheckProjectPipelines(db, p); err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot check warnings: %s", err)
		return err
	}

	app.Variable, err = application.GetAllVariableByID(db, app.ID)
	if err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot get variables: %s", err)
		return err
	}

	if err := sanity.CheckApplication(db, p, app); err != nil {
		log.Warning("addVariableInApplicationHandler: Cannot check application sanity: %s", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	return WriteJSON(w, r, app, http.StatusOK)
}
