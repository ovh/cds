package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// getActionsHandler Retrieve all public actions
// @title List all public actions
func (api *API) getActionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		acts, err := action.LoadActions(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "GetActions: Cannot load action from db")
		}
		return service.WriteJSON(w, acts, http.StatusOK)
	}
}

func (api *API) getPipelinesUsingActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get action name in URL
		vars := mux.Vars(r)
		name := vars["actionName"]
		response, err := action.GetPipelineUsingAction(api.mustDB(), name)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) getActionsRequirements() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		req, err := action.LoadAllBinaryRequirements(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getActionsRequirements> Cannot load action requirements")
		}
		return service.WriteJSON(w, req, http.StatusOK)
	}
}

func (api *API) deleteActionHandler() service.Handler {
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

		event.PublishActionDelete(*a, getUser(ctx))

		return nil
	}
}

func (api *API) updateActionHandler() service.Handler {
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

		event.PublishActionUpdate(*actionDB, a, getUser(ctx))

		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) addActionHandler() service.Handler {
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

		event.PublishActionAdd(a, getUser(ctx))

		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getActionAuditHandler() service.Handler {
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
		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permActionName"]

		a, err := action.LoadPublicAction(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getActionHandler> Cannot load action: %s", err)
		}
		return service.WriteJSON(w, a, http.StatusOK)
	}
}

func (api *API) getActionExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["permActionName"]
		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "getActionExportHandler> Format invalid")
		}

		if _, err := action.Export(api.mustDB(), name, f, w); err != nil {
			return sdk.WrapError(err, "getActionExportHandler>")
		}
		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		return nil
	}
}

// importActionHandler insert OR update an existing action.
func (api *API) importActionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var a *sdk.Action

		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return errRead
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		var ea = new(exportentities.Action)
		var errapp error
		switch contentType {
		case "application/json":
			errapp = json.Unmarshal(data, ea)
		case "application/x-yaml", "text/x-yam":
			errapp = yaml.Unmarshal(data, ea)
		default:
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unsupported content-type: %s", contentType))
		}

		if errapp != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errapp)
		}

		a, errapp = ea.Action()
		if errapp != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errapp)
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
			a.Type = sdk.DefaultAction
			if err := action.InsertAction(tx, a, true); err != nil {
				return err
			}
			code = 201
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		if exist {
			event.PublishActionUpdate(*existingAction, *a, getUser(ctx))
		} else {
			event.PublishActionAdd(*a, getUser(ctx))
		}

		return service.WriteJSON(w, a, code)
	}
}
