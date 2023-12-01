package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/cdn"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func (api *API) postTakeWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if ok := isWorker(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		consumer := getUserConsumer(ctx)
		// Locking for the parent consumer
		var hatcheryName string
		if consumer.ParentID != nil {
			parentConsumer, err := authentication.LoadUserConsumerByID(ctx, api.mustDB(), *consumer.ParentID)
			if err != nil {
				return err
			}
			s, err := services.LoadByConsumerID(ctx, api.mustDB(), parentConsumer.ID)
			if err != nil {
				return err
			}
			hatcheryName = s.Name
		}

		wk, err := worker.LoadByID(ctx, api.mustDB(), getUserConsumer(ctx).AuthConsumerUser.Worker.ID)
		if err != nil {
			return err
		}

		if wk.JobRunID == nil || *wk.JobRunID != id {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "unauthorized to take this job. booked:%d vs asked:%d", wk.JobRunID, id)
		}

		p, err := project.LoadProjectByNodeJobRunID(ctx, api.mustDB(), api.Cache, id, project.LoadOptions.WithVariables, project.LoadOptions.WithClearKeys)
		if err != nil {
			return sdk.WrapError(err, "cannot load project by nodeJobRunID: %d", id)
		}

		// Load worker model
		var workerModelName string
		if wk.ModelID != nil {
			wm, err := workermodel.LoadByID(ctx, api.mustDB(), *wk.ModelID, workermodel.LoadOptions.Default)
			if err != nil {
				return err
			}
			workerModelName = wm.Name
		} else if wk.ModelName != nil {
			workerModelName = *wk.ModelName
		}

		// Load job run
		pbj, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return sdk.WrapError(err, "cannot load job nodeJobRunID: %d", id)
		}

		telemetry.Current(ctx,
			telemetry.Tag(telemetry.TagWorkflowNodeJobRun, id),
			telemetry.Tag(telemetry.TagWorkflowNodeRun, pbj.WorkflowNodeRunID),
			telemetry.Tag(telemetry.TagJob, pbj.Job.Action.Name))

		// Checks that the token used by the worker cas access to one of the execgroups
		grantedGroupIDs := append(getUserConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID)
		if !pbj.ExecGroups.HasOneOf(grantedGroupIDs...) {
			return sdk.WrapError(sdk.ErrForbidden, "worker %s (%s) is not authorized to take this job:%d execGroups:%+v", wk.Name, workerModelName, id, pbj.ExecGroups)
		}

		pbji := &sdk.WorkflowNodeJobRunData{}
		report, err := api.takeJob(ctx, p, id, workerModelName, pbji, wk, hatcheryName)
		if err != nil {
			return sdk.WrapError(err, "cannot takeJob nodeJobRunID:%d", id)
		}

		// FIXME remove CDN info from payload, this information should be injected by the hatchery
		pbji.GelfServiceAddr, pbji.GelfServiceAddrEnableTLS, err = services.GetCDNPublicTCPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}
		pbji.CDNHttpAddr, err = services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}

		workflow.ResyncNodeRunsWithCommits(api.Router.Background, api.mustDBWithCtx(api.Router.Background), api.Cache, *p, report)
		api.GoRoutines.Exec(api.Router.Background, "workflow-send-event", func(ctx context.Context) {
			api.WorkflowSendEvent(ctx, *p, report)
		})

		return service.WriteJSON(w, pbji, http.StatusOK)
	}
}

func (api *API) takeJob(ctx context.Context, p *sdk.Project, id int64, workerModel string, wnjri *sdk.WorkflowNodeJobRunData, wk *sdk.Worker, hatcheryName string) (*workflow.ProcessorReport, error) {
	tx, err := api.mustDB().Begin()
	if err != nil {
		return nil, sdk.WrapError(err, "cannot start transaction")
	}
	defer tx.Rollback() // nolint

	// Prepare spawn infos
	infos := []sdk.SpawnInfo{{
		RemoteTime: getRemoteTime(ctx),
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID, Args: []interface{}{fmt.Sprintf("%d", id), wk.Name}},
	}, {
		RemoteTime: getRemoteTime(ctx),
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTakenWorkerVersion.ID, Args: []interface{}{wk.Name, wk.Version, wk.OS, wk.Arch}},
	}}

	// Take node job run
	job, report, err := workflow.TakeNodeJobRun(ctx, tx, api.Cache, *p, id, workerModel, wk.Name, wk.ID, infos, hatcheryName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot take job %d", id)
	}

	workerKey, err := jws.NewRandomSymmetricKey(32)
	if err != nil {
		return nil, err
	}

	// Change worker status
	if err := worker.SetToBuilding(ctx, tx, wk.ID, job.ID, workerKey); err != nil {
		return nil, sdk.WrapError(err, "cannot update worker %s status", wk.Name)
	}
	wnjri.SigningKey = base64.StdEncoding.EncodeToString(workerKey)

	// Load the node run
	noderun, err := workflow.LoadNodeRunByID(ctx, tx, job.WorkflowNodeRunID, workflow.LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get node run")
	}

	if noderun.Status == sdk.StatusWaiting {
		noderun.Status = sdk.StatusBuilding
		if err := workflow.UpdateNodeRun(tx, noderun); err != nil {
			return nil, sdk.WrapError(err, "cannot update node run")
		}
		report.Add(ctx, *noderun)
	}

	// Load workflow run
	workflowRun, err := workflow.LoadRunByID(ctx, tx, noderun.WorkflowRunID, workflow.LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "unable to load workflow run")
	}

	secrets, err := workflow.LoadDecryptSecrets(ctx, tx, workflowRun, noderun)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load secrets")
	}

	// Feed the worker
	wnjri.ProjectKey = p.Key
	wnjri.NodeJobRun = *job
	wnjri.Number = noderun.Number
	wnjri.SubNumber = noderun.SubNumber
	wnjri.Secrets = make([]sdk.Variable, 0, len(secrets))
	wnjri.RunID = workflowRun.ID
	wnjri.WorkflowID = workflowRun.WorkflowID
	wnjri.WorkflowName = workflowRun.Workflow.Name
	wnjri.NodeRunName = noderun.WorkflowNodeName
	wnjri.AscodeActions = workflowRun.Workflow.AscodeActions

	secretsReqs := job.Job.Action.Requirements.FilterByType(sdk.SecretRequirement).Values()
	secretsReqsRegs := make([]*regexp.Regexp, 0, len(secretsReqs))
	for i := range secretsReqs {
		r, err := regexp.Compile(secretsReqs[i])
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		secretsReqsRegs = append(secretsReqsRegs, r)
	}

	// Filter project's secrets depending of the region requirement that was set on job
	skipProjectSecrets := job.Region != nil && sdk.IsInArray(*job.Region, api.Config.Secrets.SkipProjectSecretsOnRegion)
	if skipProjectSecrets {
		if err := workflow.AddSpawnInfosNodeJobRun(tx, job.WorkflowNodeRunID, job.ID, []sdk.SpawnInfo{{
			RemoteTime: getRemoteTime(ctx),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoDisableSecretInjection.ID, Args: []interface{}{*job.Region}},
		}}); err != nil {
			return nil, sdk.WrapError(err, "cannot save spawn info job %d", job.ID)
		}
	}

	var countMatchedSecrets int
	for i := range secrets {
		if skipProjectSecrets && secrets[i].Context == workflow.SecretProjContext {
			var inRequirements bool
			for _, reg := range secretsReqsRegs {
				if reg.MatchString(secrets[i].Name) {
					inRequirements = true
					break
				}
			}
			if !inRequirements {
				continue
			}
			countMatchedSecrets++
		}
		wnjri.Secrets = append(wnjri.Secrets, secrets[i].ToVariable())
	}

	if skipProjectSecrets && len(secretsReqs) > 0 {
		if err := workflow.AddSpawnInfosNodeJobRun(tx, job.WorkflowNodeRunID, job.ID, []sdk.SpawnInfo{{
			RemoteTime: getRemoteTime(ctx),
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoManualSecretInjection.ID, Args: []interface{}{fmt.Sprintf("%d", countMatchedSecrets)}},
		}}); err != nil {
			return nil, sdk.WrapError(err, "cannot save spawn info job %d", job.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WithStack(err)
	}

	return report, nil
}

func (api *API) postBookWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if ok, err := isHatchery(ctx); err != nil {
			return err
		} else if !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		s, err := services.LoadByID(ctx, api.mustDB(), getUserConsumer(ctx).AuthConsumerUser.Service.ID)
		if err != nil {
			return err
		}

		if _, err := workflow.BookNodeJobRun(ctx, api.Cache, api.Config.Workflow.JobDefaultBookDelay, api.Config.Workflow.CustomServiceJobBookDelay, id, s); err != nil {
			return sdk.WrapError(err, "job already booked")
		}

		jobRun, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, id)
		if err != nil {
			return err
		}
		wnr, err := workflow.LoadNodeRunByID(ctx, api.mustDB(), jobRun.WorkflowNodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return err
		}
		wr, err := workflow.LoadRunByID(ctx, api.mustDB(), wnr.WorkflowRunID, workflow.LoadRunOptions{})
		if err != nil {
			return err
		}

		resp := sdk.WorkflowNodeJobRunBooked{
			ProjectKey:   wr.Workflow.ProjectKey,
			WorkflowName: wr.Workflow.Name,
			WorkflowID:   wr.WorkflowID,
			RunID:        wr.ID,
			NodeRunName:  wnr.WorkflowNodeName,
			NodeRunID:    wnr.ID,
			JobName:      jobRun.Job.Action.Name,
		}
		return service.WriteJSON(w, resp, http.StatusOK)
	}
}

func (api *API) deleteBookWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if ok, err := isHatchery(ctx); err != nil {
			return err
		} else if !ok {
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

func (api *API) postSpawnInfosWorkflowJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return sdk.WrapError(err, "invalid id")
		}

		hatchery, err := isHatchery(ctx)
		if err != nil {
			return err
		}
		if ok := hatchery || isWorker(ctx); !ok {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		telemetry.Current(ctx, telemetry.Tag(telemetry.TagWorkflowNodeJobRun, id))

		var s []sdk.SpawnInfo
		if err := service.UnmarshalBody(r, &s); err != nil {
			return sdk.WrapError(err, "cannot unmarshal request")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		jobRun, err := workflow.LoadNodeJobRun(ctx, tx, api.Cache, id)
		if err != nil {
			return err
		}

		if err := workflow.AddSpawnInfosNodeJobRun(tx, jobRun.WorkflowNodeRunID, jobRun.ID, s); err != nil {
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

		var wk *sdk.Worker
		var hatch *sdk.Service
		if isWorker(ctx) {
			wk, err = worker.LoadByID(ctx, api.mustDB(), getUserConsumer(ctx).AuthConsumerUser.Worker.ID)
			if err != nil {
				return err
			}
		}

		if ok, err := isHatchery(ctx); err != nil {
			return err
		} else if ok {
			hatch, err = services.LoadByID(ctx, api.mustDBWithCtx(ctx), getUserConsumer(ctx).AuthConsumerUser.Service.ID)
			if err != nil {
				return err
			}
		}

		if wk == nil && hatch == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Unmarshal into results
		var res sdk.Result
		if err := service.UnmarshalBody(r, &res); err != nil {
			return sdk.WrapError(err, "cannot unmarshal request")
		}

		customCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
		defer cancel()
		proj, err := project.LoadProjectByNodeJobRunID(ctx, api.mustDBWithCtx(customCtx), api.Cache, id,
			project.LoadOptions.WithVariables,
			project.LoadOptions.WithGroups,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load project from job %d", id)
		}

		telemetry.Current(ctx, telemetry.Tag(telemetry.TagProjectKey, proj.Key))
		ctx = context.WithValue(ctx, log.Field("action_metadata_project_key"), proj.Key)

		// Start the transaction
		tx, err := api.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Load workflow node job run
		job, err := workflow.LoadAndLockNodeJobRunSkipLocked(ctx, tx, api.Cache, res.BuildID)
		if err != nil {
			return sdk.WrapError(err, "cannot load node run job %d", res.BuildID)
		}

		// If the call is made by an hatchery, we have to check that the job is at status waiting
		if hatch != nil {
			if job.Status != sdk.StatusWaiting {
				return sdk.WithStack(sdk.ErrForbidden)
			}
		}

		// Let's work
		report, err := api.postJobResult(customCtx, tx, proj, job, wk, hatch, &res)
		if err != nil {
			return sdk.WrapError(err, "unable to post job result")
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		workflowRuns := report.WorkflowRuns()
		if len(workflowRuns) > 0 {
			telemetry.Current(ctx,
				telemetry.Tag(telemetry.TagWorkflow, workflowRuns[0].Workflow.Name))

			if workflowRuns[0].Status == sdk.StatusFail {
				telemetry.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
			}
		}

		workflow.ResyncNodeRunsWithCommits(api.Router.Background, api.mustDBWithCtx(api.Router.Background), api.Cache, *proj, report)

		api.GoRoutines.Exec(api.Router.Background, "workflow-send-event", func(ctx context.Context) {
			api.WorkflowSendEvent(ctx, *proj, report)
		})

		for i := range report.WorkflowRuns() {
			run := &report.WorkflowRuns()[i]
			if err := api.updateParentWorkflowRun(ctx, run); err != nil {
				return sdk.WithStack(err)
			}
		}

		return nil
	}
}

func (api *API) postJobResult(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, proj *sdk.Project, job *sdk.WorkflowNodeJobRun, wr *sdk.Worker, hatch *sdk.Service, res *sdk.Result) (*workflow.ProcessorReport, error) {
	// Warning: here the worker "wk" of the hatchery "hatch" can be nil

	var end func()
	ctx, end = telemetry.Span(ctx, "postJobResult")
	defer end()

	telemetry.Current(ctx,
		telemetry.Tag(telemetry.TagWorkflowNodeJobRun, res.BuildID),
		telemetry.Tag(telemetry.TagWorkflowNodeRun, job.WorkflowNodeRunID),
		telemetry.Tag(telemetry.TagJob, job.Job.Action.Name))

	// Add spawn info
	if wr != nil {
		if err := workflow.AddSpawnInfosNodeJobRun(tx, job.WorkflowNodeRunID, job.ID, []sdk.SpawnInfo{{
			RemoteTime: res.RemoteTime,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{wr.Name}},
		}}); err != nil {
			return nil, sdk.WrapError(err, "Cannot save spawn info job %d", job.ID)
		}
	}
	if hatch != nil && res.Status == sdk.StatusFail {
		if err := workflow.AddSpawnInfosNodeJobRun(tx, job.WorkflowNodeRunID, job.ID, []sdk.SpawnInfo{{
			RemoteTime: res.RemoteTime,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnErrorHatcheryRetryAttempt.ID, Args: []interface{}{hatch.Name, res.Reason}},
		}}); err != nil {
			return nil, sdk.WrapError(err, "Cannot save spawn info job %d", job.ID)
		}

		// If the call is made by a hatchery, we have to update the stage status, because it is still at status waiting
		nodeRun, err := workflow.LoadNodeRunByID(ctx, tx, job.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			return nil, sdk.WrapError(err, "Unable to load node run id %d", job.WorkflowNodeRunID)
		}

		var stageIndex = nodeRun.GetStageIndex(job)
		if stageIndex == -1 {
			if err != nil {
				return nil, sdk.NewError(sdk.ErrWrongRequest, sdk.WithStack(fmt.Errorf("stage not found")))
			}
		}
		var stage = &nodeRun.Stages[stageIndex]
		if stage.Status == sdk.StatusWaiting {
			stage.Status = sdk.StatusBuilding
		}
		if nodeRun.Status == sdk.StatusWaiting {
			nodeRun.Status = sdk.StatusBuilding
		}
		// Save the node run in database
		if err := workflow.UpdateNodeRunStatusAndStage(tx, nodeRun); err != nil {
			return nil, sdk.WrapError(err, "unable to update node id=%d at status %s", nodeRun.ID, nodeRun.Status)
		}
	}

	// Manage build variables, we have to push them on the job and to propagate on the node above
	for _, v := range res.NewVariables {
		log.Debug(ctx, "postJobResult> managing new variable %s on job %d", v.Name, job.ID)
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
			log.Debug(ctx, "postJobResult> adding new variable %s on job %d", v.Name, job.ID)
			sdk.AddParameter(&job.Parameters, v.Name, sdk.StringParameter, v.Value)
		}
	}

	if err := workflow.UpdateNodeJobRun(ctx, tx, job); err != nil {
		return nil, sdk.WrapError(err, "Unable to update node job run %d", res.BuildID)
	}

	// Handle build variables sent by the worker
	if len(res.NewVariables) > 0 {
		nodeRun, err := workflow.LoadAndLockNodeRunByID(ctx, tx, job.WorkflowNodeRunID)
		if err != nil {
			return nil, err
		}
		mustUpdateNodeRunParams := false

		for _, v := range res.NewVariables {
			log.Debug(ctx, "postJobResult> managing new variable %s on node %d", v.Name, nodeRun.ID)
			found := false
			for i := range nodeRun.BuildParameters {
				currentV := &nodeRun.BuildParameters[i]
				if currentV.Name == v.Name {
					currentV.Value = v.Value
					found = true
					break
				}
			}
			if !found {
				log.Debug(ctx, "postJobResult> add new key on node run %s", v.Name)
				mustUpdateNodeRunParams = true
				sdk.AddParameter(&nodeRun.BuildParameters, v.Name, sdk.StringParameter, v.Value)
			}
		}

		if mustUpdateNodeRunParams {
			if err := workflow.UpdateNodeRunBuildParameters(tx, nodeRun.ID, nodeRun.BuildParameters); err != nil {
				return nil, sdk.WrapError(err, "unable to update node run %d", nodeRun.ID)
			}
		}
	}
	// ^ build variables are now updated on job run and on node

	var reloadedWorker *sdk.Worker
	if wr != nil {
		//Update worker status
		if err := worker.SetStatus(ctx, tx, wr.ID, sdk.StatusWaiting); err != nil {
			return nil, sdk.WrapError(err, "cannot update worker %s status", wr.ID)
		}
		var err error
		reloadedWorker, err = worker.LoadByID(ctx, tx, wr.ID)
		if err != nil {
			log.Error(ctx, "unable to reload worker %q: %v", wr.ID, err)
		}
	}

	// Update action status
	log.Debug(ctx, "postJobResult> Updating %d to %s in queue", job.ID, res.Status)
	report, err := workflow.UpdateNodeJobRunStatus(ctx, tx, api.Cache, *proj, job, res.Status)
	if err != nil {
		// Checking the error:
		// pq: update or delete on table "workflow_node_run_job" violates foreign key constraint "fk_worker_workflow_node_run_job" on table "worker")
		if wr != nil && strings.Contains(err.Error(), "fk_worker_workflow_node_run_job") {
			log.ErrorWithStackTrace(ctx, err)
			if reloadedWorker != nil {
				log.Error(ctx, "reloaded worker %s (%s) status is %q", reloadedWorker.Name, reloadedWorker.ID, reloadedWorker.Status)
			}
		}
		return nil, sdk.WrapError(err, "cannot update NodeJobRun %d status", job.ID)
	}

	return report, nil
}

func (api *API) getWorkerCacheLinkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}
		workerTag := vars["tag"]

		p, err := project.LoadProjectByNodeJobRunID(ctx, api.mustDBWithCtx(ctx), api.Cache, id)
		if err != nil {
			return err
		}

		itemsLinks, err := cdn.ListItems(ctx, api.mustDBWithCtx(ctx), sdk.CDNTypeItemWorkerCache, map[string]string{
			cdn.ParamProjectKey: p.Key,
			cdn.ParamCacheTag:   string(workerTag),
		})
		if err != nil {
			return err
		}
		return service.WriteJSON(w, itemsLinks, http.StatusOK)
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
				log.Warn(ctx, "postWorkflowJobStepStatusHandler> sync doesn't find a nodeJobRun. JobID on handler: %d", id)
			}
			if err := workflow.UpdateNodeRun(tx, nodeRun); err != nil {
				return sdk.WrapError(err, "cannot update node run. JobID on handler: %d", id)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if nodeRun.ID == 0 {
			nodeRunP, err := workflow.LoadNodeRunByID(ctx, api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
			if err != nil {
				log.Warn(ctx, "postWorkflowJobStepStatusHandler> Unable to load node run for event: %v", err)
				return nil
			}
			nodeRun = *nodeRunP
		}

		work, err := workflow.LoadWorkflowFromWorkflowRunID(api.mustDB(), nodeRun.WorkflowRunID)
		if err != nil {
			log.Warn(ctx, "postWorkflowJobStepStatusHandler> Unable to load workflow for event: %v", err)
			return nil
		}

		wr, err := workflow.LoadRunByID(ctx, api.mustDB(), nodeRun.WorkflowRunID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			log.Warn(ctx, "postWorkflowJobStepStatusHandler> Unable to load workflow run for event: %v", err)
			return nil
		}
		nodeRun.Translate()
		eventsNotifs := notification.GetUserWorkflowEvents(ctx, api.mustDB(), api.Cache, wr.Workflow.ProjectID, wr.Workflow.ProjectKey, work.Name, wr.Workflow.Notifications, nil, nodeRun)
		event.PublishWorkflowNodeRun(context.Background(), nodeRun, wr.Workflow, eventsNotifs)
		return nil
	}
}

func (api *API) countWorkflowJobQueueHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getUserConsumer(ctx)

		var count sdk.WorkflowNodeJobRunCount

		if consumer.AuthConsumerUser.Worker != nil {
			if consumer.AuthConsumerUser.Worker.JobRunID != nil {
				count.Count = 1
			}
			return service.WriteJSON(w, count, http.StatusOK)
		}

		since, until, _ := getSinceUntilLimitHeader(ctx, w, r)
		modelType, err := getModelType(ctx, r)
		if err != nil {
			return err
		}

		filter := workflow.NewQueueFilter()
		filter.ModelType = []string{modelType}
		filter.Since = &since
		filter.Until = &until

		if !isMaintainer(ctx) {
			count, err = workflow.CountNodeJobRunQueueByGroupIDs(ctx, api.mustDB(), api.Cache, filter, getUserConsumer(ctx).GetGroupIDs())
		} else {
			count, err = workflow.CountNodeJobRunQueue(ctx, api.mustDB(), api.Cache, filter)
		}
		if err != nil {
			return sdk.WrapError(err, "unable to count queue")
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
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given status")
		}
		if len(status) == 0 {
			status = []string{sdk.StatusWaiting}
		}

		modelType, err := getModelType(ctx, r)
		if err != nil {
			return err
		}

		regions, err := QueryStrings(r, "region")
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}

		consumer := getUserConsumer(ctx)

		jobs := make([]sdk.WorkflowNodeJobRun, 0)

		if consumer.AuthConsumerUser.Worker != nil {
			if consumer.AuthConsumerUser.Worker.JobRunID != nil {
				job, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, *consumer.AuthConsumerUser.Worker.JobRunID)
				if err != nil {
					return err
				}
				jobs = []sdk.WorkflowNodeJobRun{*job}
			}
			return service.WriteJSON(w, jobs, http.StatusOK)
		}

		isS := isService(ctx)
		permissions := sdk.PermissionRead
		if isS {
			permissions = sdk.PermissionReadExecute
		}
		filter := workflow.NewQueueFilter()
		filter.Since = &since
		filter.Until = &until
		filter.Rights = permissions
		filter.Statuses = status
		filter.Limit = &limit
		filter.Regions = regions
		if modelType != "" {
			filter.ModelType = []string{modelType}
		}
		if ok, _ := isHatchery(ctx); ok {
			filter.SkipBooked = true
		}

		// If the consumer is a hatchery or a non maintainer user, filter the job by its groups
		if isS || !isMaintainer(ctx) {
			jobs, err = workflow.LoadNodeJobRunQueueByGroupIDs(ctx, api.mustDB(), api.Cache, filter, getUserConsumer(ctx).GetGroupIDs())
		} else {
			jobs, err = workflow.LoadNodeJobRunQueue(ctx, api.mustDB(), api.Cache, filter)
		}
		if err != nil {
			return sdk.WrapError(err, "unable to load queue")
		}

		return service.WriteJSON(w, jobs, http.StatusOK)
	}
}

func getModelType(ctx context.Context, r *http.Request) (string, error) {
	modelType := FormString(r, "modelType")
	if modelType != "" {
		if !sdk.WorkerModelValidate(modelType) {
			return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given modelType")
		}
	}
	return modelType, nil
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

func (api *API) postWorkflowJobTestsResultsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Unmarshal into results
		var new sdk.JUnitTestsSuites
		if err := service.UnmarshalBody(r, &new); err != nil {
			return err
		}
		new = new.EnsureData()

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
			nr.Tests = &sdk.TestsResults{}
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

		nr.Tests.JUnitTestsSuites = nr.Tests.JUnitTestsSuites.EnsureData()
		nr.Tests.TestsStats = nr.Tests.JUnitTestsSuites.ComputeStats()

		if err := workflow.UpdateNodeRun(tx, nr); err != nil {
			return sdk.WrapError(err, "cannot update node run")
		}

		// If we are on default branch, push metrics
		if nr.VCSServer != "" && nr.VCSBranch != "" {
			p, err := project.LoadProjectByNodeJobRunID(ctx, tx, api.Cache, id)
			if err != nil {
				log.Error(ctx, "postWorkflowJobTestsResultsHandler> Cannot load project by nodeJobRunID %d: %v", id, err)
				return nil
			}

			// Get vcs info to known if we are on the default branch or not
			client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, p.Key, nr.VCSServer)
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
				metrics.PushUnitTests(p.Key, nr.ApplicationID, nr.WorkflowID, nr.Number, nr.Tests.TestsStats)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot update node run")
		}

		return nil
	}
}

func (api *API) workflowRunResultCheckUploadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		jobID, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if !isCDN(ctx) && !isWorker(ctx) {
			return sdk.WrapError(sdk.ErrForbidden, "only CDN and worker can call this route")
		}

		wr, err := workflow.LoadRunByJobID(ctx, api.mustDBWithCtx(ctx), jobID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return err
		}

		var runResultCheck sdk.WorkflowRunResultCheck
		if err := service.UnmarshalBody(r, &runResultCheck); err != nil {
			return sdk.WithStack(err)
		}

		b, err := workflow.CanUploadRunResult(ctx, api.mustDBWithCtx(ctx), api.Cache, *wr, runResultCheck)
		if err != nil {
			return err
		}
		if !b {
			return sdk.WrapError(sdk.ErrInvalidData, "unable to duplicate an artifact")
		}

		// Save check
		if err := api.Cache.SetWithTTL(workflow.GetRunResultKey(runResultCheck.RunID, runResultCheck.ResultType, runResultCheck.Name), true, 600); err != nil {
			return sdk.WrapError(err, "unable to cache result artifact check %s ", runResultCheck.Name)
		}
		return nil
	}
}

func (api *API) workflowRunResultPromoteHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isWorker(ctx) {
			return sdk.WrapError(sdk.ErrForbidden, "only workers can call this route")
		}

		jobID, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		var promoteRequest sdk.WorkflowRunResultPromotionRequest
		if err := service.UnmarshalBody(r, &promoteRequest); err != nil {
			return sdk.WithStack(err)
		}

		wr, err := workflow.LoadRunByJobID(ctx, api.mustDB(), jobID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return err
		}

		// Ensure that run result are synced with the Artifact Manager
		txSync, err := api.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer txSync.Rollback()
		if err := workflow.SyncRunResultArtifactManagerByRunID(ctx, txSync, wr.ID); err != nil {
			return err
		}
		if err := txSync.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		// Process promote request
		txPromote, err := api.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer txPromote.Rollback()
		if err := workflow.ProcessRunResultPromotionByRunID(ctx, txPromote, wr.ID, sdk.WorkflowRunResultPromotionTypePromote, promoteRequest); err != nil {
			return err
		}
		if err := txPromote.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) workflowRunResultReleaseHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !isWorker(ctx) {
			return sdk.WrapError(sdk.ErrForbidden, "only workers can call this route")
		}

		jobID, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		var releaseRequest sdk.WorkflowRunResultPromotionRequest
		if err := service.UnmarshalBody(r, &releaseRequest); err != nil {
			return sdk.WithStack(err)
		}

		wr, err := workflow.LoadRunByJobID(ctx, api.mustDB(), jobID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return err
		}

		// Ensure that run result are synced with the Artifact Manager
		txSync, err := api.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer txSync.Rollback()
		if err := workflow.SyncRunResultArtifactManagerByRunID(ctx, txSync, wr.ID); err != nil {
			return err
		}
		if err := txSync.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		// Process release request
		txRelease, err := api.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer txRelease.Rollback()
		if err := workflow.ProcessRunResultPromotionByRunID(ctx, txRelease, wr.ID, sdk.WorkflowRunResultPromotionTypeRelease, releaseRequest); err != nil {
			return err
		}
		if err := txRelease.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}

func (api *API) postWorkflowRunResultsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		jobID, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		if !isCDN(ctx) && !isWorker(ctx) {
			return sdk.WrapError(sdk.ErrForbidden, "only CDN and worker can call this route")
		}

		var runResult sdk.WorkflowRunResult
		if err := service.UnmarshalBody(r, &runResult); err != nil {
			return sdk.WithStack(err)
		}

		wr, err := workflow.LoadRunByJobID(ctx, api.mustDBWithCtx(ctx), jobID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return err
		}

		if wr.ID != runResult.WorkflowRunID {
			return sdk.WrapError(sdk.ErrInvalidData, "unable to add artifact on this run: %s", runResult.ID)
		}

		nr, err := workflow.LoadNodeRunByID(ctx, api.mustDB(), runResult.WorkflowNodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			return err
		}
		if nr.WorkflowRunID != wr.ID {
			return sdk.WrapError(sdk.ErrInvalidData, "invalid node run %d", runResult.WorkflowNodeRunID)
		}
		runResult.SubNum = nr.SubNumber

		if err := workflow.AddResult(ctx, api.mustDBWithCtx(ctx), api.Cache, wr, &runResult); err != nil {
			return err
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

		workflowRun, err := workflow.LoadAndLockRunByJobID(ctx, tx, id, workflow.LoadRunOptions{})
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.NewErrorFrom(sdk.ErrLocked, "workflow run is already locked")
			}
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

func (api *API) postWorkflowJobSetVersionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		id, err := requestVarInt(r, "permJobID")
		if err != nil {
			return err
		}

		var data sdk.WorkflowRunVersion
		if err := service.UnmarshalBody(r, &data); err != nil {
			return sdk.WithStack(err)
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() // nolint

		workflowRun, err := workflow.LoadAndLockRunByJobID(ctx, tx, id, workflow.LoadRunOptions{})
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.NewErrorFrom(sdk.ErrLocked, "workflow run is already locked")
			}
			return sdk.WrapError(err, "unable to load node run id %d", id)
		}

		if workflowRun.Version != nil && *workflowRun.Version != data.Value {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot change existing workflow run version value %q", *workflowRun.Version)
		}

		workflowRun.Version = &data.Value
		if err := workflow.UpdateWorkflowRun(ctx, tx, workflowRun); err != nil {
			if sdk.ErrorIs(err, sdk.ErrConflictData) {
				return sdk.NewErrorFrom(err, "version %q already used by another workflow run", data.Value)
			}
			return err
		}

		nodeRun, err := workflow.LoadAndLockNodeRunByJobID(ctx, tx, id)
		if err != nil {
			return err
		}
		for i := range nodeRun.BuildParameters {
			if nodeRun.BuildParameters[i].Name == "cds.version" {
				nodeRun.BuildParameters[i].Value = data.Value
				break
			}
		}
		if err := workflow.UpdateNodeRunBuildParameters(tx, nodeRun.ID, nodeRun.BuildParameters); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit transaction")
		}

		return nil
	}
}
