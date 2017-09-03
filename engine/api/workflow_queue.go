package main

import (
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/protobuf/ptypes"
	"github.com/gorilla/mux"
	"github.com/ovh/venom"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func postWorkflowJobRequirementsErrorHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func postTakeWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	//Load the node run
	noderun, errn := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID)
	if errn != nil {
		return sdk.WrapError(errn, "postTakeWorkflowJobHandler> Cannot get node run")
	}

	noderun.Status = sdk.StatusBuilding.String()
	if err := workflow.UpdateNodeRun(tx, noderun); err != nil {
		return sdk.WrapError(errn, "postTakeWorkflowJobHandler> Cannot get node run")
	}

	//Load workflow node run
	nodeRun, err := workflow.LoadNodeRunByID(db, job.WorkflowNodeRunID)
	if err != nil {
		return sdk.WrapError(err, "postTakeWorkflowJobHandler> Unable to load node run")
	}

	//Load workflow run
	workflowRun, err := workflow.LoadRunByID(db, nodeRun.WorkflowRunID)
	if err != nil {
		return sdk.WrapError(err, "postTakeWorkflowJobHandler> Unable to load workflow run")
	}

	//Load the secrets
	secrets, errSecret := workflow.LoadNodeJobRunSecrets(tx, job, nodeRun, workflowRun)
	if errSecret != nil {
		return sdk.WrapError(errSecret, "postTakeWorkflowJobHandler> Cannot load secrets")
	}

	//Feed the worker
	pbji := worker.WorkflowNodeJobRunInfo{}
	pbji.NodeJobRun = *job
	pbji.Number = noderun.Number
	pbji.SubNumber = noderun.SubNumber
	pbji.Secrets = secrets

	params, secretsKeys, errK := workflow.LoadNodeJobRunKeys(tx, job, nodeRun, workflowRun)
	if errK != nil {
		return sdk.WrapError(errK, "postTakeWorkflowJobHandler> Cannot load keys")
	}
	pbji.Secrets = append(pbji.Secrets, secretsKeys...)
	pbji.NodeJobRun.Parameters = append(pbji.NodeJobRun.Parameters, params...)

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postTakeWorkflowJobHandler> Cannot commit transaction")
	}

	return WriteJSON(w, r, pbji, http.StatusOK)
}

func postBookWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "postBookWorkflowJobHandler> invalid id")
	}

	if _, err := workflow.BookNodeJobRun(id, c.Hatchery); err != nil {
		return sdk.WrapError(err, "postBookWorkflowJobHandler> job already booked")
	}
	return WriteJSON(w, r, nil, http.StatusOK)
}

func getWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "getWorkflowJobHandler> invalid id")
	}
	j, err := workflow.LoadNodeJobRun(db, id)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowJobHandler> job not found")
	}
	return WriteJSON(w, r, j, http.StatusOK)
}

func postSpawnInfosWorkflowJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func postWorkflowJobResultHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	remoteTime, errt := ptypes.Timestamp(res.RemoteTime)
	if errt != nil {
		return sdk.WrapError(errt, "postWorkflowJobResultHandler> Cannot parse remote time")
	}

	//Update spwan info
	infos := []sdk.SpawnInfo{{
		RemoteTime: remoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{c.Worker.Name, res.Duration}},
	}}

	//Add spawn infos
	if _, err := workflow.AddSpawnInfosNodeJobRun(tx, job.ID, infos); err != nil {
		log.Error("addQueueResultHandler> Cannot save spawn info job %d: %s", job.ID, err)
		return err
	}

	// Update action status
	log.Debug("postWorkflowJobResultHandler> Updating %d to %s in queue", id, res.Status)
	if err := workflow.UpdateNodeJobRunStatus(tx, job, sdk.Status(res.Status)); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot update %d status", id)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowJobResultHandler> Cannot commit tx")
	}

	return nil
}

func postWorkflowJobLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func postWorkflowJobStepStatusHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	tx, errB := db.Begin()
	if errB != nil {
		return sdk.WrapError(errB, "postWorkflowJobStepStatusHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := workflow.UpdateNodeJobRun(tx, pbJob); err != nil {
		return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Error while update job run")
	}

	return tx.Commit()
}

func getWorkflowJobQueueHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func postWorkflowJobTestsResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	if err := workflow.UpdateNodeRun(tx, wnjr); err != nil {
		return sdk.WrapError(err, "postWorkflowJobTestsResultsHandler> Cannot update node run")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowJobTestsResultsHandler> Cannot update node run")
	}
	return nil
}

func postWorkflowJobVariableHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	id, errr := requestVarInt(r, "permID")
	if errr != nil {
		return sdk.WrapError(errr, "postWorkflowJobVariableHandler> Invalid id")
	}

	// Unmarshal into variable
	var v sdk.Variable
	if err := UnmarshalBody(r, &v); err != nil {
		return sdk.WrapError(err, "postWorkflowJobVariableHandler")
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "postWorkflowJobVariableHandler> Unable to start tx")
	}
	defer tx.Rollback()

	job, errj := workflow.LoadAndLockNodeJobRun(tx, id)
	if errj != nil {
		return sdk.WrapError(errj, "postWorkflowJobVariableHandler> Unable to load job")
	}

	sdk.AddParameter(&job.Parameters, "cds.build."+v.Name, sdk.StringParameter, v.Value)

	if err := workflow.UpdateNodeJobRun(tx, job); err != nil {
		return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to update node job run")
	}

	node, errn := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID)
	if errn != nil {
		return sdk.WrapError(errn, "postWorkflowJobVariableHandler> Unable to load node")
	}

	sdk.AddParameter(&node.BuildParameters, "cds.build."+v.Name, sdk.StringParameter, v.Value)

	if err := workflow.UpdateNodeRun(tx, node); err != nil {
		return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to update node run")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to commit tx")
	}

	return nil
}

func postWorkflowJobArtifactHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Load and lock Existing workflow Run Job
	id, errI := requestVarInt(r, "permID")
	if errI != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "postWorkflowJobArtifactHandler> Invalid node job run ID")
	}

	vars := mux.Vars(r)
	tag := vars["tag"]

	_, params, errM := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
	if errM != nil {
		return sdk.WrapError(errM, "postWorkflowJobArtifactHandler> Cannot read Content Disposition header")
	}

	fileName := params["filename"]

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
		log.Warning("uploadArtifactHandler> %s header is not set", "Content-Disposition")
		return sdk.WrapError(sdk.ErrWrongRequest, "postWorkflowJobArtifactHandler> %s header is not set", "Content-Disposition")
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
		Created:           time.Now(),
	}

	files := m.File[fileName]
	if len(files) == 1 {
		file, err := files[0].Open()
		if err != nil {
			return sdk.WrapError(err, "postWorkflowJobArtifactHandler> cannot open file")

		}

		if err := artifact.SaveWorkflowFile(&art, file); err != nil {
			return sdk.WrapError(err, "postWorkflowJobArtifactHandler> Cannot save artifact in store")
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
