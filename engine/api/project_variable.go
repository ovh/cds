package main

import (
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getVariablesAuditInProjectnHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]

	audits, err := project.GetVariableAudit(db, key)
	if err != nil {
		log.Warning("getVariablesAuditInProjectnHandler: Cannot get variable audit for project %s: %s", key, err)
		return err

	}
	return WriteJSON(w, r, audits, http.StatusOK)
}

// Deprecated
func restoreProjectVariableAuditHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	auditIDString := vars["auditID"]

	auditID, err := strconv.ParseInt(auditIDString, 10, 64)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot parse auditID %s: %s", auditIDString, err)
		return sdk.ErrInvalidID

	}

	p, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot load %s: %s", key, err)
		return err

	}

	variables, err := project.GetAudit(db, key, auditID)
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot get variable audit for project %s: %s", key, err)
		return err

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot start transaction : %s", err)
		return sdk.ErrUnknownError

	}
	defer tx.Rollback()

	if err := project.DeleteAllVariable(tx, p.ID); err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot delete variables for project %s:  %s", key, err)
		return sdk.ErrUnknownError

	}

	for _, v := range variables {
		if sdk.NeedPlaceholder(v.Type) {
			value, err := secret.Decrypt([]byte(v.Value))
			if err != nil {
				log.Warning("restoreProjectVariableAuditHandler: Cannot decrypt variable %s for project %s:  %s", v.Name, key, err)
				return err

			}
			v.Value = string(value)
		}
		if err := project.InsertVariable(tx, p, &v, c.User); err != nil {
			log.Warning("restoreProjectVariableAuditHandler: Cannot insert variable %s for project %s:  %s", v.Name, key, err)
			return err
		}
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot update last modified:  %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot commit transaction:  %s", err)
		return err
	}

	if err := sanity.CheckProjectPipelines(db, p); err != nil {
		log.Warning("restoreProjectVariableAuditHandler: Cannot check warnings: %s", err)
		return err
	}

	return nil
}

func getVariablesInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, err := project.Load(db, key, c.User, project.LoadOptions.WithVariables)
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot load %s: %s", key, err)
		return sdk.ErrNotFound
	}

	return WriteJSON(w, r, p.Variable, http.StatusOK)
}

func deleteVariableFromProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	varName := vars["name"]

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteVariableFromProject: Cannot start transaction: %s", err)
		return err
	}
	defer tx.Rollback()

	varToDelete, errV := project.GetVariableInProject(db, c.Project.ID, varName)
	if errV != nil {
		return sdk.WrapError(errV, "deleteVariableFromProject> Cannot load variable %s", varName)
	}

	if err := project.DeleteVariable(tx, c.Project, varToDelete, c.User); err != nil {
		log.Warning("deleteVariableFromProject: Cannot delete %s: %s", varName, err)
		return err
	}

	if err := project.UpdateLastModified(tx, c.User, c.Project); err != nil {
		log.Warning("deleteVariableFromProject: Cannot update last modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteVariableFromProject: Cannot commit transaction: %s", err)
		return err
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

//DEPRECATED
func updateVariablesInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var projectVars []sdk.Variable
	if err := UnmarshalBody(r, &projectVars); err != nil {
		return err
	}

	tx, errtx := db.Begin()
	if errtx != nil {
		return sdk.WrapError(errtx, "updateVariablesInProjectHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	// Preload values, if one password variable has a password placeholder, we can't just insert
	// the placeholder !
	preload, errpre := project.GetAllVariableInProject(tx, c.Project.ID, project.WithClearPassword())
	if errpre != nil {
		return sdk.WrapError(errpre, "updateVariablesInProjectHandler: Cannot preload variables values")
	}

	if err := project.DeleteAllVariable(tx, c.Project.ID); err != nil {
		return sdk.WrapError(err, "updateVariablesInProjectHandler: Cannot delete all variables for project")
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
			if err := project.InsertVariable(tx, c.Project, &v, c.User); err != nil {
				return sdk.WrapError(err, "updateVariablesInProjectHandler: Cannot insert variable %s", v.Name)
			}
			break
		// In case of a key variable, if empty, generate a pair and add them as variable
		case sdk.KeyVariable:
			if v.Value == "" {
				if err := project.AddKeyPair(tx, c.Project, v.Name, c.User); err != nil {
					log.Warning("updateVariablesInProjectHandler> cannot generate keypair: %s", err)
					return err

				}
			} else if v.Value == sdk.PasswordPlaceholder {
				for _, p := range preload {
					if p.ID == v.ID {
						v.Value = p.Value
					}
				}

				if err := project.InsertVariable(tx, c.Project, &v, c.User); err != nil {
					log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s", v.Name, c.Project.Key, err)
					return err
				}
			}
			break
		default:
			if err := project.InsertVariable(tx, c.Project, &v, c.User); err != nil {
				log.Warning("updateVariablesInProjectHandler: Cannot insert variable %s in project %s: %s", v.Name, c.Project.Key, err)
				return err
			}

		}
	}
	if err := project.UpdateLastModified(tx, c.User, c.Project); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot commit transaction: %s", err)
		return sdk.ErrNotFound

	}

	if err := sanity.CheckProjectPipelines(db, c.Project); err != nil {
		return err

	}

	return nil
}

func updateVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	varName := vars["name"]

	var newVar sdk.Variable
	if err := UnmarshalBody(r, &newVar); err != nil {
		return err
	}
	if newVar.Name != varName {
		return sdk.ErrWrongRequest
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateVariableInProject: cannot start transaction: %s", err)
		return err

	}
	defer tx.Rollback()

	if err := project.UpdateVariable(tx, c.Project, &newVar, c.User); err != nil {
		log.Warning("updateVariableInProject: Cannot update variable %s in project %s:  %s", varName, c.Project.Name, err)
		return err
	}

	if err := project.UpdateLastModified(tx, c.User, c.Project); err != nil {
		log.Warning("updateVariableInProject: Cannot update last modified date: %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateVariableInProject: cannot commit transaction: %s", err)
		return err

	}

	if err := sanity.CheckProjectPipelines(db, c.Project); err != nil {
		log.Warning("updateVariableInProject: Cannot check warnings: %s", err)
		return err

	}
	return WriteJSON(w, r, newVar, http.StatusOK)
}

//DEPRECATED
func getVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	varName := vars["name"]

	v, err := project.GetVariableInProject(db, c.Project.ID, varName)
	if err != nil {
		log.Warning("getVariableInProjectHandler: Cannot get variable %s in project %s: %s", varName, c.Project.Key, err)
		return err
	}

	return WriteJSON(w, r, v, http.StatusOK)
}

func addVariableInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	varName := vars["name"]

	var newVar sdk.Variable
	if err := UnmarshalBody(r, &newVar); err != nil {
		return err
	}
	if newVar.Name != varName {
		return sdk.ErrWrongRequest

	}

	varInProject, err := project.CheckVariableInProject(db, c.Project.ID, varName)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot check if variable %s is already in the project %s: %s", varName, c.Project.Name, err)
		return err
	}

	if varInProject {
		return sdk.ErrVariableExists
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addVariableInProjectHandler: cannot begin tx: %s", err)
		return err
	}
	defer tx.Rollback()

	switch newVar.Type {
	case sdk.KeyVariable:
		err = project.AddKeyPair(tx, c.Project, newVar.Name, c.User)
		break
	default:
		err = project.InsertVariable(tx, c.Project, &newVar, c.User)
		break
	}
	if err != nil {
		log.Warning("AddVariableInProject: Cannot add variable %s in project %s:  %s", varName, c.Project.Name, err)
		return err
	}

	if err := project.UpdateLastModified(tx, c.User, c.Project); err != nil {
		log.Warning("updateVariablesInProjectHandler: Cannot update last modified:  %s", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addVariableInProjectHandler: cannot commit tx: %s", err)
		return err
	}

	err = sanity.CheckProjectPipelines(db, c.Project)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot check warnings: %s", err)
		return err
	}

	c.Project.Variable, err = project.GetAllVariableInProject(db, c.Project.ID)
	if err != nil {
		log.Warning("AddVariableInProject: Cannot get variables: %s", err)
		return err

	}

	return WriteJSON(w, r, c.Project, http.StatusOK)
}

func getVariableAuditInProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	varName := vars["name"]

	variable, errV := project.GetVariableInProject(db, c.Project.ID, varName)
	if errV != nil {
		return sdk.WrapError(errV, "getVariableAuditInProjectHandler> Cannot load variable %s", varName)
	}

	audits, errA := project.LoadVariableAudits(db, c.Project.ID, variable.ID)
	if errA != nil {
		return sdk.WrapError(errA, "getVariableAuditInProjectHandler> Cannot load audit for variable %s", varName)
	}
	return WriteJSON(w, r, audits, http.StatusOK)
}
