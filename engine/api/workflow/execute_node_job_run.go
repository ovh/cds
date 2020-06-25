package workflow

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
			log.Warning(ctx, "ProcessorReport> unknown type %T", w)
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

//All returns all the objects in the reports
func (r *ProcessorReport) All() []interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	res := []interface{}{}
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
func UpdateNodeJobRunStatus(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, job *sdk.WorkflowNodeJobRun, status string) (*ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.UpdateNodeJobRunStatus",
		observability.Tag(observability.TagWorkflowNodeJobRun, job.ID),
		observability.Tag("workflow_node_run_job_status", status),
	)
	defer end()

	report := new(ProcessorReport)

	_, next := observability.Span(ctx, "workflow.LoadRunByID")
	nodeRun, errLoad := LoadNodeRunByID(db, job.WorkflowNodeRunID, LoadRunOptions{})
	next()
	if errLoad != nil {
		return nil, sdk.WrapError(errLoad, "Unable to load node run id %d", job.WorkflowNodeRunID)
	}

	query := `SELECT status FROM workflow_node_run_job WHERE id = $1`
	var currentStatus string
	if err := db.QueryRow(query, job.ID).Scan(&currentStatus); err != nil {
		return nil, sdk.WrapError(err, "Cannot select status from workflow_node_run_job node job run %d", job.ID)
	}

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
			log.Debug("workflow.UpdateNodeJobRunStatus> Status is %s, cannot update %d to %s", currentStatus, job.ID, status)
			// too late, Nate
			return nil, nil
		}
		job.Done = time.Now()
		job.Status = status

		_, next := observability.Span(ctx, "workflow.LoadRunByID")
		wf, errLoadWf := LoadRunByID(db, nodeRun.WorkflowRunID, LoadRunOptions{
			WithDeleted: true,
		})
		next()
		if errLoadWf != nil {
			return nil, sdk.WrapError(errLoadWf, "workflow.UpdateNodeJobRunStatus> Unable to load run id %d", nodeRun.WorkflowRunID)
		}

		wf.LastExecution = time.Now()
		if err := UpdateWorkflowRun(ctx, db, wf); err != nil {
			return nil, sdk.WrapError(err, "Cannot update WorkflowRun %d", wf.ID)
		}
	default:
		return nil, sdk.WithStack(fmt.Errorf("cannot update WorkflowNodeJobRun %d to status %v", job.ID, status))
	}

	//If the job has been set to building, set the stage to building
	var stageIndex int
	for i := range nodeRun.Stages {
		s := &nodeRun.Stages[i]
		for _, j := range s.Jobs {
			if j.Action.ID == job.Job.Job.Action.ID {
				stageIndex = i
			}
		}
	}

	if err := UpdateNodeJobRun(ctx, db, job); err != nil {
		return nil, sdk.WrapError(err, "Cannot update WorkflowNodeJobRun %d", job.ID)
	}

	report.Add(ctx, *job)

	if status == sdk.StatusBuilding {
		// Sync job status in noderun
		r, err := syncTakeJobInNodeRun(ctx, db, nodeRun, job, stageIndex)
		report.Merge(ctx, r)
		return report, err
	}

	spawnInfos, err := LoadNodeRunJobInfo(ctx, db, job.ID)
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
	return nil, nil
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
	prepared := []sdk.SpawnInfo{}
	for _, info := range infos {
		prepared = append(prepared, sdk.SpawnInfo{
			APITime:     now,
			RemoteTime:  info.RemoteTime,
			Message:     info.Message,
			UserMessage: info.Message.DefaultUserMessage(),
		})
	}
	return prepared
}

// TakeNodeJobRun Take an a job run for update
func TakeNodeJobRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj sdk.Project, jobID int64,
	workerModel, workerName, workerID string, infos []sdk.SpawnInfo, hatcheryName string) (*sdk.WorkflowNodeJobRun, *ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.TakeNodeJobRun")
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

	if err := AddSpawnInfosNodeJobRun(db, job.WorkflowNodeRunID, jobID, PrepareSpawnInfos(infos)); err != nil {
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

// LoadNodeJobRunKeys loads all keys for a job run
func LoadNodeJobRunKeys(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, wr *sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun) ([]sdk.Parameter, []sdk.Variable, error) {
	var app *sdk.Application
	var env *sdk.Environment

	n := wr.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)
	if n.Context.ApplicationID != 0 {
		appMap, has := wr.Workflow.Applications[n.Context.ApplicationID]
		if has {
			app = &appMap
			keys, err := application.LoadAllKeysWithPrivateContent(db, app.ID)
			if err != nil {
				return nil, nil, err
			}
			app.Keys = keys
		}
	}
	if n.Context.EnvironmentID != 0 {
		envMap, has := wr.Workflow.Environments[n.Context.EnvironmentID]
		if has {
			env = &envMap
			keys, err := environment.LoadAllKeysWithPrivateContent(db, n.Context.EnvironmentID)
			if err != nil {
				return nil, nil, err
			}
			env.Keys = keys
		}
	}

	params := []sdk.Parameter{}
	secrets := []sdk.Variable{}

	for _, k := range proj.Keys {
		params = append(params, sdk.Parameter{
			Name:  "cds.key." + k.Name + ".pub",
			Type:  "string",
			Value: k.Public,
		})
		params = append(params, sdk.Parameter{
			Name:  "cds.key." + k.Name + ".id",
			Type:  "string",
			Value: k.KeyID,
		})
		secrets = append(secrets, sdk.Variable{
			Name:  "cds.key." + k.Name + ".priv",
			Type:  string(k.Type),
			Value: k.Private,
		})
	}

	//Load node definition
	if app != nil {
		for _, k := range app.Keys {
			params = append(params, sdk.Parameter{
				Name:  "cds.key." + k.Name + ".pub",
				Type:  "string",
				Value: k.Public,
			})
			params = append(params, sdk.Parameter{
				Name:  "cds.key." + k.Name + ".id",
				Type:  "string",
				Value: k.KeyID,
			})

			secrets = append(secrets, sdk.Variable{
				Name:  "cds.key." + k.Name + ".priv",
				Type:  string(k.Type),
				Value: k.Private,
			})
		}
	}

	if env != nil && env.ID != sdk.DefaultEnv.ID {
		for _, k := range env.Keys {
			params = append(params, sdk.Parameter{
				Name:  "cds.key." + k.Name + ".pub",
				Type:  "string",
				Value: k.Public,
			})
			params = append(params, sdk.Parameter{
				Name:  "cds.key." + k.Name + ".id",
				Type:  "string",
				Value: k.KeyID,
			})
			secrets = append(secrets, sdk.Variable{
				Name:  "cds.key." + k.Name + ".priv",
				Type:  string(k.Type),
				Value: k.Private,
			})
		}

	}
	return params, secrets, nil
}

// LoadSecrets loads all secrets for a job run
func LoadSecrets(db gorp.SqlExecutor, store cache.Store, nodeRun *sdk.WorkflowNodeRun, w *sdk.WorkflowRun, projV []sdk.ProjectVariable) ([]sdk.Variable, error) {
	var secrets []sdk.Variable

	pv := sdk.VariablesFilter(sdk.FromProjectVariables(projV), sdk.SecretVariable, sdk.KeyVariable)
	pv = sdk.VariablesPrefix(pv, "cds.proj.")
	secrets = append(secrets, pv...)

	var app *sdk.Application
	var env *sdk.Environment
	var pp *sdk.ProjectIntegration

	// Load node definition
	if nodeRun != nil {
		node := w.Workflow.WorkflowData.NodeByID(nodeRun.WorkflowNodeID)
		if node != nil && node.Context != nil {
			if node.Context.ApplicationID != 0 {
				a := w.Workflow.Applications[node.Context.ApplicationID]
				app = &a
			}
			if node.Context.EnvironmentID != 0 {
				e := w.Workflow.Environments[node.Context.EnvironmentID]
				env = &e
			}
			if node.Context.ProjectIntegrationID != 0 {
				p := w.Workflow.ProjectIntegrations[node.Context.ProjectIntegrationID]
				pp = &p
			}
		}

		// Application variables
		av := []sdk.Variable{}
		if app != nil {
			// FIXME manage secret ascode
			// try to retreive application from database, application can be not found for ascode run
			tempApp, err := application.LoadByIDWithClearVCSStrategyPassword(db, app.ID)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return nil, err
			}
			if tempApp != nil {
				app.RepositoryStrategy.Password = tempApp.RepositoryStrategy.Password
			}

			appVariables, err := application.LoadAllVariablesWithDecrytion(db, app.ID)
			if err != nil {
				return nil, sdk.WrapError(err, "LoadSecrets> Cannot load application variables")
			}
			av = sdk.VariablesFilter(sdk.FromAplicationVariables(appVariables), sdk.SecretVariable)
			av = sdk.VariablesPrefix(av, "cds.app.")

			av = append(av, sdk.Variable{
				Name:  "git.http.password",
				Type:  sdk.SecretVariable,
				Value: app.RepositoryStrategy.Password,
			})
		}
		secrets = append(secrets, av...)

		// Environment variables
		ev := []sdk.Variable{}
		if env != nil {
			envv, errE := environment.LoadAllVariablesWithDecrytion(db, env.ID)
			if errE != nil {
				return nil, sdk.WrapError(errE, "LoadSecrets> Cannot load environment variables")
			}
			ev = sdk.VariablesFilter(sdk.FromEnvironmentVariables(envv), sdk.SecretVariable, sdk.KeyVariable)
			ev = sdk.VariablesPrefix(ev, "cds.env.")
		}
		secrets = append(secrets, ev...)

		if pp != nil {
			projectIntegration, err := integration.LoadProjectIntegrationByIDWithClearPassword(db, pp.ID)
			if err != nil {
				return nil, sdk.WrapError(err, "LoadSecrets> Cannot load integration %d", pp.ID)
			}

			// Project integration variable
			pfv := make([]sdk.Variable, 0, len(projectIntegration.Config))
			for k, v := range projectIntegration.Config {
				pfv = append(pfv, sdk.Variable{
					Name:  k,
					Type:  v.Type,
					Value: v.Value,
				})
			}
			pfv = sdk.VariablesPrefix(pfv, "cds.integration.")
			pfv = sdk.VariablesFilter(pfv, sdk.SecretVariable)

			if app != nil && app.DeploymentStrategies != nil {
				strats, err := application.LoadDeploymentStrategies(db, app.ID, true)
				if err != nil {
					return nil, sdk.WrapError(err, "LoadSecrets> Cannot load application deployment strategies %d", app.ID)
				}
				start, has := strats[pp.Name]

				// Application deployment strategies variables
				apv := []sdk.Variable{}
				if has {
					for k, v := range start {
						apv = append(apv, sdk.Variable{
							Name:  k,
							Type:  v.Type,
							Value: v.Value,
						})
					}
				}
				apv = sdk.VariablesPrefix(apv, "cds.integration.")
				apv = sdk.VariablesFilter(apv, sdk.SecretVariable)
				secrets = append(secrets, apv...)
			}
			secrets = append(secrets, pfv...)
		}
	}

	return secrets, nil
}

//BookNodeJobRun  Book a job for a hatchery
func BookNodeJobRun(ctx context.Context, store cache.Store, id int64, hatchery *sdk.Service) (*sdk.Service, error) {
	k := keyBookJob(id)
	h := sdk.Service{}
	find, err := store.Get(k, &h)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", k, err)
	}
	if !find {
		// job not already booked, book it for 2 min
		if err := store.SetWithTTL(k, hatchery, 120); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", k, err)
		}
		return nil, nil
	}
	if h.ID == hatchery.ID {
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrJobAlreadyBooked, "BookNodeJobRun> job %d already booked by %s (%d)", id, h.Name, h.ID)
}

//FreeNodeJobRun  Free a job for a hatchery
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

//AddLog adds a build log
func AddLog(db gorp.SqlExecutor, job *sdk.WorkflowNodeJobRun, logs *sdk.Log, maxLogSize int64) error {
	if job != nil {
		logs.JobID = job.ID
		logs.NodeRunID = job.WorkflowNodeRunID
	}

	// check if log exists without loading data but with log size
	exists, size, err := ExistsStepLog(db, logs.JobID, logs.StepOrder)
	if err != nil {
		return sdk.WrapError(err, "cannot check if log exists")
	}

	// ignore the log if max size already reached
	if maxReached := truncateLogs(maxLogSize, size, logs); maxReached {
		log.Debug("truncated logs")
		return nil
	}

	if !exists {
		return sdk.WrapError(insertLog(db, logs), "cannot insert log")
	}

	return sdk.WrapError(updateLog(db, logs), "cannot update log")
}

//AddServiceLog adds a service log
func AddServiceLog(db gorp.SqlExecutor, job *sdk.WorkflowNodeJobRun, logs *sdk.ServiceLog, maxLogSize int64) error {
	if job != nil {
		logs.WorkflowNodeJobRunID = job.ID
		logs.WorkflowNodeRunID = job.WorkflowNodeRunID
	}

	// check if log exists without loading data but with log size
	exists, size, err := ExistsServiceLog(db, logs.WorkflowNodeJobRunID, logs.ServiceRequirementName)
	if err != nil {
		return sdk.WrapError(err, "cannot check if log exists")
	}

	// ignore the log if max size already reached
	if maxReached := truncateServiceLogs(maxLogSize, size, logs); maxReached {
		return nil
	}

	if !exists {
		return sdk.WrapError(insertServiceLog(db, logs), "Cannot insert log")
	}

	existingLogs, err := LoadServiceLog(db, logs.WorkflowNodeJobRunID, logs.ServiceRequirementName)
	if err != nil {
		return sdk.WrapError(err, "cannot load existing logs")
	}

	logbuf := bytes.NewBufferString(existingLogs.Val)
	logbuf.WriteString(logs.Val)
	existingLogs.Val = logbuf.String()
	existingLogs.LastModified = logs.LastModified

	return sdk.WrapError(updateServiceLog(db, existingLogs), "Cannot update log")
}

// RestartWorkflowNodeJob restart all workflow node job and update logs to indicate restart
func RestartWorkflowNodeJob(ctx context.Context, db gorp.SqlExecutor, wNodeJob sdk.WorkflowNodeJobRun) error {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.RestartWorkflowNodeJob")
	defer end()

	for iS := range wNodeJob.Job.StepStatus {
		step := &wNodeJob.Job.StepStatus[iS]
		if step.Status == sdk.StatusNeverBuilt || step.Status == sdk.StatusSkipped || step.Status == sdk.StatusDisabled {
			continue
		}
		l, errL := LoadStepLogs(db, wNodeJob.ID, int64(step.StepOrder))
		if errL != nil {
			return sdk.WrapError(errL, "RestartWorkflowNodeJob> error while load step logs")
		}
		wNodeJob.Job.Reason = "Killed (Reason: Timeout)\n"
		step.Status = sdk.StatusWaiting
		step.Done = time.Time{}
		if l != nil { // log could be nil here
			l.Done = nil
			logbuf := bytes.NewBufferString(l.Val)
			logbuf.WriteString("\n\n\n-=-=-=-=-=- Worker timeout: job replaced in queue -=-=-=-=-=-\n\n\n")
			l.Val = logbuf.String()
			if err := updateLog(db, l); err != nil {
				return sdk.WrapError(errL, "RestartWorkflowNodeJob> error while update step log")
			}
		}
	}
	nodeRun, errNR := LoadAndLockNodeRunByID(ctx, db, wNodeJob.WorkflowNodeRunID)
	if errNR != nil {
		return errNR
	}

	//Synchronize struct but not in db
	sync, errS := SyncNodeRunRunJob(ctx, db, nodeRun, wNodeJob)
	if errS != nil {
		return sdk.WrapError(errS, "RestartWorkflowNodeJob> error on sync nodeJobRun")
	}
	if !sync {
		log.Warning(ctx, "RestartWorkflowNodeJob> sync doesn't find a nodeJobRun")
	}

	if errU := UpdateNodeRun(db, nodeRun); errU != nil {
		return sdk.WrapError(errU, "RestartWorkflowNodeJob> Cannot update node run")
	}

	if err := replaceWorkflowJobRunInQueue(db, wNodeJob); err != nil {
		return sdk.WrapError(err, "Cannot replace workflow job in queue")
	}

	return nil
}
