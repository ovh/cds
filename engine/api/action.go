package api

import (
	"context"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// getActionsHandler Retrieve all public actions
// @title List all public actions
func (api *API) getActionsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		acts, err := action.LoadActions(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "GetActions: Cannot load action from db")
		}
		return WriteJSON(w, acts, http.StatusOK)
	}
}

func (api *API) getPipelinesUsingActionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get action name in URL
		vars := mux.Vars(r)
		name := vars["actionName"]
		response, err := action.GetPipelineUsingAction(api.mustDB(), name)
		if err != nil {
			return err
		}
		return WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) getActionsRequirements() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		req, err := action.LoadAllBinaryRequirements(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getActionsRequirements> Cannot load action requirements")
		}
		return WriteJSON(w, req, http.StatusOK)
	}
}

func (api *API) deleteActionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		// Get action name in URL
		vars := mux.Vars(r)
		name := vars["permActionName"]

		a, errLoad := action.LoadPublicAction(api.mustDB(), name)
		if errLoad != nil {
			if errLoad != sdk.ErrNoAction {
				log.Warning("deleteAction> Cannot load action %s: %T %s", name, errLoad, errLoad)
			}
			return errLoad
		}

		used, errUsed := action.Used(api.mustDB(), a.ID)
		if errUsed != nil {
			return errUsed
		}
		if used {
			return sdk.WrapError(sdk.ErrForbidden, "deleteAction> Cannot delete action %s: used in pipelines", name)
		}

		tx, errbegin := api.mustDB().Begin()
		if errbegin != nil {
			return sdk.WrapError(errbegin, "deleteAction> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := action.DeleteAction(tx, a.ID, getUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "deleteAction> Cannot delete action %s", name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteAction> Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) updateActionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get action name in URL
		vars := mux.Vars(r)
		name := vars["permActionName"]

		// Get body
		var a sdk.Action
		if err := UnmarshalBody(r, &a); err != nil {
			return err
		}

		// Check that action  already exists
		actionDB, err := action.LoadPublicAction(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "updateAction> Cannot check if action %s exist", a.Name)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateAction> Cannot begin tx")
		}
		defer tx.Rollback()

		a.ID = actionDB.ID

		if err = action.UpdateActionDB(tx, &a, getUser(ctx).ID); err != nil {
			return sdk.WrapError(err, "updateAction: Cannot update action")
		}

		if err = tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateAction> Cannot commit transaction")
		}

		return WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) addActionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var a sdk.Action
		if err := UnmarshalBody(r, &a); err != nil {
			return err
		}

		// Check that action does not already exists
		conflict, errConflict := action.Exists(api.mustDB(), a.Name)
		if errConflict != nil {
			return errConflict
		}

		if conflict {
			return sdk.WrapError(sdk.ErrConflict, "addAction> Action %s already exists", a.Name)
		}

		tx, errDB := api.mustDB().Begin()
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

		return WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getActionAuditHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		actionIDString := vars["actionID"]

		actionID, err := strconv.Atoi(actionIDString)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getActionAuditHandler> ActionID must be a number, got %s: %s", actionIDString, err)
		}
		// Load action
		a, err := action.LoadAuditAction(api.mustDB(), actionID, true)
		if err != nil {
			return sdk.WrapError(err, "getActionAuditHandler> Cannot load audit for action %s", actionID)
		}
		return WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getActionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permActionName"]

		a, err := action.LoadPublicAction(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getActionHandler> Cannot load action: %s", err)
		}
		return WriteJSON(w, a, http.StatusOK)
	}
}

// importActionHandler insert OR update an existing action.
func (api *API) importActionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

		tx, errbegin := api.mustDB().Begin()
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
			if err := action.UpdateActionDB(tx, a, getUser(ctx).ID); err != nil {
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

		return WriteJSON(w, a, code)
	}
}
