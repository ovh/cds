package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func getWorkflowHookModelsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	m, err := workflow.LoadHookModels(db)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowHookModelsHandler")
	}
	return WriteJSON(w, r, m, http.StatusOK)
}

func getWorkflowHookModelHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	name := vars["model"]
	m, err := workflow.LoadHookModelByName(db, name)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowHookModelHandler")
	}
	return WriteJSON(w, r, m, http.StatusOK)
}

func postWorkflowHookModelHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	m := &sdk.WorkflowHookModel{}
	if err := UnmarshalBody(r, m); err != nil {
		return sdk.WrapError(err, "postWorkflowHookModelHandler")
	}

	tx, errtx := db.Begin()
	if errtx != nil {
		return sdk.WrapError(errtx, "postWorkflowHookModelHandler> Unable to start transaction")
	}
	defer tx.Rollback()

	if err := workflow.InsertHookModel(tx, m); err != nil {
		return sdk.WrapError(err, "postWorkflowHookModelHandler")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowHookModelHandler> Unable to commit transaction")
	}

	return WriteJSON(w, r, m, http.StatusCreated)
}

func putWorkflowHookModelHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	m := &sdk.WorkflowHookModel{}
	if err := UnmarshalBody(r, m); err != nil {
		return err
	}

	tx, errtx := db.Begin()
	if errtx != nil {
		return sdk.WrapError(errtx, "putWorkflowHookModelHandler> Unable to start transaction")
	}

	defer tx.Rollback()

	if err := workflow.UpdateHookModel(tx, m); err != nil {
		return sdk.WrapError(err, "putWorkflowHookModelHandler")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(errtx, "putWorkflowHookModelHandler> Unable to commit transaction")
	}

	return WriteJSON(w, r, m, http.StatusOK)
}
