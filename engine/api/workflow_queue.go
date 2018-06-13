package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/protobuf/ptypes"
	"github.com/ovh/venom"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/tracing"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

//DEPRECATED
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

		takeForm := &sdk.WorkerTakeForm{}
		if err := UnmarshalBody(r, takeForm); err != nil {
			return sdk.WrapError(err, "postTakeWorkflowJobHandler> cannot unmarshal request")
		}

		p, errP := project.LoadProjectByNodeJobRunID(ctx, api.mustDB(), api.Cache, id, getUser(ctx), project.LoadOptions.WithVariables, project.LoadOptions.WithClearKeys)
		if errP != nil {
			return sdk.WrapError(errP, "postTakeWorkflowJobHandler> Cannot load project nodeJobRunID:%d", id)
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

		pbj, errl := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if errl != nil {
			return sdk.WrapError(errl, "postTakeWorkflowJobHandler> Cannot load job nodeJobRunID:%d", id)
		}

		tracing.Current(ctx,
			tracing.Tag("node_job_run", id),
			tracing.Tag("workflow_node_run", pbj.WorkflowNodeRunID),
			tracing.Tag("job", pbj.Job.Action.Name))

		// a worker can have only one group
		groups := getUser(ctx).Groups
		if len(groups) != 1 {
			return sdk.WrapError(errl, "postTakeWorkflowJobHandler> too many groups detected on worker:%d", len(groups))
		}

		var isGroupOK bool
		if len(pbj.ExecGroups) == 0 {
			isGroupOK = true
		} else {
			for _, g := range pbj.ExecGroups {
				if g.ID == groups[0].ID {
					isGroupOK = true
					break
				}
			}
		}

		if !isGroupOK {
			return sdk.WrapError(sdk.ErrForbidden, "postTakeWorkflowJobHandler> this worker is not authorized to take this job:%d execGroups:%+v", id, pbj.ExecGroups)
		}

		pbji := &worker.WorkflowNodeJobRunInfo{}

		report, errT := takeJob(ctx, api.mustDB(), api.Cache, p, getWorker(ctx), id, takeForm, workerModel, pbji)
		if errT != nil {
			return sdk.WrapError(errT, "postTakeWorkflowJobHandler> Cannot takeJob nodeJobRunID:%d", id)
		}

		workflowRuns, workflowNodeRuns, workflowNodeJobRuns := workflow.GetWorkflowRunEventData(report, p.Key)
		workflow.ResyncNodeRunsWithCommits(api.mustDB(), api.Cache, p, workflowNodeRuns)

		go workflow.SendEvent(api.mustDB(), workflowRuns, workflowNodeRuns, workflowNodeJobRuns, p.Key)

		return WriteJSON(w, pbji, http.StatusOK)
	}
}

func takeJob(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, wr *sdk.Worker, id int64, takeForm *sdk.WorkerTakeForm, workerModel string, wnjri *worker.WorkflowNodeJobRunInfo) (*workflow.ProcessorReport, error) {
	// Start a tx
	tx, errBegin := db.Begin()
	if errBegin != nil {
		return nil, sdk.WrapError(errBegin, "takeJob> Cannot start transaction")
	}
	defer tx.Rollback()

	//Prepare spawn infos
	infos := []sdk.SpawnInfo{
		{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID, Args: []interface{}{fmt.Sprintf("%d", id), getWorker(ctx).Name}},
		},
		{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTakenWorkerVersion.ID, Args: []interface{}{getWorker(ctx).Name, takeForm.Version, takeForm.OS, takeForm.Arch}},
		},
	}
	if takeForm.BookedJobID != 0 && takeForm.BookedJobID == id {
		infos = append(infos, sdk.SpawnInfo{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJob.ID, Args: []interface{}{getWorker(ctx).Name}},
		})
	}

	//Take node job run
	job, report, errTake := workflow.TakeNodeJobRun(ctx, db, tx, store, p, id, workerModel, getWorker(ctx).Name, getWorker(ctx).ID, infos)
	if errTake != nil {
		return nil, sdk.WrapError(errTake, "takeJob> Cannot take job %d", id)
	}

	//Change worker status
	if err := worker.SetToBuilding(tx, getWorker(ctx).ID, job.ID, sdk.JobTypeWorkflowNode); err != nil {
		return nil, sdk.WrapError(err, "takeJob> Cannot update worker status")
	}

	//Load the node run
	noderun, errn := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID, workflow.LoadRunOptions{})
	if errn != nil {
		return nil, sdk.WrapError(errn, "takeJob> Cannot get node run")
	}

	if noderun.Status == sdk.StatusWaiting.String() {
		noderun.Status = sdk.StatusBuilding.String()
		if err := workflow.UpdateNodeRun(tx, noderun); err != nil {
			return nil, sdk.WrapError(err, "takeJob> Cannot get node run")
		}
		report.Add(*noderun)
	}

	//Load workflow run
	workflowRun, err := workflow.LoadRunByID(tx, noderun.WorkflowRunID, workflow.LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "takeJob> Unable to load workflow run")
	}

	//Load the secrets
	pv, err := project.GetAllVariableInProject(tx, p.ID, project.WithClearPassword())
	if err != nil {
		return nil, sdk.WrapError(err, "takeJob> Cannot load project variable")
	}

	secrets, errSecret := workflow.LoadNodeJobRunSecrets(tx, store, job, noderun, workflowRun, pv)
	if errSecret != nil {
		return nil, sdk.WrapError(errSecret, "takeJob> Cannot load secrets")
	}

	//Feed the worker
	wnjri.NodeJobRun = *job
	wnjri.Number = noderun.Number
	wnjri.SubNumber = noderun.SubNumber
	wnjri.Secrets = secrets

	params, secretsKeys, errK := workflow.LoadNodeJobRunKeys(tx, store, job, noderun, workflowRun, p)
	if errK != nil {
		return nil, sdk.WrapError(errK, "takeJob> Cannot load keys")
	}
	wnjri.Secrets = append(wnjri.Secrets, secretsKeys...)
	wnjri.NodeJobRun.Parameters = append(wnjri.NodeJobRun.Parameters, params...)

	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "takeJob> Cannot commit transaction")
	}

	return report, nil
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
		return WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postIncWorkflowJobAttemptHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errc := requestVarInt(r, "id")
		if errc != nil {
			return sdk.WrapError(errc, "postIncWorkflowJobAttemptHandler> invalid id")
		}
		h := getHatchery(ctx)
		if h == nil {
			return WriteJSON(w, nil, http.StatusUnauthorized)
		}
		spawnAttempts, err := workflow.AddNodeJobAttempt(api.mustDB(), id, h.ID)
		if err != nil {
			return sdk.WrapError(err, "postIncWorkflowJobAttemptHandler> job already booked")
		}

		hCount, err := hatchery.LoadHatcheriesCountByNodeJobRunID(api.mustDB(), id)
		if err != nil {
			return sdk.WrapError(err, "postIncWorkflowJobAttemptHandler> cannot get hatcheries count")
		}

		if int64(len(spawnAttempts)) >= hCount {
			infos := []sdk.SpawnInfo{
				{
					RemoteTime: time.Now(),
					Message: sdk.SpawnMsg{
						ID:   sdk.MsgSpawnInfoHatcheryCannotStartJob.ID,
						Args: []interface{}{},
					},
				},
			}

			tx, errBegin := api.mustDB().Begin()
			if errBegin != nil {
				return sdk.WrapError(errBegin, "postIncWorkflowJobAttemptHandler> Cannot start transaction")
			}
			defer tx.Rollback()

			if err := workflow.AddSpawnInfosNodeJobRun(tx, id, infos); err != nil {
				return sdk.WrapError(err, "postIncWorkflowJobAttemptHandler> Cannot save spawn info on node job run %d", id)
			}

			wfNodeJobRun, errLj := workflow.LoadNodeJobRun(tx, api.Cache, id)
			if errLj != nil {
				return sdk.WrapError(errLj, "postIncWorkflowJobAttemptHandler> Cannot load node job run")
			}

			wfNodeRun, errLr := workflow.LoadAndLockNodeRunByID(ctx, tx, wfNodeJobRun.WorkflowNodeRunID, true)
			if errLr != nil {
				return sdk.WrapError(errLr, "postIncWorkflowJobAttemptHandler> Cannot load node run")
			}

			if found, err := workflow.SyncNodeRunRunJob(ctx, tx, wfNodeRun, *wfNodeJobRun); err != nil || !found {
				return sdk.WrapError(err, "postIncWorkflowJobAttemptHandler> Cannot sync run job (found=%v)", found)
			}

			if err := workflow.UpdateNodeRun(tx, wfNodeRun); err != nil {
				return sdk.WrapError(err, "postIncWorkflowJobAttemptHandler> Cannot update node job run")
			}

			if err := tx.Commit(); err != nil {
				return sdk.WrapError(err, "postIncWorkflowJobAttemptHandler> Cannot commit tx")
			}
		}

		return WriteJSON(w, spawnAttempts, http.StatusOK)
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
		return WriteJSON(w, j, http.StatusOK)
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

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "postSpawnInfosWorkflowJobHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.AddSpawnInfosNodeJobRun(tx, id, s); err != nil {
			return sdk.WrapError(err, "postSpawnInfosWorkflowJobHandler> Cannot save spawn info on node job run %d for %s name %s", id, getAgent(r), r.Header.Get(cdsclient.RequestedNameHeader))
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postSpawnInfosWorkflowJobHandler> Cannot commit tx")
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

		// Unmarshal into results
		var res sdk.Result
		if err := UnmarshalBody(r, &res); err != nil {
			return sdk.WrapError(err, "postWorkflowJobResultHandler> cannot unmarshal request")
		}
		customCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		dbWithCtx := api.mustDBWithCtx(customCtx)

		proj, errP := project.LoadProjectByNodeJobRunID(ctx, dbWithCtx, api.Cache, id, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			if sdk.ErrorIs(errP, sdk.ErrNoProject) {
				_, errLn := workflow.LoadNodeJobRun(dbWithCtx, api.Cache, id)
				if sdk.ErrorIs(errLn, sdk.ErrWorkflowNodeRunJobNotFound) {
					// job result already send as job is no more in database
					// this log is here to stats it and we returns nil for unlock the worker
					// and avoid a "worker timeout"
					log.Warning("postWorkflowJobResultHandler> NodeJobRun not found: %d err:%v", id, errLn)
					return nil
				}
				return sdk.WrapError(errLn, "postWorkflowJobResultHandler> Cannot load NodeJobRun %d", id)
			}
			return sdk.WrapError(errP, "postWorkflowJobResultHandler> Cannot load project from job %d", id)
		}

		report, err := postJobResult(ctx, dbWithCtx, api.Cache, proj, getWorker(ctx), &res)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowJobResultHandler> unable to post job result")
		}

		workflowRuns, workflowNodeRuns, workflowNodeJobRuns := workflow.GetWorkflowRunEventData(report, proj.Key)
		db := api.mustDB()

		workflow.ResyncNodeRunsWithCommits(db, api.Cache, proj, workflowNodeRuns)

		go workflow.SendEvent(db, workflowRuns, workflowNodeRuns, workflowNodeJobRuns, proj.Key)

		return nil
	}
}

func postJobResult(ctx context.Context, db *gorp.DbMap, store cache.Store, proj *sdk.Project, wr *sdk.Worker, res *sdk.Result) (*workflow.ProcessorReport, error) {
	var end func()
	ctx, end = tracing.Span(ctx, "postJobResult")
	defer end()

	//Start the transaction
	tx, errb := db.Begin()
	if errb != nil {
		return nil, sdk.WrapError(errb, "postJobResult> Cannot begin tx")
	}
	defer tx.Rollback()

	//Load workflow node job run
	job, errj := workflow.LoadAndLockNodeJobRunNoWait(ctx, tx, store, res.BuildID)
	if errj != nil {
		return nil, sdk.WrapError(errj, "postJobResult> Unable to load node run job %d", res.BuildID)
	}

	tracing.Current(ctx,
		tracing.Tag("node_job_run", res.BuildID),
		tracing.Tag("workflow_node_run", job.WorkflowNodeRunID),
		tracing.Tag("job", job.Job.Action.Name))

	remoteTime, errt := ptypes.Timestamp(res.RemoteTime)
	if errt != nil {
		return nil, sdk.WrapError(errt, "postJobResult> Cannot parse remote time")
	}

	infos := []sdk.SpawnInfo{{
		RemoteTime: remoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{wr.Name, res.Duration}},
	}}

	if err := workflow.AddSpawnInfosNodeJobRun(tx, job.ID, workflow.PrepareSpawnInfos(infos)); err != nil {
		return nil, sdk.WrapError(err, "postJobResult> Cannot save spawn info job %d", job.ID)
	}

	// Update action status
	log.Debug("postJobResult> Updating %d to %s in queue", job.ID, res.Status)
	report, err := workflow.UpdateNodeJobRunStatus(ctx, db, tx, store, proj, job, sdk.Status(res.Status))
	if err != nil {
		return nil, sdk.WrapError(err, "postJobResult> Cannot update NodeJobRun %d status", job.ID)
	}

	//Update worker status
	if err := worker.UpdateWorkerStatus(tx, wr.ID, sdk.StatusWaiting); err != nil {
		return nil, sdk.WrapError(err, "postJobResult> Cannot update worker %d status", wr.ID)
	}

	//Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "postJobResult> Cannot commit tx")
	}

	return report, nil
}

func (api *API) postWorkflowJobLogsHandler() AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
		id, errr := requestVarInt(r, "permID")
		if errr != nil {
			return sdk.WrapError(errr, "postWorkflowJobLogsHandler> Invalid id")
		}

		pbJob, errJob := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if errJob != nil {
			return sdk.WrapError(errJob, "postWorkflowJobLogsHandler> Cannot get job run %d", id)
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

func (api *API) postWorkflowJobStepStatusHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "permID")
		if errr != nil {
			return sdk.WrapError(errr, "postWorkflowJobStepStatusHandler> Invalid id")
		}
		dbWithCtx := api.mustDBWithCtx(ctx)

		nodeJobRun, errJob := workflow.LoadNodeJobRun(dbWithCtx, api.Cache, id)
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
				if sdk.StatusIsTerminated(step.Status) {
					jobStep.Done = step.Done
				}
				found = true
				break
			}
		}
		if !found {
			step.Done = time.Time{}
			nodeJobRun.Job.StepStatus = append(nodeJobRun.Job.StepStatus, step)
		}

		tx, errB := dbWithCtx.Begin()
		if errB != nil {
			return sdk.WrapError(errB, "postWorkflowJobStepStatusHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.UpdateNodeJobRun(ctx, tx, api.Cache, nodeJobRun); err != nil {
			return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Error while update job run. JobID on handler: %d", id)
		}

		var nodeRun sdk.WorkflowNodeRun
		if !found {
			nodeRun, errNR := workflow.LoadAndLockNodeRunByID(ctx, tx, nodeJobRun.WorkflowNodeRunID, false)
			if errNR != nil {
				return sdk.WrapError(errNR, "postWorkflowJobStepStatusHandler> Cannot load node run")
			}
			sync, errS := workflow.SyncNodeRunRunJob(ctx, tx, nodeRun, *nodeJobRun)
			if errS != nil {
				return sdk.WrapError(errS, "postWorkflowJobStepStatusHandler> unable to sync nodeJobRun. JobID on handler: %d", id)
			}
			if !sync {
				log.Warning("postWorkflowJobStepStatusHandler> sync doesn't find a nodeJobRun. JobID on handler: %d", id)
			}
			if errU := workflow.UpdateNodeRun(tx, nodeRun); errU != nil {
				return sdk.WrapError(errU, "postWorkflowJobStepStatusHandler> Cannot update node run. JobID on handler: %d", id)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowJobStepStatusHandler> Cannot commit transaction")
		}

		if nodeRun.ID == 0 {
			nodeRunP, errN := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{
				DisableDetailledNodeRun: true,
			})
			if errN != nil {
				log.Warning("postWorkflowJobStepStatusHandler> Unable to load node run for event: %v", errN)
				return nil
			}
			nodeRun = *nodeRunP
		}

		work, errW := workflow.LoadWorkflowFromWorkflowRunID(api.mustDB(), nodeRun.WorkflowRunID)
		if errW != nil {
			log.Warning("postWorkflowJobStepStatusHandler> Unable to load workflow for event: %v", errW)
			return nil
		}
		event.PublishWorkflowNodeJobRun(work.ProjectKey, *nodeJobRun, nodeRun, work.GetNode(nodeRun.WorkflowNodeID), work.Name)
		return nil
	}
}

func (api *API) countWorkflowJobQueueHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		since, until := getSinceUntilHeader(ctx, w, r)
		groupsID := []int64{}
		usr := getUser(ctx)
		for _, g := range usr.Groups {
			groupsID = append(groupsID, g.ID)
		}
		if isHatcheryOrWorker(r) {
			usr = nil
		}

		count, err := workflow.CountNodeJobRunQueue(api.mustDB(), api.Cache, groupsID, usr, &since, &until)
		if err != nil {
			return sdk.WrapError(err, "countWorkflowJobQueueHandler> Unable to count queue")
		}

		return WriteJSON(w, count, http.StatusOK)
	}
}

func (api *API) getWorkflowJobQueueHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		since, until := getSinceUntilHeader(ctx, w, r)
		groupsID := make([]int64, len(getUser(ctx).Groups))
		usr := getUser(ctx)
		for i, g := range usr.Groups {
			groupsID[i] = g.ID
		}

		permissions := permission.PermissionReadExecute
		if !isHatcheryOrWorker(r) {
			permissions = permission.PermissionRead
		} else {
			usr = nil
		}

		jobs, err := workflow.LoadNodeJobRunQueue(api.mustDB(), api.Cache, permissions, groupsID, usr, &since, &until)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowJobQueueHandler> Unable to load queue")
		}

		return WriteJSON(w, jobs, http.StatusOK)
	}
}

func getSinceUntilHeader(ctx context.Context, w http.ResponseWriter, r *http.Request) (time.Time, time.Time) {
	sinceHeader := r.Header.Get("If-Modified-Since")
	since := time.Unix(0, 0)
	if sinceHeader != "" {
		since, _ = time.Parse(time.RFC1123, sinceHeader)
	}

	untilHeader := r.Header.Get("X-CDS-Until")
	until := time.Now()
	if untilHeader != "" {
		until, _ = time.Parse(time.RFC1123, untilHeader)
	}

	return since, until
}

func (api *API) postWorkflowJobTestsResultsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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

		wnjr, err := workflow.LoadAndLockNodeRunByID(ctx, tx, nodeRunJob.WorkflowNodeRunID, false)
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

func (api *API) postWorkflowJobTagsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "permID")
		if errr != nil {
			return sdk.WrapError(errr, "postWorkflowJobTagsHandler> Invalid id")
		}

		var tags = []sdk.WorkflowRunTag{}
		if err := UnmarshalBody(r, &tags); err != nil {
			return sdk.WrapError(err, "postWorkflowJobTagsHandler> Unable to unmarshal body")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "postWorkflowJobTagsHandler> Unable to start transaction")
		}
		defer tx.Rollback()

		workflowRun, errl := workflow.LoadAndLockRunByJobID(tx, id, workflow.LoadRunOptions{})
		if errl != nil {
			return sdk.WrapError(errl, "postWorkflowJobTagsHandler> Unable to load node run id %d", id)
		}

		for _, t := range tags {
			workflowRun.Tag(t.Tag, t.Value)
		}

		if err := workflow.UpdateWorkflowRunTags(tx, workflowRun); err != nil {
			return sdk.WrapError(err, "postWorkflowJobTagsHandler> Unable to insert tags")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowJobTagsHandler> Unable to commit transaction")
		}

		return nil
	}
}

func (api *API) postWorkflowJobVariableHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "permID")
		if errr != nil {
			return sdk.WrapError(errr, "postWorkflowJobVariableHandler> Invalid id")
		}

		// Unmarshal into variable
		var v sdk.Variable
		if err := UnmarshalBody(r, &v); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "postWorkflowJobVariableHandler> Unable to start tx")
		}
		defer tx.Rollback()

		job, errj := workflow.LoadAndLockNodeJobRunNoWait(ctx, tx, api.Cache, id)
		if errj != nil {
			return sdk.WrapError(errj, "postWorkflowJobVariableHandler> Unable to load job %d", id)
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

		if err := workflow.UpdateNodeJobRun(ctx, tx, api.Cache, job); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to update node job run %d", id)
		}

		node, errn := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if errn != nil {
			return sdk.WrapError(errn, "postWorkflowJobVariableHandler> Unable to load node %d", job.WorkflowNodeRunID)
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

		if err := workflow.UpdateNodeRunBuildParameters(tx, node.ID, node.BuildParameters); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to update node run %d", node.ID)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowJobVariableHandler> Unable to commit tx")
		}

		return nil
	}
}
