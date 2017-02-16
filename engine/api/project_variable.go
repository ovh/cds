package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getVariablesAuditInProjectnHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]

	audits, err := project.GetVariableAudit(db, key)
	if err != nil {
		log.Warning("getVariablesAuditInProjectnHandler: Cannot get variable audit for project %s: %s\n", key, err)
		return err

	}
	return WriteJSON(w, r, audits, http.StatusOK)
}

func restoreProjectVariableAuditHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	auditIDString := vars["auditID"]

	auditID, err := strconv.ParseInt(auditIDString, 10, 64)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot parse auditID %s: %s\n", auditIDString, err)
		return sdk.ErrInvalidID

	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot load %s: %s\n", key, err)
		return err

	}

	variables, err := project.GetAudit(db, key, auditID)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot get variable audit for project %s: %s\n", key, err)
		return err

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot start transaction : %s\n", err)
		return sdk.ErrUnknownError

	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot create audit: %s\n", err)
		return err

	}

	err = project.DeleteAllVariableFromProject(tx, p.ID)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot delete variables for project %s:  %s\n", key, err)
		return sdk.ErrUnknownError

	}

	for _, v := range variables {
		if sdk.NeedPlaceholder(v.Type) {
			value, err := secret.Decrypt([]byte(v.Value))
			if err != nil {
				log.Warning("restoreProjectVariableAuditHandler: Cannot decrypt variable %s for project %s:  %s\n", v.Name, key, err)
				return err

			}
			v.Value = string(value)
		}
		err := project.InsertVariableInProject(tx, p, v)
		if err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot insert variable %s for project %s:  %s\n", v.Name, key, err)
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot commit transaction:  %s\n", err)
		return sdk.ErrUnknownError
	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot check warnings: %s\n", err)
		return err
	}

	return nil
}

func getVariablesInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, err := project.Load(db, key, c.User, project.WithVariables())
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot load %s: %s\n", key, err)
		return sdk.ErrNotFound
	}

	return WriteJSON(w, r, p.Variable, http.StatusOK)
}

func deleteVariableFromProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	varName := vars["name"]

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot load %s: %s\n", key, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("deleteVariableFromProject: cannot create audit for project variable: %s\n", err)
		return err
	}

	err = project.DeleteVariableFromProject(tx, p, varName)
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot delete %s: %s\n", varName, err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot commit transaction: %s\n", err)
		return err
	}

	p.Variable, err = project.GetAllVariableInProject(db, p.ID)
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot load all variables: %s\n", err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func updateVariablesInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot read body: %s\n", err)
		return sdk.ErrWrongRequest

	}

	var projectVars []sdk.Variable
	err = json.Unmarshal(data, &projectVars)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot unmarshal body: %s\n", err)
		return sdk.ErrWrongRequest

	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot load %s: %s\n", key, err)
		return sdk.ErrNotFound

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot start transaction: %s\n", err)
		return sdk.ErrNotFound

	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: cannot create audit for project variable: %s\n", err)
		return err

	}

	// Preload values, if one password variable has a password placeholder, we can't just insert
	// the placeholder !
	preload, err := project.GetAllVariableInProject(tx, p.ID, project.WithClearPassword())
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot preload variables values: %s\n", err)
		return err

	}

	err = project.DeleteAllVariableFromProject(tx, p.ID)
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot delete all variables for project %s: %s\n", p.Key, err)
		return sdk.ErrNotFound

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
			err = project.InsertVariableInProject(tx, p, v)
			if err != nil {
				log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
				return err

			}
			break
		// In case of a key variable, if empty, generate a pair and add them as variable
		case sdk.KeyVariable:
			if v.Value == "" {
				err := project.AddKeyPairToProject(tx, p, v.Name)
				if err != nil {
					log.Warning("updateVariablesInProjectHandler> cannot generate keypair: %s\n", err)
					return err

				}
			} else if v.Value == sdk.PasswordPlaceholder {
				for _, p := range preload {
					if p.ID == v.ID {
						v.Value = p.Value
					}
				}
				err = project.InsertVariableInProject(tx, p, v)
				if err != nil {
					log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
					return err

				}
			}
			break
		default:
			err = project.InsertVariableInProject(tx, p, v)
			if err != nil {
				log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s\n", v.Name, p.Key, err)
				return err

			}
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot commit transaction: %s\n", err)
		return sdk.ErrNotFound

	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("updateVariablesInApplicationHandler: Cannot check warnings: %s\n", err)
		return err

	}

	return nil
}

func updateVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	varName := vars["name"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	var newVar sdk.Variable
	err = json.Unmarshal(data, &newVar)
	if err != nil {
		return sdk.ErrWrongRequest

	}
	if newVar.Name != varName {
		return sdk.ErrWrongRequest

	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("updateVariableInProject: Cannot load %s: %s\n", key, err)
		return err

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariableInProject: cannot start transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("updateVariableInProject: cannot create audit for project variable: %s\n", err)
		return err

	}

	err = project.UpdateVariableInProject(tx, p, newVar)
	if err != nil {
		log.Warning("updateVariableInProject: Cannot update variable %s in project %s:  %s\n", varName, p.Name, err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateVariableInProject: cannot commit transaction: %s\n", err)
		return err

	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("updateVariableInProject: Cannot check warnings: %s\n", err)
		return err

	}

	p.Variable, err = project.GetAllVariableInProject(db, p.ID)
	if err != nil {
		log.Warning("updateVariableInProject: Cannot get all variables: %s\n", err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func getVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	varName := vars["name"]

	proj, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("getVariableInProjectHandler: Cannot load project %s: %s\n", key, err)
		return err

	}

	v, err := project.GetVariableInProject(db, proj.ID, varName)
	if err != nil {
		log.Warning("getVariableInProjectHandler: Cannot get variable %s in project %s: %s\n", varName, key, err)
		return err

	}

	return WriteJSON(w, r, v, http.StatusOK)
}

func addVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	varName := vars["name"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	var newVar sdk.Variable
	err = json.Unmarshal(data, &newVar)
	if err != nil {
		return sdk.ErrWrongRequest

	}
	if newVar.Name != varName {
		return sdk.ErrWrongRequest

	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot load %s: %s\n", key, err)
		return err
	}

	varInProject, err := project.CheckVariableInProject(db, p.ID, varName)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot check if variable %s is already in the project %s: %s\n", varName, p.Name, err)
		return err
	}

	if varInProject {
		return sdk.ErrVariableExists
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addVariableInProjectHandler: cannot begin tx: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = project.CreateAudit(tx, p, c.User)
	if err != nil {
		log.Warning("addVariableInProjectHandler: cannot create audit for project variable: %s\n", err)
		return err
	}

	switch newVar.Type {
	case sdk.KeyVariable:
		err = project.AddKeyPairToProject(tx, p, newVar.Name)
		break
	default:
		err = project.InsertVariableInProject(tx, p, newVar)
		break
	}
	if err != nil {
		log.Warning("AddVariableInProject: Cannot add variable %s in project %s:  %s\n", varName, p.Name, err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addVariableInProjectHandler: cannot commit tx: %s\n", err)
		return err

	}

	err = sanity.CheckProjectPipelines(db, p)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot check warnings: %s\n", err)
		return err

	}

	p.Variable, err = project.GetAllVariableInProject(db, p.ID)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot get variables: %s\n", err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)
}
