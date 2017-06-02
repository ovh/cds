package main

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func postWorkflowJobRequirementsErrorHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("requirementsErrorHandler> %s", err)
		return err
	}

	if c.Worker.ID != "" {
		// Load calling worker
		caller, err := worker.LoadWorker(db, c.Worker.ID)
		if err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "requirementsErrorHandler> cannot load calling worker: %s", err)
		}

		log.Warning("%s (%s) > %s", c.Worker.ID, caller.Name, string(body))
	}
	return nil
}

func postTakeWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "postTakeWorkflowJobHandler> invalid id")
	}

	takeForm := &worker.TakeForm{}
	if err := UnmarshalBody(r, takeForm); err != nil {
		return sdk.WrapError(err, "postTakeWorkflowJobHandler> cannot unmarshal request")
	}

	// Start a tx
	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "postTakeWorkflowJobHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	//Load worker model
	workerModel := c.Worker.Name
	if c.Worker.Model != 0 {
		wm, errModel := worker.LoadWorkerModelByID(db, c.Worker.Model)
		if errModel != nil {
			return sdk.ErrNoWorkerModel
		}
		workerModel = wm.Name
	}

	//Prepare spawn infos
	infos := []sdk.SpawnInfo{{
		RemoteTime: takeForm.Time,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID, Args: []interface{}{c.Worker.Name}},
	}}
	if takeForm.BookedJobID != 0 && takeForm.BookedJobID == id {
		infos = append(infos, sdk.SpawnInfo{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJob.ID, Args: []interface{}{c.Worker.Name}},
		})
	}

	//Take node job run
	job, errTake := workflow.TakeNodeJobRun(tx, id, workerModel, c.Worker.Name, c.Worker.ID, infos)
	if errTake != nil {
		return sdk.WrapError(errTake, "postTakeWorkflowJobHandler> Cannot take job %d", id)
	}

	//Change worker status
	if err := worker.SetToBuilding(tx, c.Worker.ID, job.ID); err != nil {
		return sdk.WrapError(err, "postTakeWorkflowJobHandler> Cannot update worker status")
	}

	//Load the secrets
	secrets, errSecret := workflow.LoadNodeJobRunSecrets(db, job)
	if errSecret != nil {
		return sdk.WrapError(errSecret, "postTakeWorkflowJobHandler> Cannot load secrets")
	}

	//Load the node run
	noderun, errn := workflow.LoadNodeRunByID(db, job.WorkflowNodeRunID)
	if errn != nil {
		return sdk.WrapError(errn, "postTakeWorkflowJobHandler> Cannot get node run")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postTakeWorkflowJobHandler> Cannot commit transaction")
	}

	//Feed the worker
	pbji := worker.WorkflowNodeJobRunInfo{}
	pbji.NodeJobRun = *job
	pbji.Secrets = secrets
	pbji.Number = noderun.Number
	pbji.SubNumber = noderun.SubNumber

	return WriteJSON(w, r, pbji, http.StatusOK)
}

func postBookWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "postBookWorkflowJobHandler> invalid id")
	}

	if _, err := workflow.BookNodeJobRun(id, c.Hatchery); err != nil {
		return sdk.WrapError(err, "postBookWorkflowJobHandler> job already booked")
	}
	return WriteJSON(w, r, nil, http.StatusOK)
}

func postSpawnInfosWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "postSpawnInfosWorkflowJobHandler> invalid id")
	}
	var s []sdk.SpawnInfo
	if err := UnmarshalBody(r, &s); err != nil {
		return sdk.WrapError(err, "postSpawnInfosWorkflowJobHandler> cannot unmarshal request")
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "postSpawnInfosWorkflowJobHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if _, err := workflow.AddSpawnInfosNodeJobRun(tx, id, s); err != nil {
		return sdk.WrapError(err, "postSpawnInfosWorkflowJobHandler> Cannot save job %d", id)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> Cannot commit tx")
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

func postWorkflowJobResultHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "permID")
	if errc != nil {
		return sdk.WrapError(errc, "postWorkflowJobResultHandler> invalid id")
	}

	//Load workflow node job run
	job, errj := workflow.LoadNodeJobRun(db, id)
	if errj != nil {
		return sdk.WrapError(errj, "postWorkflowJobResultHandler> Unable to load node run job")
	}

	// Unmarshal into results
	var res sdk.Result
	if err := UnmarshalBody(r, &res); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> cannot unmarshal request")
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "postWorkflowJobResultHandler> Cannot begin tx")
	}
	defer tx.Rollback()

	//Update worker status
	if err := worker.UpdateWorkerStatus(tx, c.Worker.ID, sdk.StatusWaiting); err != nil {
		log.Warning("postWorkflowJobResultHandler> Cannot update worker status (%s): %s", c.Worker.ID, err)
	}

	// Update action status
	log.Debug("postWorkflowJobResultHandler> Updating %d to %s in queue", id, res.Status)
	if err := workflow.UpdateNodeJobRunStatus(tx, job, res.Status); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot update %d status", id)
	}

	//Update spwan info
	infos := []sdk.SpawnInfo{{
		RemoteTime: res.RemoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{c.Worker.Name, res.Duration}},
	}}

	//Add spawn infos
	if _, err := workflow.AddSpawnInfosNodeJobRun(tx, job.ID, infos); err != nil {
		log.Error("addQueueResultHandler> Cannot save spawn info job %d: %s", job.ID, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot commit tx")
	}

	return nil
}

//TODO grpc
func postWorkflowJobLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errr := requestVarInt(r, "permID")
	if errr != nil {
		return sdk.WrapError(errr, "postWorkflowJobStepStatusHandler> Invalid id")
	}

	pbJob, errJob := workflow.LoadNodeJobRun(db, id)
	if errJob != nil {
		return sdk.WrapError(errJob, "postWorkflowJobStepStatusHandler> Cannot get job run %d", id)
	}

	var logs sdk.Log
	if err := UnmarshalBody(r, &logs); err != nil {
		return sdk.WrapError(err, "postWorkflowJobLogsHandler> Unable to parse body")
	}

	if err := workflow.AddLog(db, pbJob, &logs); err != nil {
		return sdk.WrapError(err, "postWorkflowJobLogsHandler")
	}

	return nil
}

func postWorkflowJobStepStatusHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errr := requestVarInt(r, "permID")
	if errr != nil {
		return sdk.WrapError(errr, "postWorkflowJobStepStatusHandler> Invalid id")
	}

	pbJob, errJob := workflow.LoadNodeJobRun(db, id)
	if errJob != nil {
		return sdk.WrapError(errJob, "postWorkflowJobStepStatusHandler> Cannot get job run %d", id)
	}

	var step sdk.StepStatus
	if err := UnmarshalBody(r, &step); err != nil {
		return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Error while unmarshal job")
	}

	found := false
	for i := range pbJob.Job.StepStatus {
		jobStep := &pbJob.Job.StepStatus[i]
		if step.StepOrder == jobStep.StepOrder {
			jobStep.Status = step.Status
			found = true
		}
	}
	if !found {
		pbJob.Job.StepStatus = append(pbJob.Job.StepStatus, step)
	}

	if err := workflow.UpdateNodeJobRun(db, pbJob); err != nil {
		return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Error while update job run")
	}

	return nil
}

func getWorkflowJobQueueHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	sinceHeader := r.Header.Get("If-Modified-Since")
	since := time.Unix(0, 0)
	if sinceHeader != "" {
		since, _ = time.Parse(time.RFC1123, sinceHeader)
	}

	groupsID := []int64{}
	for _, g := range c.User.Groups {
		groupsID = append(groupsID, g.ID)
	}
	jobs, err := workflow.LoadNodeJobRunQueue(db, groupsID, &since)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowJobQueueHandler> Unable to load queue")
	}

	return WriteJSON(w, r, jobs, http.StatusOK)
}

func postWorkflowJobTestsResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postWorkflowJobVariableHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postWorkflowJobArtifactHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func getWorkflowJobArtifactsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func getDownloadArtifactHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}
