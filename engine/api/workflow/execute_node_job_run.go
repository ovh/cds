package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// ProcessorReport represents the state of the workflow processor
type ProcessorReport struct {
	mutex     sync.Mutex
	jobs      []sdk.WorkflowNodeJobRun
	nodes     []sdk.WorkflowNodeRun
	workflows []sdk.WorkflowRun
	errors    []error
}

func (r *ProcessorReport) Jobs() []sdk.WorkflowNodeJobRun {
	return r.jobs
}

func (r *ProcessorReport) Nodes() []sdk.WorkflowNodeRun {
	return r.nodes
}

func (r *ProcessorReport) Workflows() []sdk.WorkflowRun {
	return r.workflows
}

// WorkflowRuns returns the list of concerned workflow runs
func (r *ProcessorReport) WorkflowRuns() []sdk.WorkflowRun {
	if r == nil {
		return nil
	}
	return r.workflows
}

// Add something to the report
func (r *ProcessorReport) Add(ctx context.Context, i ...interface{}) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, w := range i {
		switch x := w.(type) {
		case error:
			r.errors = append(r.errors, x)
		case sdk.WorkflowNodeJobRun:
			r.jobs = append(r.jobs, x)
		case *sdk.WorkflowNodeJobRun:
			r.jobs = append(r.jobs, *x)
		case sdk.WorkflowNodeRun:
			r.addWorkflowNodeRun(x)
		case *sdk.WorkflowNodeRun:
			r.addWorkflowNodeRun(*x)
		case sdk.WorkflowRun:
			r.workflows = append(r.workflows, x)
		case *sdk.WorkflowRun:
			r.workflows = append(r.workflows, *x)
		default:
			log.Warn(ctx, "ProcessorReport> unknown type %T", w)
		}
	}
}

func (r *ProcessorReport) addWorkflowNodeRun(nr sdk.WorkflowNodeRun) {
	for i := range r.nodes {
		if nr.ID == r.nodes[i].ID {
			r.nodes[i] = nr
			return
		}
	}
	r.nodes = append(r.nodes, nr)
}

// All returns all the objects in the reports
func (r *ProcessorReport) All() []interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	res := make([]interface{}, 0)
	res = append(res, sdk.InterfaceSlice(r.workflows)...)
	res = append(res, sdk.InterfaceSlice(r.nodes)...)
	res = append(res, sdk.InterfaceSlice(r.jobs)...)
	res = append(res, sdk.InterfaceSlice(r.errors)...)
	return res
}

// Merge to the provided report and the current report
func (r *ProcessorReport) Merge(ctx context.Context, r1 *ProcessorReport) {
	if r1 == nil {
		return
	}
	if r == nil {
		*r = ProcessorReport{}
	}
	r.Add(ctx, r1.All()...)
}

// Errors return errors
func (r *ProcessorReport) Errors() []error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.errors) > 0 {
		return r.errors
	}
	return nil
}

// UpdateNodeJobRunStatus Update status of an workflow_node_run_job.
func UpdateNodeJobRunStatus(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, job *sdk.WorkflowNodeJobRun, status string) (*ProcessorReport, error) {
	ctx, end := telemetry.Span(ctx, "workflow.UpdateNodeJobRunStatus",
		telemetry.Tag(telemetry.TagWorkflowNodeJobRun, job.ID),
		telemetry.Tag("workflow_node_run_job_status", status),
	)
	defer end()

	report := new(ProcessorReport)

	_, next := telemetry.Span(ctx, "workflow.LoadRunByID")
	nodeRun, err := LoadNodeRunByID(ctx, db, job.WorkflowNodeRunID, LoadRunOptions{})
	next()
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to load node run id %d", job.WorkflowNodeRunID)
	}

	query := `SELECT status FROM workflow_node_run_job WHERE id = $1`
	var currentStatus string
	if err := db.QueryRow(query, job.ID).Scan(&currentStatus); err != nil {
		return nil, sdk.WrapError(err, "Cannot select status from workflow_node_run_job node job run %d", job.ID)
	}

	log.Info(ctx, "job %d current status %q, new status %q", job.ID, status, currentStatus)

	switch status {
	case sdk.StatusBuilding:
		if currentStatus != sdk.StatusWaiting {
			return nil, sdk.WithStack(fmt.Errorf("cannot update status of WorkflowNodeJobRun %d to %s, expected current status %s, got %s",
				job.ID, status, sdk.StatusWaiting, currentStatus))
		}
		job.Start = time.Now()
		job.Status = status

	case sdk.StatusFail, sdk.StatusSuccess, sdk.StatusDisabled, sdk.StatusSkipped, sdk.StatusStopped:
		if currentStatus != sdk.StatusWaiting && currentStatus != sdk.StatusBuilding && status != sdk.StatusDisabled && status != sdk.StatusSkipped {
			log.Debug(ctx, "workflow.UpdateNodeJobRunStatus> Status is %s, cannot update %d to %s", currentStatus, job.ID, status)
			// too late, Nate
			return nil, nil
		}
		job.Done = time.Now()
		job.Status = status

		wf, err := LoadRunByID(ctx, db, nodeRun.WorkflowRunID, LoadRunOptions{
			WithDeleted: true,
		})
		if err != nil {
			return nil, sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Unable to load run id %d", nodeRun.WorkflowRunID)
		}

		wf.LastExecution = time.Now()
		if err := UpdateWorkflowRun(ctx, db, wf); err != nil {
			return nil, sdk.WrapError(err, "Cannot update WorkflowRun %d", wf.ID)
		}
	default:
		return nil, sdk.WithStack(fmt.Errorf("cannot update WorkflowNodeJobRun %d to status %v", job.ID, status))
	}

	if err := UpdateNodeJobRun(ctx, db, job); err != nil {
		return nil, sdk.WrapError(err, "Cannot update WorkflowNodeJobRun %d", job.ID)
	}

	report.Add(ctx, *job)

	//If the job has been set to building, set the stage to building
	var stageIndex = nodeRun.GetStageIndex(job)
	if status == sdk.StatusBuilding {
		// Sync job status in noderun
		r, err := syncTakeJobInNodeRun(ctx, db, nodeRun, job, stageIndex)
		report.Merge(ctx, r)
		return report, err
	}

	spawnInfos, err := LoadNodeRunJobInfo(ctx, db, nodeRun.ID, job.ID)
	if err != nil {
		return report, sdk.WrapError(err, "unable to load spawn infos for runJob: %d", job.ID)
	}
	job.SpawnInfos = spawnInfos

	syncJobInNodeRun(nodeRun, job, stageIndex)

	if job.Status != sdk.StatusStopped {
		r, err := executeNodeRun(ctx, db, store, proj, nodeRun)
		report.Merge(ctx, r)
		return report, err
	}

	return report, nil
}

// AddSpawnInfosNodeJobRun saves spawn info before starting worker
func AddSpawnInfosNodeJobRun(db gorp.SqlExecutor, nodeID, jobID int64, infos []sdk.SpawnInfo) error {
	wnjri := &sdk.WorkflowNodeJobRunInfo{
		WorkflowNodeRunID:    nodeID,
		WorkflowNodeJobRunID: jobID,
		SpawnInfos:           PrepareSpawnInfos(infos),
	}
	if err := insertNodeRunJobInfo(db, wnjri); err != nil {
		return sdk.WrapError(err, "Cannot update node job run infos %d", jobID)
	}
	return nil
}

// PrepareSpawnInfos helps yoi to create sdk.SpawnInfo array
func PrepareSpawnInfos(infos []sdk.SpawnInfo) []sdk.SpawnInfo {
	now := time.Now()
	prepared := make([]sdk.SpawnInfo, 0)
	for _, info := range infos {
		preparedInfo := sdk.SpawnInfo{
			APITime:     now,
			RemoteTime:  info.RemoteTime,
			Message:     info.Message,
			UserMessage: info.Message.DefaultUserMessage(),
		}
		if preparedInfo.RemoteTime.IsZero() {
			preparedInfo.RemoteTime = now
		}
		prepared = append(prepared, preparedInfo)
	}
	return prepared
}

// TakeNodeJobRun Take an a job run for update
func TakeNodeJobRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, jobID int64,
	workerModel, workerName, workerID string, infos []sdk.SpawnInfo, hatcheryName string) (*sdk.WorkflowNodeJobRun, *ProcessorReport, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.TakeNodeJobRun")
	defer end()

	report := new(ProcessorReport)

	// first load without FOR UPDATE WAIT to quick check status
	currentStatus, err := db.SelectStr(`SELECT status FROM workflow_node_run_job WHERE id = $1`, jobID)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "cannot select status from workflow_node_run_job node job run %d", jobID)
	}

	if err := checkStatusWaiting(ctx, store, jobID, currentStatus); err != nil {
		return nil, nil, err
	}

	// reload and recheck status
	job, err := LoadAndLockNodeJobRunSkipLocked(ctx, db, store, jobID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrLocked) {
			err = sdk.NewErrorWithStack(err, sdk.ErrJobAlreadyBooked)
		}
		return nil, nil, sdk.WrapError(err, "cannot load node job run (WAIT) %d", jobID)
	}
	if err := checkStatusWaiting(ctx, store, jobID, job.Status); err != nil {
		return nil, report, err
	}

	job.HatcheryName = hatcheryName
	job.WorkerName = workerName
	job.Model = workerModel
	job.Job.WorkerName = workerName
	job.Job.WorkerID = workerID
	job.Start = time.Now()

	if _, err := db.Exec("UPDATE workflow_node_run_job SET worker_id = $2 WHERE id = $1", job.ID, workerID); err != nil {
		return nil, nil, sdk.WrapError(err, "cannot update worker_id in node job run %d", jobID)
	}

	if err := AddSpawnInfosNodeJobRun(db, job.WorkflowNodeRunID, jobID, infos); err != nil {
		return nil, nil, sdk.WrapError(err, "cannot save spawn info on node job run %d", jobID)
	}

	r, err := UpdateNodeJobRunStatus(ctx, db, store, proj, job, sdk.StatusBuilding)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "cannot update node job run %d status from %s to %s", job.ID, job.Status, sdk.StatusBuilding)
	}
	report.Merge(ctx, r)

	return job, report, nil
}

func checkStatusWaiting(ctx context.Context, store cache.Store, jobID int64, status string) error {
	if status != sdk.StatusWaiting {
		k := keyBookJob(jobID)
		h := sdk.Service{}
		find, err := store.Get(k, &h)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
		if find {
			return sdk.WrapError(sdk.ErrAlreadyTaken, "job %d is not waiting status and was booked by hatchery %d. current status: %s", jobID, h.ID, status)
		}
		return sdk.WrapError(sdk.ErrAlreadyTaken, "job %d is not waiting status. current status: %s", jobID, status)
	}
	return nil
}

// LoadDecryptSecrets loads all secrets for a job run
func LoadDecryptSecrets(ctx context.Context, db gorp.SqlExecutor, wr *sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun) (sdk.WorkflowRunSecrets, error) {
	entities := []string{SecretProjContext}

	for _, integ := range wr.Workflow.Integrations {
		if integ.ProjectIntegration.Model.Event {
			continue
		}
		entities = append(entities, fmt.Sprintf(SecretProjIntegrationContext, integ.ProjectIntegrationID))
	}

	if nodeRun != nil {
		node := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)
		if node == nil {
			return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "unable to find node %d in worflow run", nodeRun.WorkflowNodeID)
		}

		if node.Context != nil {
			if node.Context.ApplicationID != 0 {
				entities = append(entities, fmt.Sprintf(SecretAppContext, node.Context.ApplicationID))
			}

			if node.Context.EnvironmentID != 0 {
				entities = append(entities, fmt.Sprintf(SecretEnvContext, node.Context.EnvironmentID))
			}

			if node.Context.ProjectIntegrationID != 0 {
				entities = append(entities, fmt.Sprintf(SecretProjIntegrationContext, node.Context.ProjectIntegrationID))

				if node.Context.ApplicationID != 0 {
					entities = append(entities, fmt.Sprintf(SecretApplicationIntegrationContext, node.Context.ApplicationID, wr.Workflow.ProjectIntegrations[node.Context.ProjectIntegrationID].Name))
				}
			}
		}
	}

	secrets, err := loadRunSecretWithDecryption(ctx, db, wr.ID, entities)
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// BookNodeJobRun  Book a job for a hatchery
func BookNodeJobRun(ctx context.Context, store cache.Store, defaultBookDelay int64, customBookDelay map[string]int64, id int64, hatchery *sdk.Service) (*sdk.Service, error) {
	k := keyBookJob(id)
	h := sdk.Service{}
	find, err := store.Get(k, &h)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	if !find {
		// job not already booked, book it for 2 min
		delay := 120
		if defaultBookDelay > 0 {
			delay = int(defaultBookDelay)
		}
		if customBookDelay != nil {
			if d, ok := customBookDelay[hatchery.Name]; ok {
				delay = int(d)
			}
		}
		if err := store.SetWithTTL(k, hatchery, delay); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
		return nil, nil
	}
	if h.ID == hatchery.ID {
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrJobAlreadyBooked, "BookNodeJobRun> job %d already booked by %s (%d)", id, h.Name, h.ID)
}

// FreeNodeJobRun  Free a job for a hatchery
func FreeNodeJobRun(ctx context.Context, store cache.Store, id int64) error {
	k := keyBookJob(id)
	h := sdk.Service{}
	find, err := store.Get(k, &h)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	if find {
		if err := store.Delete(k); err != nil {
			log.Error(ctx, "error on cache delete %v: %v", k, err)
		}
		return nil
	}
	return sdk.WrapError(sdk.ErrJobNotBooked, "BookNodeJobRun> job %d already released", id)
}

// RestartWorkflowNodeJob restart all workflow node job and update logs to indicate restart
func RestartWorkflowNodeJob(ctx context.Context, db gorp.SqlExecutor, wNodeJob sdk.WorkflowNodeJobRun, maxLogSize int64) error {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.RestartWorkflowNodeJob")
	defer end()

	for iS := range wNodeJob.Job.StepStatus {
		step := &wNodeJob.Job.StepStatus[iS]
		if step.Status == sdk.StatusNeverBuilt || step.Status == sdk.StatusSkipped || step.Status == sdk.StatusDisabled {
			continue
		}
		wNodeJob.Job.Reason = "Killed (Reason: Timeout)\n"
		step.Status = sdk.StatusWaiting
		step.Done = time.Time{}
	}

	nodeRun, err := LoadAndLockNodeRunByID(ctx, db, wNodeJob.WorkflowNodeRunID)
	if err != nil {
		return err
	}

	//Synchronize struct but not in db
	sync, err := SyncNodeRunRunJob(ctx, db, nodeRun, wNodeJob)
	if err != nil {
		return sdk.WrapError(err, "error on sync nodeJobRun")
	}
	if !sync {
		log.Warn(ctx, "sync doesn't find a nodeJobRun")
	}

	if err := UpdateNodeRun(db, nodeRun); err != nil {
		return sdk.WrapError(err, "cannot update node run")
	}

	if err := replaceWorkflowJobRunInQueue(db, wNodeJob); err != nil {
		return sdk.WrapError(err, "cannot replace workflow job in queue")
	}

	return nil
}
