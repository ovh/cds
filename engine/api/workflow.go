package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// getWorkflowsHandler returns ID and name of workflows for a given project/user
func getWorkflowsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	ws, err := workflow.LoadAll(db, key)
	if err != nil {
		return err
	}

	return WriteJSON(w, r, ws, http.StatusOK)
}

// getWorkflowHandler returns a full workflow
func getWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	w1, err := workflow.Load(db, key, name, c.User)
	if err != nil {
		return err
	}
	return WriteJSON(w, r, w1, http.StatusOK)
}

// postWorkflowHandler create a new workflow
func postWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "Cannot load Project %s", key)
	}
	var wf sdk.Workflow
	if err := UnmarshalBody(r, &wf); err != nil {
		return sdk.WrapError(err, "Cannot read body")
	}
	wf.ProjectID = p.ID
	wf.ProjectKey = key

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "Cannot start transaction")
	}
	defer tx.Rollback()

	if err := workflow.Insert(tx, &wf, c.User); err != nil {
		return sdk.WrapError(err, "Cannot insert workflow")
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "Cannot commit transaction")
	}

	wf1, errl := workflow.LoadByID(db, wf.ID, c.User)
	if errl != nil {
		return sdk.WrapError(errl, "Cannot load workflow")
	}

	return WriteJSON(w, r, wf1, http.StatusCreated)
}

// putWorkflowHandler updates a workflow
func putWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	p, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "Cannot load Project %s", key)
	}

	oldW, errW := workflow.Load(db, key, name, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "Cannot load Workflow %s", key)
	}

	var wf sdk.Workflow
	if err := UnmarshalBody(r, &wf); err != nil {
		return sdk.WrapError(err, "Cannot read body")
	}
	wf.ID = oldW.ID
	wf.RootID = oldW.RootID
	wf.Root.ID = oldW.RootID
	wf.ProjectID = p.ID
	wf.ProjectKey = key

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "Cannot start transaction")
	}
	defer tx.Rollback()

	if err := workflow.Update(tx, &wf, oldW, c.User); err != nil {
		return sdk.WrapError(err, "Cannot update workflow")
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "Cannot commit transaction")
	}

	wf1, errl := workflow.LoadByID(db, wf.ID, c.User)
	if errl != nil {
		return sdk.WrapError(errl, "Cannot load workflow")
	}

	return WriteJSON(w, r, wf1, http.StatusOK)
}

// putWorkflowHandler deletes a workflow
func deleteWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	p, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "Cannot load Project %s", key)
	}

	oldW, errW := workflow.Load(db, key, name, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "Cannot load Workflow %s", key)
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "Cannot start transaction")
	}
	defer tx.Rollback()

	if err := workflow.Delete(tx, oldW, c.User); err != nil {
		return sdk.WrapError(err, "Cannot delete workflow")
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(errT, "Cannot commit transaction")
	}
	return WriteJSON(w, r, nil, http.StatusOK)
}

func getWorkflowNodeRunArtifactsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	number, errNu := requestVarInt(r, "number")
	if errNu != nil {
		return sdk.WrapError(errNu, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
	}

	id, errI := requestVarInt(r, "id")
	if errI != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
	}
	nodeRun, errR := workflow.LoadNodeRun(db, key, name, number, id)
	if errR != nil {
		return sdk.WrapError(errR, "getWorkflowJobArtifactsHandler> Cannot load node run")
	}

	return WriteJSON(w, r, nodeRun.Artifacts, http.StatusOK)
}
