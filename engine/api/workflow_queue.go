package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/venom"
	"github.com/sguiheux/go-coverage"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postTakeWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "id")
		if err != nil {
			return err
		}

		if ok := isWorker(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		wk, err := worker.LoadByID(ctx, api.mustDB(), getAPIConsumer(ctx).Worker.ID)
		if err != nil {
			return err
		}

		p, err := project.LoadProjectByNodeJobRunID(ctx, api.mustDB(), api.Cache, id, project.LoadOptions.WithVariables, project.LoadOptions.WithClearKeys)
		if err != nil {
			return sdk.WrapError(err, "cannot load project by nodeJobRunID: %d", id)
		}

		// Load worker model
		var workerModelName string
		if wk.ModelID != nil {
			wm, err := workermodel.LoadByID(api.mustDB(), *wk.ModelID)
			if err != nil {
				return sdk.WithStack(sdk.ErrNoWorkerModel)
			}
			workerModelName = wm.Name
		}

		// Load job run
		pbj, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "cannot load job nodeJobRunID: %d", id)
		}

		observability.Current(ctx,
			observability.Tag(observability.TagWorkflowNodeJobRun, id),
			observability.Tag(observability.TagWorkflowNodeRun, pbj.WorkflowNodeRunID),
			observability.Tag(observability.TagJob, pbj.Job.Action.Name))

		// Checks that the token used by the worker cas access to one of the execgroups
		grantedGroupIDs := append(getAPIConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID)
		if !pbj.ExecGroups.HasOneOf(grantedGroupIDs...) {
			return sdk.WrapError(sdk.ErrForbidden, "worker %s (%s) is not authorized to take this job:%d execGroups:%+v", wk.Name, workerModelName, id, pbj.ExecGroups)
		}

		pbji := &sdk.WorkflowNodeJobRunData{}
		report, err := takeJob(ctx, api.mustDB, api.Cache, p, id, workerModelName, pbji, wk)
		if err != nil {
			return sdk.WrapError(err, "cannot takeJob nodeJobRunID:%d", id)
		}

		workflow.ResyncNodeRunsWithCommits(ctx, api.mustDB(), api.Cache, *p, report)
		go WorkflowSendEvent(context.Background(), api.mustDB(), api.Cache, p.Key, report)

		return service.WriteJSON(w, pbji, http.StatusOK)
	}
}

func takeJob(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, p *sdk.Project, id int64, workerModel string, wnjri *sdk.WorkflowNodeJobRunData, wk *sdk.Worker) (*workflow.ProcessorReport, error) {
	// Start a tx
	tx, errBegin := dbFunc().Begin()
	if errBegin != nil {
		return nil, sdk.WrapError(errBegin, "Cannot start transaction")
	}
	defer tx.Rollback() // nolint

	//Prepare spawn infos
	infos := []sdk.SpawnInfo{
		{
			RemoteTime: getRemoteTime(ctx),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID, Args: []interface{}{fmt.Sprintf("%d", id), wk.Name}},
		},
		{
			RemoteTime: getRemoteTime(ctx),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTakenWorkerVersion.ID, Args: []interface{}{wk.Name, wk.Version, wk.OS, wk.Arch}},
		},
	}

	// Take node job run
	job, report, err := workflow.TakeNodeJobRun(ctx, tx, store, *p, id, workerModel, wk.Name, wk.ID, infos)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot take job %d", id)
	}

	// Change worker status
	if err := worker.SetToBuilding(tx, wk.ID, job.ID); err != nil {
		return nil, sdk.WrapError(err, "cannot update worker %s status", wk.Name)
	}

	// Load the node run
	noderun, err := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID, workflow.LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get node run")
	}

	if noderun.Status == sdk.StatusWaiting {
		noderun.Status = sdk.StatusBuilding
		if err := workflow.UpdateNodeRun(tx, noderun); err != nil {
			return nil, sdk.WrapError(err, "Cannot update node run")
		}
		report.Add(ctx, *noderun)
	}

	// Load workflow run
	workflowRun, err := workflow.LoadRunByID(tx, noderun.WorkflowRunID, workflow.LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to load workflow run")
	}

	// Load the secrets
	pv, err := project.LoadAllVariablesWithDecrytion(tx, p.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot load project variable")
	}

	secrets, errSecret := workflow.LoadSecrets(tx, store, noderun, workflowRun, pv)
	if errSecret != nil {
		return nil, sdk.WrapError(errSecret, "Cannot load secrets")
	}

	// Feed the worker
	wnjri.NodeJobRun = *job
	wnjri.Number = noderun.Number
	wnjri.SubNumber = noderun.SubNumber
	wnjri.Secrets = secrets

	params, secretsKeys, errK := workflow.LoadNodeJobRunKeys(ctx, tx, p, workflowRun, noderun)
	if errK != nil {
		return nil, sdk.WrapError(errK, "Cannot load keys")
	}
	wnjri.Secrets = append(wnjri.Secrets, secretsKeys...)
	wnjri.NodeJobRun.Parameters = append(wnjri.NodeJobRun.Parameters, params...)

	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "Cannot commit transaction")
	}

	return report, nil
}

func (api *API) postBookWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if ok := isHatchery(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		s, err := services.LoadByID(ctx, api.mustDB(), getAPIConsumer(ctx).Service.ID)
		if err != nil {
			return err
		}

		if _, err := workflow.BookNodeJobRun(ctx, api.Cache, id, s); err != nil {
			return sdk.WrapError(err, "job already booked")
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) deleteBookWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if ok := isHatchery(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		if err := workflow.FreeNodeJobRun(ctx, api.Cache, id); err != nil {
			return sdk.WrapError(err, "job not booked")
		}
		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return sdk.WrapError(err, "invalid id")
		}

		j, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "job not found")
		}

		return service.WriteJSON(w, j, http.StatusOK)
	}
}

func (api *API) postVulnerabilityReportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return sdk.WrapError(err, "invalid id")
		}

		nr, err := workflow.LoadNodeRunByNodeJobID(api.mustDB(), id, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to save vulnerability report")
		}
		if nr.ApplicationID == 0 {
			return sdk.WrapError(sdk.ErrApplicationNotFound, "there is no application linked")
		}

		var report sdk.VulnerabilityWorkerReport
		if err := service.UnmarshalBody(r, &report); err != nil {
			return sdk.WrapError(err, "unable to read body")
		}

		p, err := project.LoadProjectByNodeJobRunID(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "cannot load project by nodeJobRunID: %d", id)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() // nolint

		if err := workflow.HandleVulnerabilityReport(ctx, tx, api.Cache, *p, nr, report); err != nil {
			return sdk.WrapError(err, "unable to handle report")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) postSpawnInfosWorkflowJobHandler() service.AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return sdk.WrapError(err, "invalid id")
		}

		if ok := isHatchery(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		observability.Current(ctx, observability.Tag(observability.TagWorkflowNodeJobRun, id))

		var s []sdk.SpawnInfo
		if err := service.UnmarshalBody(r, &s); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal request")
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if _, err := workflow.LoadNodeJobRun(ctx, tx, api.Cache, id); err != nil {
			if !sdk.ErrorIs(err, sdk.ErrWorkflowNodeRunJobNotFound) {
				return err
			}
			return nil
		}
		if err := workflow.AddSpawnInfosNodeJobRun(tx, id, s); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) postWorkflowJobResultHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if ok := isWorker(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		wk, err := worker.LoadByID(ctx, api.mustDB(), getAPIConsumer(ctx).Worker.ID)
		if err != nil {
			return err
		}

		// Unmarshal into results
		var res sdk.Result
		if err := service.UnmarshalBody(r, &res); err != nil {
			return sdk.WrapError(err, "cannot unmarshal request")
		}
		customCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
		defer cancel()
		dbWithCtx := api.mustDBWithCtx(customCtx)

		_, next := observability.Span(ctx, "project.LoadProjectByNodeJobRunID")
		proj, err := project.LoadProjectByNodeJobRunID(ctx, dbWithCtx, api.Cache, id, project.LoadOptions.WithVariables)
		next()
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNoProject) {
				_, err := workflow.LoadNodeJobRun(ctx, dbWithCtx, api.Cache, id)
				if sdk.ErrorIs(err, sdk.ErrWorkflowNodeRunJobNotFound) {
					// job result already send as job is no more in database
					// this log is here to stats it and we returns nil for unlock the worker
					// and avoid a "worker timeout"
					log.Warning(ctx, "NodeJobRun not found: %d err:%v", id, err)
					return nil
				}
				return sdk.WrapError(err, "cannot load NodeJobRun %d", id)
			}
			return sdk.WrapError(err, "cannot load project from job %d", id)
		}

		observability.Current(ctx,
			observability.Tag(observability.TagProjectKey, proj.Key),
		)

		report, err := postJobResult(customCtx, api.mustDBWithCtx, api.Cache, proj, wk, &res)
		if err != nil {
			return sdk.WrapError(err, "unable to post job result")
		}

		workflowRuns := report.WorkflowRuns()
		if len(workflowRuns) > 0 {
			observability.Current(ctx,
				observability.Tag(observability.TagWorkflow, workflowRuns[0].Workflow.Name))

			if workflowRuns[0].Status == sdk.StatusFail {
				observability.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
			}
		}

		_, next = observability.Span(ctx, "workflow.ResyncNodeRunsWithCommits")
		workflow.ResyncNodeRunsWithCommits(ctx, api.mustDB(), api.Cache, *proj, report)
		next()

		go WorkflowSendEvent(context.Background(), api.mustDB(), api.Cache, proj.Key, report)

		return nil
	}
}

func postJobResult(ctx context.Context, dbFunc func(context.Context) *gorp.DbMap, store cache.Store, proj *sdk.Project, wr *sdk.Worker, res *sdk.Result) (*workflow.ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "postJobResult")
	defer end()

	//Start the transaction
	tx, errb := dbFunc(ctx).Begin()
	if errb != nil {
		return nil, sdk.WrapError(errb, "postJobResult> Cannot begin tx")
	}
	defer tx.Rollback() // nolint

	//Load workflow node job run
	job, errj := workflow.LoadAndLockNodeJobRunSkipLocked(ctx, tx, store, res.BuildID)
	if errj != nil {
		return nil, sdk.WrapError(errj, "cannot load node run job %d", res.BuildID)
	}

	observability.Current(ctx,
		observability.Tag(observability.TagWorkflowNodeJobRun, res.BuildID),
		observability.Tag(observability.TagWorkflowNodeRun, job.WorkflowNodeRunID),
		observability.Tag(observability.TagJob, job.Job.Action.Name))

	infos := []sdk.SpawnInfo{{
		RemoteTime: res.RemoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{wr.Name, res.Duration}},
	}}

	if err := workflow.AddSpawnInfosNodeJobRun(tx, job.ID, workflow.PrepareSpawnInfos(infos)); err != nil {
		return nil, sdk.WrapError(err, "Cannot save spawn info job %d", job.ID)
	}

	// Manage build variables, we have to push them on the job and to propagate on the node above
	for _, v := range res.NewVariables {
		log.Debug("postJobResult> managing new variable %s on job %d", v.Name, job.ID)
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
			log.Debug("postJobResult> adding new variable %s on job %d", v.Name, job.ID)
			sdk.AddParameter(&job.Parameters, v.Name, sdk.StringParameter, v.Value)
		}
	}

	if err := workflow.UpdateNodeJobRun(ctx, tx, job); err != nil {
		return nil, sdk.WrapError(err, "Unable to update node job run %d", res.BuildID)
	}

	node, errn := workflow.LoadNodeRunByID(tx, job.WorkflowNodeRunID, workflow.LoadRunOptions{})
	if errn != nil {
		return nil, sdk.WrapError(errn, "postJobResult> Unable to load node %d", job.WorkflowNodeRunID)
	}

	for _, v := range res.NewVariables {
		log.Debug("postJobResult> managing new variable %s on node %d", v.Name, node.ID)
		found := false
		for i := range node.BuildParameters {
			currentV := &node.BuildParameters[i]
			if currentV.Name == v.Name {
				currentV.Value = v.Value
				found = true
				break
			}
		}
		if !found {
			sdk.AddParameter(&node.BuildParameters, v.Name, sdk.StringParameter, v.Value)
		}
	}

	if err := workflow.UpdateNodeRunBuildParameters(tx, node.ID, node.BuildParameters); err != nil {
		return nil, sdk.WrapError(err, "unable to update node run %d", node.ID)
	}
	// ^ build variables are now updated on job run and on node

	//Update worker status
	if err := worker.SetStatus(tx, wr.ID, sdk.StatusWaiting); err != nil {
		return nil, sdk.WrapError(err, "cannot update worker %s status", wr.ID)
	}

	// Update action status
	log.Debug("postJobResult> Updating %d to %s in queue", job.ID, res.Status)
	newDBFunc := func() *gorp.DbMap {
		return dbFunc(context.Background())
	}
	report, err := workflow.UpdateNodeJobRunStatus(ctx, tx, store, *proj, job, res.Status)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot update NodeJobRun %d status", job.ID)
	}

	//Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "cannot commit tx")
	}

	for i := range report.WorkflowRuns() {
		run := &report.WorkflowRuns()[i]
		report, err := updateParentWorkflowRun(ctx, newDBFunc, store, run)
		if err != nil {
			return nil, sdk.WithStack(err)
		}

		go WorkflowSendEvent(context.Background(), tx, store, proj.Key, report)

		if sdk.StatusIsTerminated(run.Status) {
			//Start a goroutine to update commit statuses in repositories manager
			go func(wRun *sdk.WorkflowRun) {
				//The function could be called with nil project so we need to test if project is not nil
				if sdk.StatusIsTerminated(wRun.Status) && proj != nil {
					wRun.LastExecution = time.Now()
					if err := workflow.ResyncCommitStatus(context.Background(), dbFunc(context.Background()), store, *proj, wRun); err != nil {
						log.Error(ctx, "workflow.UpdateNodeJobRunStatus> %v", err)
					}
				}
			}(run)
		}
	}

	return report, nil
}

func (api *API) postWorkflowJobLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return sdk.WrapError(err, "invalid id")
		}

		pbJob, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "cannot get job run %d", id)
		}

		if ok := isWorker(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Checks that the token used by the worker cas access to one of the execgroups
		grantedGroupIDs := append(getAPIConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID)
		if !pbJob.ExecGroups.HasOneOf(grantedGroupIDs...) {
			return sdk.WrapError(sdk.ErrForbidden, "this worker is not authorized to send logs for this job: %d execGroups: %+v", id, pbJob.ExecGroups)
		}

		var logs sdk.Log
		if err := service.UnmarshalBody(r, &logs); err != nil {
			return err
		}

		log.Debug("postWorkflowJobLogsHandler> Logs: %+v", logs)

		if err := workflow.AddLog(api.mustDB(), pbJob, &logs, api.Config.Log.StepMaxSize); err != nil {
			return err
		}

		return nil
	}
}

func (api *API) postWorkflowJobServiceLogsHandler() service.AsynchronousHandler {
	return func(ctx context.Context, r *http.Request) error {
		if ok := isHatchery(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		var logs []sdk.ServiceLog
		if err := service.UnmarshalBody(r, &logs); err != nil {
			return err
		}
		db := api.mustDB()

		globalErr := &sdk.MultiError{}
		errorOccured := false
		for _, log := range logs {
			nodeRunJob, errJob := workflow.LoadNodeJobRun(ctx, db, api.Cache, log.WorkflowNodeJobRunID)
			if errJob != nil {
				errorOccured = true
				globalErr.Append(fmt.Errorf("postWorkflowJobServiceLogsHandler> Cannot get job run %d : %v", log.WorkflowNodeJobRunID, errJob))
				continue
			}
			log.WorkflowNodeRunID = nodeRunJob.WorkflowNodeRunID

			// Checks that the token used by the worker cas access to one of the execgroups
			grantedGroupIDs := append(getAPIConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID)
			if !nodeRunJob.ExecGroups.HasOneOf(grantedGroupIDs...) {
				errorOccured = true
				globalErr.Append(fmt.Errorf("postWorkflowJobServiceLogsHandler> Forbidden, you have no execution rights on workflow node"))
				continue
			}

			if err := workflow.AddServiceLog(db, nodeRunJob, &log, api.Config.Log.ServiceMaxSize); err != nil {
				errorOccured = true
				globalErr.Append(fmt.Errorf("postWorkflowJobServiceLogsHandler> %v", err))
			}
		}

		if errorOccured {
			log.Error(ctx, globalErr.Error())
			return globalErr
		}

		return nil
	}
}

func (api *API) postWorkflowJobStepStatusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}
		dbWithCtx := api.mustDBWithCtx(ctx)

		nodeJobRun, err := workflow.LoadNodeJobRun(ctx, dbWithCtx, api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "cannot get job run %d", id)
		}

		var step sdk.StepStatus
		if err := service.UnmarshalBody(r, &step); err != nil {
			return err
		}

		found := false
		for i := range nodeJobRun.Job.StepStatus {
			jobStep := &nodeJobRun.Job.StepStatus[i]
			if step.StepOrder == jobStep.StepOrder {
				if nodeJobRun.Status == sdk.StatusStopped {
					jobStep.Status = sdk.StatusStopped
				} else {
					jobStep.Status = step.Status
				}
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

		tx, err := dbWithCtx.Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := workflow.UpdateNodeJobRun(ctx, tx, nodeJobRun); err != nil {
			return sdk.WrapError(err, "error while update job run. JobID on handler: %d", id)
		}

		var nodeRun sdk.WorkflowNodeRun
		if !found {
			nodeRun, err := workflow.LoadAndLockNodeRunByID(ctx, tx, nodeJobRun.WorkflowNodeRunID)
			if err != nil {
				return sdk.WrapError(err, "cannot load node run: %d", nodeJobRun.WorkflowNodeRunID)
			}
			sync, err := workflow.SyncNodeRunRunJob(ctx, tx, nodeRun, *nodeJobRun)
			if err != nil {
				return sdk.WrapError(err, "unable to sync nodeJobRun. JobID on handler: %d", id)
			}
			if !sync {
				log.Warning(ctx, "postWorkflowJobStepStatusHandler> sync doesn't find a nodeJobRun. JobID on handler: %d", id)
			}
			if err := workflow.UpdateNodeRun(tx, nodeRun); err != nil {
				return sdk.WrapError(err, "cannot update node run. JobID on handler: %d", id)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		if nodeRun.ID == 0 {
			nodeRunP, err := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{
				DisableDetailledNodeRun: true,
			})
			if err != nil {
				log.Warning(ctx, "postWorkflowJobStepStatusHandler> Unable to load node run for event: %v", err)
				return nil
			}
			nodeRun = *nodeRunP
		}

		work, err := workflow.LoadWorkflowFromWorkflowRunID(api.mustDB(), nodeRun.WorkflowRunID)
		if err != nil {
			log.Warning(ctx, "postWorkflowJobStepStatusHandler> Unable to load workflow for event: %v", err)
			return nil
		}
		nodeRun.Translate(r.Header.Get("Accept-Language"))
		event.PublishWorkflowNodeRun(context.Background(), api.mustDB(), api.Cache, nodeRun, work, nil)
		return nil
	}
}

func (api *API) countWorkflowJobQueueHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		since, until, _ := getSinceUntilLimitHeader(ctx, w, r)
		modelType, ratioService, err := getModelTypeRatioService(ctx, r)
		if err != nil {
			return err
		}

		filter := workflow.NewQueueFilter()
		filter.ModelType = []string{modelType}
		filter.RatioService = ratioService
		filter.Since = &since
		filter.Until = &until

		var count sdk.WorkflowNodeJobRunCount
		if !isMaintainer(ctx) && !isAdmin(ctx) {
			count, err = workflow.CountNodeJobRunQueueByGroupIDs(ctx, api.mustDB(), api.Cache, filter, getAPIConsumer(ctx).GetGroupIDs())
		} else {
			count, err = workflow.CountNodeJobRunQueue(ctx, api.mustDB(), api.Cache, filter)
		}
		if err != nil {
			return sdk.WrapError(err, "Unable to count queue")
		}

		return service.WriteJSON(w, count, http.StatusOK)
	}
}

func (api *API) getWorkflowJobQueueHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		since, until, limit := getSinceUntilLimitHeader(ctx, w, r)
		status, err := QueryStrings(r, "status")
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		if !sdk.StatusValidate(status...) {
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Invalid given status"))
		}
		if len(status) == 0 {
			status = []string{sdk.StatusWaiting}
		}

		modelType, ratioService, errM := getModelTypeRatioService(ctx, r)
		if errM != nil {
			return errM
		}

		permissions := sdk.PermissionReadExecute

		isW := isWorker(ctx)
		isS := isService(ctx)
		if !isW && !isS {
			permissions = sdk.PermissionRead
		}

		filter := workflow.NewQueueFilter()
		filter.RatioService = ratioService
		filter.Since = &since
		filter.Until = &until
		filter.Rights = permissions
		filter.Statuses = status
		filter.Limit = &limit
		if modelType != "" {
			filter.ModelType = []string{modelType}
		}
		var jobs []sdk.WorkflowNodeJobRun
		if !isMaintainer(ctx) && !isAdmin(ctx) {
			jobs, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, api.mustDB(), api.Cache, filter, getAPIConsumer(ctx).GetGroupIDs())
		} else {
			jobs, err = workflow.LoadNodeJobRunQueue(ctx, api.mustDB(), api.Cache, filter)
		}
		if err != nil {
			return sdk.WrapError(err, "Unable to load queue")
		}

		return service.WriteJSON(w, jobs, http.StatusOK)
	}
}

func getModelTypeRatioService(ctx context.Context, r *http.Request) (string, *int, error) {
	modelType := FormString(r, "modelType")
	if modelType != "" {
		if !sdk.WorkerModelValidate(modelType) {
			return "", nil, sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Invalid given modelType"))
		}
	}
	ratioService := FormString(r, "ratioService")
	var ratio *int
	if ratioService != "" {
		i, err := strconv.Atoi(ratioService)
		if err != nil {
			return "", nil, sdk.WrapError(sdk.ErrInvalidNumber, "getModelTypeRatioService> %s is not a integer", ratioService)
		}
		ratio = &i
	}
	return modelType, ratio, nil
}

// getSinceUntilLimitHeader returns since, until, limit
func getSinceUntilLimitHeader(ctx context.Context, w http.ResponseWriter, r *http.Request) (time.Time, time.Time, int) {
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

	limitHeader := r.Header.Get("X-CDS-Limit")
	var limit int
	if limitHeader != "" {
		limit, _ = strconv.Atoi(limitHeader)
	}

	return since, until, limit
}

func (api *API) postWorkflowJobCoverageResultsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Load and lock Existing workflow Run Job
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		var report coverage.Report
		if err := service.UnmarshalBody(r, &report); err != nil {
			return err
		}

		wnr, err := workflow.LoadNodeRunByNodeJobID(api.mustDB(), id, workflow.LoadRunOptions{})
		if err != nil {
			return sdk.WrapError(err, "unable to load node run")
		}

		existingReport, errLoad := workflow.LoadCoverageReport(api.mustDB(), wnr.ID)
		if errLoad != nil && !sdk.ErrorIs(errLoad, sdk.ErrNotFound) {
			return sdk.WrapError(errLoad, "unable to load coverage report")
		}

		p, err := project.LoadProjectByNodeJobRunID(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "cannot load project by nodeJobRunID:%d", id)
		}
		if sdk.ErrorIs(errLoad, sdk.ErrNotFound) {
			if err := workflow.ComputeNewReport(ctx, api.mustDB(), api.Cache, report, wnr, *p); err != nil {
				return sdk.WrapError(err, "cannot compute new coverage report")
			}
			return nil
		}

		// update
		existingReport.Report = report
		if err := workflow.ComputeLatestDefaultBranchReport(ctx, api.mustDB(), api.Cache, *p, wnr, &existingReport); err != nil {
			return sdk.WrapError(err, "cannot compute default branch coverage report")
		}

		if err := workflow.UpdateCoverage(api.mustDB(), existingReport); err != nil {
			return sdk.WrapError(err, "unable to update code coverage")
		}

		return nil
	}
}

func (api *API) postWorkflowJobTestsResultsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Unmarshal into results
		var new venom.Tests
		if err := service.UnmarshalBody(r, &new); err != nil {
			return err
		}

		// Load and lock Existing workflow Run Job
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		nodeRunJob, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "cannot load node run job")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		nr, err := workflow.LoadAndLockNodeRunByID(ctx, tx, nodeRunJob.WorkflowNodeRunID)
		if err != nil {
			return sdk.WrapError(err, "node run not found: %d", nodeRunJob.WorkflowNodeRunID)
		}

		if nr.Tests == nil {
			nr.Tests = &venom.Tests{}
		}

		for k := range new.TestSuites {
			for i := range nr.Tests.TestSuites {
				if nr.Tests.TestSuites[i].Name == new.TestSuites[k].Name {
					// testsuite with same name already exists,
					// Create a unique name
					new.TestSuites[k].Name = fmt.Sprintf("%s.%d", new.TestSuites[k].Name, id)
					break
				}
			}
			nr.Tests.TestSuites = append(nr.Tests.TestSuites, new.TestSuites[k])
		}

		// update total values
		nr.Tests.Total = 0
		nr.Tests.TotalOK = 0
		nr.Tests.TotalKO = 0
		nr.Tests.TotalSkipped = 0
		for _, ts := range nr.Tests.TestSuites {
			nr.Tests.Total += ts.Total
			nr.Tests.TotalKO += ts.Failures + ts.Errors
			nr.Tests.TotalOK += ts.Total - ts.Skipped - ts.Failures - ts.Errors
			nr.Tests.TotalSkipped += ts.Skipped
		}

		if err := workflow.UpdateNodeRun(tx, nr); err != nil {
			return sdk.WrapError(err, "cannot update node run")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot update node run")
		}

		// If we are on default branch, push metrics
		if nr.VCSServer != "" && nr.VCSBranch != "" {
			p, err := project.LoadProjectByNodeJobRunID(ctx, api.mustDB(), api.Cache, id)
			if err != nil {
				log.Error(ctx, "postWorkflowJobTestsResultsHandler> Cannot load project by nodeJobRunID %d: %v", id, err)
				return nil
			}

			// Get vcs info to known if we are on the default branch or not
			projectVCSServer := repositoriesmanager.GetProjectVCSServer(*p, nr.VCSServer)
			client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, p.Key, projectVCSServer)
			if err != nil {
				log.Error(ctx, "postWorkflowJobTestsResultsHandler> Cannot get repo client %s : %v", nr.VCSServer, err)
				return nil
			}

			defaultBranch, err := repositoriesmanager.DefaultBranch(ctx, client, nr.VCSRepository)
			if err != nil {
				log.Error(ctx, "postWorkflowJobTestsResultsHandler> Unable to get default branch: %v", err)
				return nil
			}

			if defaultBranch.DisplayID == nr.VCSBranch {
				// Push metrics
				metrics.PushUnitTests(p.Key, nr.ApplicationID, nr.WorkflowID, nr.Number, *nr.Tests)
			}

		}
		return nil
	}
}

func (api *API) postWorkflowJobTagsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		var tags []sdk.WorkflowRunTag
		if err := service.UnmarshalBody(r, &tags); err != nil {
			return err
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "unable to start transaction")
		}
		defer tx.Rollback() // nolint

		workflowRun, err := workflow.LoadAndLockRunByJobID(tx, id, workflow.LoadRunOptions{})
		if err != nil {
			return sdk.WrapError(err, "unable to load node run id %d", id)
		}

		for _, t := range tags {
			workflowRun.Tag(t.Tag, t.Value)
		}

		if err := workflow.UpdateWorkflowRunTags(tx, workflowRun); err != nil {
			return sdk.WrapError(err, "unable to insert tags")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit transaction")
		}

		return nil
	}
}
