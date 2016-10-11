package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getVariablesAuditInProjectnHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]

	audits, err := project.GetVariableAudit(db, key)
	if err != nil {
		log.Warning("getVariablesAuditInProjectnHandler: Cannot get variable audit for project %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, audits, http.StatusOK)
}

func restoreProjectVariableAuditHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	auditIDString := vars["auditID"]

	auditID, err := strconv.ParseInt(auditIDString, 10, 64)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot parse auditID %s: %s\n", auditIDString, err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	variables, err := project.GetAudit(db, key, auditID)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot get variable audit for project %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot start transaction : %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot create audit: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = project.DeleteAllVariableFromProject(tx, p.ID)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot delete variables for project %s:  %s\n", key, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	for _, v := range variables {
		if sdk.NeedPlaceholder(v.Type) {
			value, err := secret.Decrypt([]byte(v.Value))
			if err != nil {
				log.Warning("restoreProjectVariableAuditHandler: Cannot decrypt variable %s for project %s:  %s\n", v.Name, key, err)
				WriteError(w, r, err)
				return
			}
			v.Value = string(value)
		}
		err := project.InsertVariableInProject(tx, p.ID, v)
		if err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot insert variable %s for project %s:  %s\n", v.Name, key, err)
			WriteError(w, r, err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func getVariablesInProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, err := project.LoadProject(db, key, c.User, project.WithVariables())
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot load %s: %s\n", key, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	WriteJSON(w, r, p.Variable, http.StatusOK)
}

func deleteVariableFromProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	varName := vars["name"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("deleteVariableFromProject: cannot create audit for project variable: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = project.DeleteVariableFromProject(tx, p.ID, varName)
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot delete %s: %s\n", varName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func updateVariablesInProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot read body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var projectVars []sdk.Variable
	err = json.Unmarshal(data, &projectVars)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot unmarshal body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot load %s: %s\n", key, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot start transaction: %s\n", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: cannot create audit for project variable: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Preload values, if one password variable has a password placeholder, we can't just insert
	// the placeholder !
	preload, err := project.GetAllVariableInProject(tx, p.ID, project.WithClearPassword())
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot preload variables values: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = project.DeleteAllVariableFromProject(tx, p.ID)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot delete all variables for project %s: %s\n", p.Key, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, v := range projectVars {
		switch v.Type {
		case sdk.SecretVariable:
			if sdk.NeedPlaceholder(v.Type) && v.Value == sdk.PasswordPlaceholder {
				for _, p := range preload {
					if p.ID == v.ID {
						v.Value = p.Value
					}
				}
			}
			err = project.InsertVariableInProject(tx, p.ID, v)
			if err != nil {
				log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
				WriteError(w, r, err)
				return
			}
			break
		// In case of a key variable, if empty, generate a pair and add them as variable
		case sdk.KeyVariable:
			if v.Value == "" {
				err := keys.AddKeyPairToProject(tx, p.ID, v.Name)
				if err != nil {
					log.Warning("updateVariablesInProjectHandler> cannot generate keypair: %s\n", err)
					WriteError(w, r, err)
					return
				}
			} else if v.Value == sdk.PasswordPlaceholder {
				for _, p := range preload {
					if p.ID == v.ID {
						v.Value = p.Value
					}
				}
				err = project.InsertVariableInProject(tx, p.ID, v)
				if err != nil {
					log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
					WriteError(w, r, err)
					return
				}
			}
			break
		default:
			err = project.InsertVariableInProject(tx, p.ID, v)
			if err != nil {
				log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
				WriteError(w, r, err)
				return
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot commit transaction: %s\n", err)
		w.WriteHeader(http.StatusNotFound)
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

func updateVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	varName := vars["name"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	var newVar sdk.Variable
	err = json.Unmarshal(data, &newVar)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	if newVar.Name != varName {
		WriteError(w, r, err)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateVariableInProject: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	varInProject, err := project.CheckVariableInProject(db, p.ID, varName)
	if err != nil {
		log.Warning("updateVariableInProject: Cannot check if variable %s is already in the project %s: %s\n", varName, p.Name, err)
		WriteError(w, r, err)
		return
	}
	if varInProject {

		tx, err := db.Begin()
		if err != nil {
			log.Warning("updateVariableInProject: cannot start transaction: %s\n", err)
			WriteError(w, r, err)
			return
		}
		defer tx.Rollback()

		err = project.CreateAudit(tx, p, c.User)
		if err != nil {
			log.Warning("updateVariableInProject: cannot create audit for project variable: %s\n", err)
			WriteError(w, r, err)
			return
		}

		err = project.UpdateVariableInProject(tx, p.ID, newVar)
		if err != nil {
			log.Warning("updateVariableInProject: Cannot update variable %s in project %s:  %s\n", varName, p.Name, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = tx.Commit()
		if err != nil {
			log.Warning("updateVariableInProject: cannot commit transaction: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("updateVariableInProject: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func addVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	varName := vars["name"]

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

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	varInProject, err := project.CheckVariableInProject(db, p.ID, varName)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot check if variable %s is already in the project %s: %s\n", varName, p.Name, err)
		WriteError(w, r, err)
		return
	}

	if varInProject {
		WriteError(w, r, sdk.ErrVariableExists)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addVariableInProjectHandler: cannot begin tx: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("addVariableInProjectHandler: cannot create audit for project variable: %s\n", err)
		WriteError(w, r, err)
		return
	}

	switch newVar.Type {
	case sdk.KeyVariable:
		err = keys.AddKeyPairToProject(tx, p.ID, newVar.Name)
		break
	default:
		err = project.InsertVariableInProject(tx, p.ID, newVar)
		break
	}
	if err != nil {
		log.Warning("AddVariableInProject: Cannot add variable %s in project %s:  %s\n", varName, p.Name, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addVariableInProjectHandler: cannot commit tx: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot check warnings: %s\n", err)
		WriteError(w, r, err)
		return
	}

}
