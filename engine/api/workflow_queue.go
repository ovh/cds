package api

import (
	"context"
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
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postWorkflowJobRequirementsErrorHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "requirementsErrorHandler> cannot read body")
		}

		if getWorker(ctx).ID != "" {
			// Load calling worker
			caller, err := worker.LoadWorker(api.mustDB(), getWorker(ctx).ID)
			if err != nil {
				return sdk.WrapError(sdk.ErrWrongRequest, "requirementsErrorHandler> cannot load calling worker: %s", err)
			}

			log.Warning("%s (%s) > %s", getWorker(ctx).ID, caller.Name, string(body))
		}
		return nil
	}
}

func (api *API) postTakeWorkflowJobHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "postTakeWorkflowJobHandler> invalid id")
		}

		takeForm := &worker.TakeForm{}
		if err := UnmarshalBody(r, takeForm); err != nil {
			return sdk.WrapError(err, "postTakeWorkflowJobHandler> cannot unmarshal request")
		}

		p, errP := project.LoadProjectByNodeJobRunID(api.mustDB(), api.Cache, id, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "postTakeWorkflowJobHandler> Cannot load project")
		}

		//Load worker model
		workerModel := getWorker(ctx).Name
		if getWorker(ctx).ModelID != 0 {
			wm, errModel := worker.LoadWorkerModelByID(api.mustDB(), getWorker(ctx).ModelID)
			if errModel != nil {
				return sdk.ErrNoWorkerModel
			}
			workerModel = wm.Name
		}

		// Start a tx
		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "postTakeWorkflowJobHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		//Prepare spawn infos
		infos := []sdk.SpawnInfo{{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID, Args: []interface{}{getWorker(ctx).Name}},
		}}
		if takeForm.BookedJobID != 0 && takeForm.BookedJobID == id {
			infos = append(infos, sdk.SpawnInfo{
				RemoteTime: takeForm.Time,
				Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJob.ID, Args: []interface{}{getWorker(ctx).Name}},
			})
		}

		//Take node job run
		job, errTake := workflow.TakeNodeJobRun(tx, api.Cache, p, id, workerModel, getWorker(ctx).Name, getWorker(ctx).ID, infos, nil)
		if errTake != nil {
			return sdk.WrapError(errTake, "postTakeWorkflowJobHandler> Cannot take job %d", id)
		}

		//Change worker status
		if err := worker.SetToBuilding(tx, getWorker(ctx).ID, job.ID, sdk.JobTypeWorkflowNode); err != nil {
			return sdk.WrapError(err, "postTakeWorkflowJobHandler> Cannot update worker status")
		}

		//Load the node run
		noderun, errn := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID)
		if errn != nil {
			return sdk.WrapError(errn, "postTakeWorkflowJobHandler> Cannot get node run")
		}

		workflowNodeRunEvent := []sdk.WorkflowNodeRun{}
		if noderun.Status == sdk.StatusWaiting.String() {
			noderun.Status = sdk.StatusBuilding.String()
			if err := workflow.UpdateNodeRun(tx, noderun); err != nil {
				return sdk.WrapError(errn, "postTakeWorkflowJobHandler> Cannot get node run")
			}
			workflowNodeRunEvent = append(workflowNodeRunEvent, *noderun)
		}

		//Load workflow run
		workflowRun, err := workflow.LoadRunByID(api.mustDB(), noderun.WorkflowRunID)
		if err != nil {
			return sdk.WrapError(err, "postTakeWorkflowJobHandler> Unable to load workflow run")
		}

		//Load the secrets
		pv, err := project.GetAllVariableInProject(api.mustDB(), p.ID, project.WithClearPassword())
		if err != nil {
			return sdk.WrapError(err, "postTakeWorkflowJobHandler> Cannot load project variable")
		}

		secrets, errSecret := workflow.LoadNodeJobRunSecrets(tx, api.Cache, job, noderun, workflowRun, pv)
		if errSecret != nil {
			return sdk.WrapError(errSecret, "postTakeWorkflowJobHandler> Cannot load secrets")
		}

		//Feed the worker
		pbji := worker.WorkflowNodeJobRunInfo{}
		pbji.NodeJobRun = *job
		pbji.Number = noderun.Number
		pbji.SubNumber = noderun.SubNumber
		pbji.Secrets = secrets

		params, secretsKeys, errK := workflow.LoadNodeJobRunKeys(tx, api.Cache, job, noderun, workflowRun, p)
		if errK != nil {
			return sdk.WrapError(errK, "postTakeWorkflowJobHandler> Cannot load keys")
		}
		pbji.Secrets = append(pbji.Secrets, secretsKeys...)
		pbji.NodeJobRun.Parameters = append(pbji.NodeJobRun.Parameters, params...)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postTakeWorkflowJobHandler> Cannot commit transaction")
		}

		workflow.SendEvent(api.mustDB(), nil, workflowNodeRunEvent, []sdk.WorkflowNodeJobRun{*job}, p.Key)

		return WriteJSON(w, r, pbji, http.StatusOK)
	}
}

func (api *API) postBookWorkflowJobHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "postBookWorkflowJobHandler> invalid id")
		}

		if _, err := workflow.BookNodeJobRun(api.Cache, id, getHatchery(ctx)); err != nil {
			return sdk.WrapError(err, "postBookWorkflowJobHandler> job already booked")
		}
		return WriteJSON(w, r, nil, http.StatusOK)
	}
}

func (api *API) getWorkflowJobHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "getWorkflowJobHandler> invalid id")
		}
		j, err := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowJobHandler> job not found")
		}
		return WriteJSON(w, r, j, http.StatusOK)
	}
}

func (api *API) postSpawnInfosWorkflowJobHandler() AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "postSpawnInfosWorkflowJobHandler> invalid id")
		}
		var s []sdk.SpawnInfo
		if err := UnmarshalBody(r, &s); err != nil {
			return sdk.WrapError(err, "postSpawnInfosWorkflowJobHandler> cannot unmarshal request")
		}

		p, errP := project.LoadProjectByNodeJobRunID(api.mustDB(), api.Cache, id, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "postSpawnInfosWorkflowJobHandler> Cannot load project")
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "postSpawnInfosWorkflowJobHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if _, err := workflow.AddSpawnInfosNodeJobRun(tx, api.Cache, p, id, s); err != nil {
			return sdk.WrapError(err, "postSpawnInfosWorkflowJobHandler> Cannot save job %d", id)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> Cannot commit tx")
		}
		return nil
	}
}

func (api *API) postWorkflowJobResultHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "permID")
		if errc != nil {
			return sdk.WrapError(errc, "postWorkflowJobResultHandler> invalid id")
		}

		p, errP := project.LoadProjectByNodeJobRunID(api.mustDB(), api.Cache, id, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowJobResultHandler> Cannot load project")
		}

		// Unmarshal into results
		var res sdk.Result
		if err := UnmarshalBody(r, &res); err != nil {
			return sdk.WrapError(err, "postWorkflowJobResultHandler> cannot unmarshal request")
		}

		chanEvent := make(chan interface{}, 1)
		chanError := make(chan error, 1)
		go postJobResult(chanEvent, chanError, api.mustDB(), api.Cache, p, getWorker(ctx).Name, &res)

		workflowRuns, workflowNodeRuns, workflowNodeJobRuns, err := workflow.GetWorkflowRunEventData(chanError, chanEvent)
		if err != nil {
			return err
		}
		go workflow.SendEvent(api.mustDB(), workflowRuns, workflowNodeRuns, workflowNodeJobRuns, p.Key)

		return nil
	}
}

func postJobResult(chEvent chan<- interface{}, chError chan<- error, db *gorp.DbMap, store cache.Store, p *sdk.Project, workerName string, res *sdk.Result) {
	defer close(chEvent)
	defer close(chError)

	//Start the transaction
	tx, errb := db.Begin()
	if errb != nil {
		chError <- sdk.WrapError(errb, "postJobResult> Cannot begin tx")
		return
	}
	defer tx.Rollback()

	//Load workflow node job run
	job, errj := workflow.LoadAndLockNodeJobRunWait(tx, store, res.BuildID)
	if errj != nil {
		chError <- sdk.WrapError(errj, "postJobResult> Unable to load node run job %d", res.BuildID)
		return
	}

	remoteTime, errt := ptypes.Timestamp(res.RemoteTime)
	if errt != nil {
		chError <- sdk.WrapError(errt, "postJobResult> Cannot parse remote time")
		return
	}

	//Update spwan info
	infos := []sdk.SpawnInfo{{
		RemoteTime: remoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{workerName, res.Duration}},
	}}

	//Add spawn infos
	if _, err := workflow.AddSpawnInfosNodeJobRun(tx, store, p, job.ID, infos); err != nil {
		chError <- sdk.WrapError(err, "postJobResult> Cannot save spawn info job %d", job.ID)
		return
	}

	// Update action status
	log.Debug("postJobResult> Updating %d to %s in queue", job.ID, res.Status)
	if err := workflow.UpdateNodeJobRunStatus(tx, store, p, job, sdk.Status(res.Status), chEvent); err != nil {
		chError <- sdk.WrapError(err, "postJobResult> Cannot update %d status", job.ID)
		return
	}

	//Commit the transaction
	if err := tx.Commit(); err != nil {
		chError <- sdk.WrapError(err, "postJobResult> Cannot commit tx")
		return
	}
}

func (api *API) postWorkflowJobLogsHandler() AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
		id, errr := requestVarInt(r, "permID")
		if errr != nil {
			return sdk.WrapError(errr, "postWorkflowJobStepStatusHandler> Invalid id")
		}

		pbJob, errJob := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if errJob != nil {
			return sdk.WrapError(errJob, "postWorkflowJobStepStatusHandler> Cannot get job run %d", id)
		}

		var logs sdk.Log
		if err := UnmarshalBody(r, &logs); err != nil {
			return sdk.WrapError(err, "postWorkflowJobLogsHandler> Unable to parse body")
		}

		if err := workflow.AddLog(api.mustDB(), pbJob, &logs); err != nil {
			return sdk.WrapError(err, "postWorkflowJobLogsHandler")
		}

		return nil
	}
}

func (api *API) postWorkflowJobStepStatusHandler() AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
		id, errr := requestVarInt(r, "permID")
		if errr != nil {
			return sdk.WrapError(errr, "postWorkflowJobStepStatusHandler> Invalid id")
		}

		nodeJobRun, errJob := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if errJob != nil {
			return sdk.WrapError(errJob, "postWorkflowJobStepStatusHandler> Cannot get job run %d", id)
		}

		var step sdk.StepStatus
		if err := UnmarshalBody(r, &step); err != nil {
			return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Error while unmarshal job")
		}

		found := false
		for i := range nodeJobRun.Job.StepStatus {
			jobStep := &nodeJobRun.Job.StepStatus[i]
			if step.StepOrder == jobStep.StepOrder {
				jobStep.Status = step.Status
				found = true
			}
		}
		if !found {
			nodeJobRun.Job.StepStatus = append(nodeJobRun.Job.StepStatus, step)
		}

		p, errP := project.LoadProjectByNodeJobRunID(api.mustDB(), api.Cache, id, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowJobStepStatusHandler> Cannot load project")
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "postWorkflowJobStepStatusHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.UpdateNodeJobRun(tx, api.Cache, p, nodeJobRun); err != nil {
			return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Error while update job run")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) getWorkflowJobQueueHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sinceHeader := r.Header.Get("If-Modified-Since")
		since := time.Unix(0, 0)
		if sinceHeader != "" {
			since, _ = time.Parse(time.RFC1123, sinceHeader)
		}

		groupsID := []int64{}
		for _, g := range getUser(ctx).Groups {
			groupsID = append(groupsID, g.ID)
		}
		jobs, err := workflow.LoadNodeJobRunQueue(api.mustDB(), api.Cache, groupsID, &since)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowJobQueueHandler> Unable to load queue")
		}

		return WriteJSON(w, r, jobs, http.StatusOK)
	}
}

func (api *API) postWorkflowJobTestsResultsHandler() AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
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

		nodeRunJob, errJobRun := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if errJobRun != nil {
			return sdk.WrapError(errJobRun, "postWorkflowJobTestsResultsHandler> Cannot load node run job")
		}

		tx, errB := api.mustDB().Begin()
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
}

func (api *API) postWorkflowJobVariableHandler() AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
		id, errr := requestVarInt(r, "permID")
		if errr != nil {
			return sdk.WrapError(errr, "postWorkflowJobVariableHandler> Invalid id")
		}

		// Unmarshal into variable
		var v sdk.Variable
		if err := UnmarshalBody(r, &v); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler")
		}

		p, errP := project.LoadProjectByNodeJobRunID(api.mustDB(), api.Cache, id, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowJobVariableHandler> Cannot load project")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "postWorkflowJobVariableHandler> Unable to start tx")
		}
		defer tx.Rollback()

		job, errj := workflow.LoadAndLockNodeJobRunWait(tx, api.Cache, id)
		if errj != nil {
			return sdk.WrapError(errj, "postWorkflowJobVariableHandler> Unable to load job")
		}

		found := false
		for i := range job.Parameters {
			currentV := &job.Parameters[i]
			if currentV.Name == v.Name {
				currentV.Value = v.Value
				found = true
				break
			}
		}
		if !found {
			sdk.AddParameter(&job.Parameters, v.Name, sdk.StringParameter, v.Value)
		}

		if err := workflow.UpdateNodeJobRun(tx, api.Cache, p, job); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to update node job run")
		}

		node, errn := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID)
		if errn != nil {
			return sdk.WrapError(errn, "postWorkflowJobVariableHandler> Unable to load node")
		}

		found = false
		for i := range node.BuildParameters {
			currentP := &node.BuildParameters[i]
			if currentP.Name == v.Name {
				currentP.Value = v.Value
				found = true
				break
			}
		}
		if !found {
			sdk.AddParameter(&node.BuildParameters, v.Name, sdk.StringParameter, v.Value)
		}

		if err := workflow.UpdateNodeRun(tx, node); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to update node run")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to commit tx")
		}

		return nil
	}
}

func (api *API) postWorkflowJobArtifactHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

		nodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if errJ != nil {
			return sdk.WrapError(errJ, "Cannot load node job run")
		}

		nodeRun, errR := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID)
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
		if err := workflow.InsertArtifact(api.mustDB(), &art); err != nil {
			_ = objectstore.DeleteArtifact(&art)
			return sdk.WrapError(err, "postWorkflowJobArtifactHandler> Cannot update workflow node run")
		}
		return nil
	}
}
