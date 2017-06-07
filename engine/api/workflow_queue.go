package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/runabove/venom"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/objectstore"
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
	// Unmarshal into results
	var new venom.Tests
	if err := UnmarshalBody(r, &new); err != nil {
		return sdk.WrapError(err, "postWorkflowJobTestsResultsHandler> cannot unmarshal request")
	}

	// Load and lock Existing workflow Run Job
	id, errI := requestVarInt(r, "permID")
	if errI != nil {
		return sdk.WrapError(errI, "postWorkflowJobTestsResultsHandler> Invalid node job run ID")
	}

	nodeRunJob, errJobRun := workflow.LoadNodeJobRun(db, id)
	if errJobRun != nil {
		return sdk.WrapError(errJobRun, "postWorkflowJobTestsResultsHandler> Cannot load node run job")
	}

	tx, errB := db.Begin()
	if errB != nil {
		return sdk.WrapError(errB, "postWorkflowJobTestsResultsHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	wnjr, err := workflow.LoadAndLockNodeRunByID(tx, nodeRunJob.WorkflowNodeRunID)
	if err != nil {
		return sdk.WrapError(err, "postWorkflowJobTestsResultsHandler> Cannot load node job")
	}

	if wnjr.Tests == nil {
		wnjr.Tests = &venom.Tests{}
	}

	for k := range new.TestSuites {
		for i := range wnjr.Tests.TestSuites {
			if wnjr.Tests.TestSuites[i].Name == new.TestSuites[k].Name {
				// testsuite with same name already exists,
				// Create a unique name
				new.TestSuites[k].Name = fmt.Sprintf("%s.%d", new.TestSuites[k].Name, id)
				break
			}
		}
		wnjr.Tests.TestSuites = append(wnjr.Tests.TestSuites, new.TestSuites[k])
	}

	// update total values
	wnjr.Tests.Total = 0
	wnjr.Tests.TotalOK = 0
	wnjr.Tests.TotalKO = 0
	wnjr.Tests.TotalSkipped = 0
	for _, ts := range wnjr.Tests.TestSuites {
		wnjr.Tests.Total += ts.Total
		wnjr.Tests.TotalKO += ts.Failures + ts.Errors
		wnjr.Tests.TotalOK += ts.Total - ts.Skipped - ts.Failures - ts.Errors
		wnjr.Tests.TotalSkipped += ts.Skipped
	}

	if err := workflow.UpdateWorkflowNodeRun(tx, wnjr); err != nil {
		return sdk.WrapError(err, "postWorkflowJobTestsResultsHandler> Cannot update node run")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowJobTestsResultsHandler> Cannot update node run")
	}
	return nil
}

func postWorkflowJobVariableHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func postWorkflowJobArtifactHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Load and lock Existing workflow Run Job
	id, errI := requestVarInt(r, "permID")
	if errI != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "postWorkflowJobArtifactHandler> Invalid node job run ID")
	}

	vars := mux.Vars(r)
	tag := vars["tag"]
	fileName := r.Header.Get(sdk.ArtifactFileName)

	//parse the multipart form in the request
	if err := r.ParseMultipartForm(100000); err != nil {
		return sdk.WrapError(err, "postWorkflowJobArtifactHandler: Error parsing multipart form")

	}
	//get a ref to the parsed multipart form
	m := r.MultipartForm

	var sizeStr, permStr, md5sum string
	if len(m.Value["size"]) > 0 {
		sizeStr = m.Value["size"][0]
	}
	if len(m.Value["perm"]) > 0 {
		permStr = m.Value["perm"][0]
	}
	if len(m.Value["md5sum"]) > 0 {
		md5sum = m.Value["md5sum"][0]
	}

	if fileName == "" {
		log.Warning("uploadArtifactHandler> %s header is not set", sdk.ArtifactFileName)
		return sdk.WrapError(sdk.ErrWrongRequest, "postWorkflowJobArtifactHandler> %s header is not set", sdk.ArtifactFileName)
	}

	nodeJobRun, errJ := workflow.LoadNodeJobRun(db, id)
	if errJ != nil {
		return sdk.WrapError(errJ, "Cannot load node job run")
	}

	nodeRun, errR := workflow.LoadNodeRunByID(db, nodeJobRun.WorkflowNodeRunID)
	if errR != nil {
		return sdk.WrapError(errR, "Cannot load node run")
	}

	hash, errG := generateHash()
	if errG != nil {
		return sdk.WrapError(errG, "postWorkflowJobArtifactHandler> Could not generate hash")
	}

	var size int64
	var perm uint64

	if sizeStr != "" {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	}

	if permStr != "" {
		perm, _ = strconv.ParseUint(permStr, 10, 32)
	}

	art := sdk.WorkflowNodeRunArtifact{
		Name:              fileName,
		Tag:               tag,
		DownloadHash:      hash,
		Size:              size,
		Perm:              uint32(perm),
		MD5sum:            md5sum,
		WorkflowNodeRunID: nodeRun.ID,
		WorkflowID:        nodeRun.WorkflowRunID,
	}

	files := m.File[fileName]
	if len(files) == 1 {
		file, err := files[0].Open()
		if err != nil {
			log.Warning("postWorkflowJobArtifactHandler> cannot open file: %s\n", err)
			return err

		}

		if err := artifact.SaveWorkflowFile(&art, file); err != nil {
			log.Warning("postWorkflowJobArtifactHandler> cannot save file: %s\n", err)
			file.Close()
			return err
		}
		file.Close()
	}

	nodeRun.Artifacts = append(nodeRun.Artifacts, art)
	if err := workflow.InsertArtifact(db, &art); err != nil {
		_ = objectstore.DeleteArtifact(&art)
		return sdk.WrapError(err, "postWorkflowJobArtifactHandler> Cannot update workflow node run")
	}
	return nil
}

func getDownloadArtifactHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}
