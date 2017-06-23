package main

import (
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"sort"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// getWorkflowsHandler returns ID and name of workflows for a given project/user
func getWorkflowsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	ws, err := workflow.LoadAll(db, key)
	if err != nil {
		return err
	}

	return WriteJSON(w, r, ws, http.StatusOK)
}

// getWorkflowHandler returns a full workflow
func getWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
func postWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
func putWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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
func deleteWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getWorkflowNodeRunArtifactsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getDownloadArtifactHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	id, errI := requestVarInt(r, "artifactId")
	if errI != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "getDownloadArtifactHandler> Invalid node job run ID")
	}

	work, errW := workflow.Load(db, key, name, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "getDownloadArtifactHandler> Cannot load workflow")
	}

	art, errA := workflow.LoadArtifactByIDs(db, work.ID, id)
	if errA != nil {
		return sdk.WrapError(errA, "getDownloadArtifactHandler> Cannot load artifacts")
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

	if err := artifact.StreamFile(w, art); err != nil {
		return sdk.WrapError(err, "Cannot stream artifact %s", art.Name)
	}
	return nil
}

func getWorkflowRunArtifactsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	number, errNu := requestVarInt(r, "number")
	if errNu != nil {
		return sdk.WrapError(errNu, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
	}

	wr, errW := workflow.LoadRun(db, key, name, number)
	if errW != nil {
		return errW
	}

	arts := []sdk.WorkflowNodeRunArtifact{}
	for _, runs := range wr.WorkflowNodeRuns {
		if len(runs) == 0 {
			continue
		}

		sort.Slice(runs, func(i, j int) bool {
			return runs[i].SubNumber > runs[j].SubNumber
		})

		arts = append(arts, runs[0].Artifacts...)
	}

	return WriteJSON(w, r, arts, http.StatusOK)
}

func getWorkflowNodeRunJobStepHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	workflowName := vars["workflowName"]
	number, errN := requestVarInt(r, "number")
	if errN != nil {
		return sdk.WrapError(errN, "getWorkflowNodeRunJobBuildLogsHandler> Number: invalid number")
	}
	nodeRunID, errNI := requestVarInt(r, "id")
	if errNI != nil {
		return sdk.WrapError(errNI, "getWorkflowNodeRunJobBuildLogsHandler> id: invalid number")
	}
	runJobID, errJ := requestVarInt(r, "runJobId")
	if errJ != nil {
		return sdk.WrapError(errJ, "getWorkflowNodeRunJobBuildLogsHandler> runJobId: invalid number")
	}
	stepOrder, errS := requestVarInt(r, "stepOrder")
	if errS != nil {
		return sdk.WrapError(errS, "getWorkflowNodeRunJobBuildLogsHandler> stepOrder: invalid number")
	}

	// Check workflow is in project
	if _, errW := workflow.Load(db, projectKey, workflowName, c.User); errW != nil {
		return sdk.WrapError(errW, "getWorkflowNodeRunJobBuildLogsHandler> Cannot find workflow %s in project %s", workflowName, projectKey)
	}

	// Check nodeRunID is link to workflow
	nodeRun, errNR := workflow.LoadNodeRun(db, projectKey, workflowName, number, nodeRunID)
	if errNR != nil {
		return sdk.WrapError(errNR, "getWorkflowNodeRunJobBuildLogsHandler> Cannot find nodeRun %d/%d for workflow %s in project %s", nodeRunID, number, workflowName, projectKey)
	}

	var stepStatus string
	// Find job/step in nodeRun
stageLoop:
	for _, s := range nodeRun.Stages {
		for _, rj := range s.RunJobs {
			if rj.ID != runJobID {
				continue
			}
			ss := rj.Job.StepStatus
			for _, sss := range ss {
				if int64(sss.StepOrder) == stepOrder {
					stepStatus = sss.Status
					break
				}
			}
			break stageLoop
		}
	}

	if stepStatus == "" {
		return sdk.WrapError(fmt.Errorf("getWorkflowNodeRunJobBuildLogsHandler> Cannot find step %d on job %d in nodeRun %d/%d for workflow %s in project %s",
			stepOrder, runJobID, nodeRunID, number, workflowName, projectKey), "")
	}

	logs, errL := workflow.LoadStepLogs(db, runJobID, stepOrder)
	if errL != nil {
		return sdk.WrapError(errL, "getWorkflowNodeRunJobBuildLogsHandler> Cannot load log for runJob %d on step %d", runJobID, stepOrder)
	}

	result := &sdk.BuildState{
		Status:   sdk.StatusFromString(stepStatus),
		StepLogs: *logs,
	}

	return WriteJSON(w, r, result, http.StatusOK)
}
