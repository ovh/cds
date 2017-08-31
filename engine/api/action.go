package api

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getActionsHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		acts, err := action.LoadActions(db)
		if err != nil {
			return sdk.WrapError(err, "GetActions: Cannot load action from db")
		}
		return WriteJSON(w, r, acts, http.StatusOK)
	}
}

func getPipelinesUsingActionHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
		rows, errq := db.Query(query, name)
		if errq != nil {
			return sdk.WrapError(errq, "getPipelinesUsingActionHandler> Cannot load pipelines using action %s", name)
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
			if err := rows.Scan(&a.ActionType, &a.ActionName, &a.ActionID, &stageID, &pipName, &appName, &projName, &projKey); err != nil {
				return sdk.WrapError(err, "getPipelinesUsingActionHandler> Cannot read sql response")
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
}

func getActionsRequirements(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		req, err := action.LoadAllBinaryRequirements(db)
		if err != nil {
			return sdk.WrapError(err, "getActionsRequirements> Cannot load action requirements")
		}
		return WriteJSON(w, r, req, http.StatusOK)
	}
}

func deleteActionHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		// Get action name in URL
		vars := mux.Vars(r)
		name := vars["permActionName"]

		a, errLoad := action.LoadPublicAction(db, name)
		if errLoad != nil {
			if errLoad != sdk.ErrNoAction {
				log.Warning("deleteAction> Cannot load action %s: %T %s", name, errLoad, errLoad)
			}
			return errLoad
		}

		used, errUsed := action.Used(db, a.ID)
		if errUsed != nil {
			return errUsed
		}
		if used {
			return sdk.WrapError(sdk.ErrForbidden, "deleteAction> Cannot delete action %s: used in pipelines", name)
		}

		tx, errbegin := db.Begin()
		if errbegin != nil {
			log.Warning("deleteAction> Cannot start transaction: %s\n", errbegin)
			return errbegin
		}
		defer tx.Rollback()

		if err := action.DeleteAction(tx, a.ID, c.User.ID); err != nil {
			return sdk.WrapError(err, "deleteAction> Cannot delete action %s", name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteAction> Cannot commit transaction")
		}

		return nil
	}
}

func updateActionHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
			return sdk.WrapError(err, "updateAction> Cannot check if action %s exist", a.Name)
		}

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "updateAction> Cannot begin tx")
		}
		defer tx.Rollback()

		a.ID = actionDB.ID

		if err = action.UpdateActionDB(tx, &a, c.User.ID); err != nil {
			return sdk.WrapError(err, "updateAction: Cannot update action")
		}

		if err = tx.Commit(); err != nil {
			log.Warning("updateAction> Cannot commit transaction: %s\n", err)
			return err
		}

		return WriteJSON(w, r, a, http.StatusOK)
	}
}

func addActionHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
			return sdk.WrapError(sdk.ErrConflict, "addAction> Action %s already exists", a.Name)
		}

		tx, errDB := db.Begin()
		if errDB != nil {
			return errDB
		}
		defer tx.Rollback()

		a.Type = sdk.DefaultAction
		if err := action.InsertAction(tx, &a, true); err != nil {
			return sdk.WrapError(err, "Action: Cannot insert action")
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return WriteJSON(w, r, a, http.StatusOK)
	}
}

func getActionAuditHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		vars := mux.Vars(r)
		actionIDString := vars["actionID"]

		actionID, err := strconv.Atoi(actionIDString)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getActionAuditHandler> ActionID must be a number, got %s: %s", actionIDString, err)
		}
		// Load action
		a, err := action.LoadAuditAction(db, actionID, true)
		if err != nil {
			return sdk.WrapError(err, "getActionAuditHandler> Cannot load audit for action %s", actionID)
		}
		return WriteJSON(w, r, a, http.StatusOK)
	}
}

func getActionHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
		vars := mux.Vars(r)
		name := vars["permActionName"]

		a, err := action.LoadPublicAction(db, name)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getActionHandler> Cannot load action: %s", err)
		}
		return WriteJSON(w, r, a, http.StatusOK)
	}
}

// importActionHandler insert OR update an existing action.
func importActionHandler(router *Router) Handler {
	return func(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
				return sdk.WrapError(sdk.ErrWrongRequest, "importActionHandler> Cannot load file uploaded: %s", errUpload)
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
}
