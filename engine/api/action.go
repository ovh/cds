package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getActionsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	acts, err := action.LoadActions(db)
	if err != nil {
		log.Warning("GetActions: Cannot load action from db: %s\n", err)
		return err
	}
	return WriteJSON(w, r, acts, http.StatusOK)
}

func getPipelinesUsingActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get action name in URL
	vars := mux.Vars(r)
	name := vars["actionName"]

	query := `
		SELECT
			action.type, action.name as actionName, action.id as actionId,
			pipeline_stage.id as stageId,
			pipeline.name as pipName, application.name as appName, project.name, project.projectkey
		FROM action_edge
		LEFT JOIN action on action.id = parent_id
		LEFT OUTER JOIN pipeline_action ON pipeline_action.action_id = action.id
		LEFT OUTER JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
		LEFT OUTER JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
		LEFT OUTER JOIN application_pipeline ON application_pipeline.pipeline_id = pipeline.id
		LEFT OUTER JOIN application ON application.id = application_pipeline.application_id
		LEFT OUTER JOIN project ON pipeline.project_id = project.id
		LEFT JOIN action as actionChild ON  actionChild.id = child_id
		WHERE actionChild.name = $1 and actionChild.public = true
		ORDER BY projectkey, appName, pipName, actionName;
	`
	rows, err := db.Query(query, name)
	if err != nil {
		log.Warning("getPipelinesUsingActionHandler> Cannot load pipelines using action %s: %s\n", name, err)
		return err
	}
	defer rows.Close()

	type pipelineUsingAction struct {
		ActionID   int    `json:"action_id"`
		ActionType string `json:"type"`
		ActionName string `json:"action_name"`
		PipName    string `json:"pipeline_name"`
		AppName    string `json:"application_name"`
		ProjName   string `json:"project_name"`
		ProjKey    string `json:"key"`
		StageID    int64  `json:"stage_id"`
	}

	response := []pipelineUsingAction{}

	for rows.Next() {
		var a pipelineUsingAction
		var pipName, appName, projName, projKey sql.NullString
		var stageID sql.NullInt64
		err = rows.Scan(&a.ActionType, &a.ActionName, &a.ActionID, &stageID, &pipName, &appName, &projName, &projKey)
		if err != nil {
			log.Warning("getPipelinesUsingActionHandler> Cannot read sql response: %s\n", err)
			return err
		}
		if stageID.Valid {
			a.StageID = stageID.Int64
		}
		if pipName.Valid {
			a.PipName = pipName.String
		}
		if appName.Valid {
			a.AppName = appName.String
		}
		if projName.Valid {
			a.ProjName = projName.String
		}
		if projKey.Valid {
			a.ProjKey = projKey.String
		}

		response = append(response, a)
	}
	return WriteJSON(w, r, response, http.StatusOK)
}

func getActionsRequirements(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	req, err := action.LoadAllBinaryRequirements(db)
	if err != nil {
		log.Warning("getActionsRequirements> Cannot load action requirements: %s\n", err)
		return err
	}
	return WriteJSON(w, r, req, http.StatusOK)
}

func deleteActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get action name in URL
	vars := mux.Vars(r)
	name := vars["permActionName"]

	a, errLoad := action.LoadPublicAction(db, name)
	if errLoad != nil {
		if errLoad != sdk.ErrNoAction {
			log.Warning("deleteAction> Cannot load action %s: %T %s\n", name, errLoad, errLoad)
		}
		return errLoad
	}

	used, errUsed := action.Used(db, a.ID)
	if errUsed != nil {
		return errUsed
	}
	if used {
		log.Warning("deleteAction> Cannot delete action %s: used in pipelines\n", name)
		return sdk.ErrForbidden
	}

	tx, errbegin := db.Begin()
	if errbegin != nil {
		log.Warning("deleteAction> Cannot start transaction: %s\n", errbegin)
		return errbegin
	}
	defer tx.Rollback()

	if err := action.DeleteAction(tx, a.ID, c.User.ID); err != nil {
		log.Warning("deleteAction> Cannot delete action %s: %s\n", name, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteAction> Cannot commit transaction: %s\n", err)
		return err
	}

	return nil
}

func updateActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get action name in URL
	vars := mux.Vars(r)
	name := vars["permActionName"]

	// Get body
	var a sdk.Action
	if err := UnmarshalBody(r, &a); err != nil {
		return err
	}

	// Check that action  already exists
	//actionDB, err := action.LoadPublicAction(db, name, action.WithClearPasswords())
	actionDB, err := action.LoadPublicAction(db, name)
	if err != nil {
		log.Warning("updateAction> Cannot check if action %s exist: %s\n", a.Name, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateAction> Cannot begin tx: %s\n", err)
		return err
	}
	defer tx.Rollback()

	a.ID = actionDB.ID

	if err = action.UpdateActionDB(tx, &a, c.User.ID); err != nil {
		log.Warning("updateAction: Cannot update action: %s\n", err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Warning("updateAction> Cannot commit transaction: %s\n", err)
		return err
	}

	return WriteJSON(w, r, a, http.StatusOK)
}

func addActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var a sdk.Action
	if err := UnmarshalBody(r, &a); err != nil {
		return err
	}

	// Check that action does not already exists
	conflict, errConflict := action.Exists(db, a.Name)
	if errConflict != nil {
		return errConflict
	}

	if conflict {
		log.Warning("addAction> Action %s already exists\n", a.Name)
		return sdk.ErrConflict
	}

	tx, errDB := db.Begin()
	if errDB != nil {
		return errDB
	}
	defer tx.Rollback()

	a.Type = sdk.DefaultAction
	if err := action.InsertAction(tx, &a, true); err != nil {
		log.Warning("Action: Cannot insert action: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return WriteJSON(w, r, a, http.StatusOK)
}

func getActionAuditHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get action name in URL
	vars := mux.Vars(r)
	actionIDString := vars["actionID"]

	actionID, err := strconv.Atoi(actionIDString)
	if err != nil {
		log.Warning("getActionAuditHandler> ActionID must be a number, got %s: %s\n", actionIDString, err)
		return sdk.ErrInvalidID
	}
	// Load action
	a, err := action.LoadAuditAction(db, actionID, true)
	if err != nil {
		log.Warning("getActionAuditHandler> Cannot load audit for action %s: %s\n", actionID, err)
		return err
	}
	return WriteJSON(w, r, a, http.StatusOK)
}

func getActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get action name in URL
	vars := mux.Vars(r)
	name := vars["permActionName"]

	// Load action
	a, err := action.LoadPublicAction(db, name)
	if err != nil {
		log.Warning("getActionHandler> Cannot load action: %s\n", err)
		return sdk.ErrNotFound
	}
	return WriteJSON(w, r, a, http.StatusOK)
}

// importActionHandler insert OR update an existing action.
func importActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var a *sdk.Action
	url := r.Form.Get("url")
	//Load action from url
	if url != "" {
		var errnew error
		a, errnew = sdk.NewActionFromRemoteScript(url, nil)
		if errnew != nil {
			return errnew
		}
	} else if r.Header.Get("content-type") == "multipart/form-data" {
		//Try to load from the file
		r.ParseMultipartForm(64 << 20)
		file, _, errUpload := r.FormFile("UploadFile")
		if errUpload != nil {
			log.Warning("importActionHandler> Cannot load file uploaded: %s\n", errUpload)
			return sdk.ErrWrongRequest
		}
		btes, errRead := ioutil.ReadAll(file)
		if errRead != nil {
			return errRead
		}

		var errnew error
		a, errnew = sdk.NewActionFromScript(btes)
		if errnew != nil {
			return errnew
		}
	} else { // a jsonified action is posted in body
		if err := UnmarshalBody(r, &a); err != nil {
			return err
		}
	}

	if a == nil {
		return sdk.ErrWrongRequest
	}

	tx, errbegin := db.Begin()
	if errbegin != nil {
		return errbegin
	}

	defer tx.Rollback()

	//Check if action exists
	exist := false
	existingAction, errload := action.LoadPublicAction(tx, a.Name)
	if errload == nil {
		exist = true
		a.ID = existingAction.ID
	}

	//http code status
	var code int

	//Update or Insert the action
	if exist {
		if err := action.UpdateActionDB(tx, a, c.User.ID); err != nil {
			return err
		}
		code = 200
	} else {
		a.Enabled = true
		a.Type = sdk.DefaultAction
		if err := action.InsertAction(tx, a, true); err != nil {
			return err
		}
		code = 201
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return WriteJSON(w, r, a, code)
}
