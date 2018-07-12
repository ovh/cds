package workflow

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/engine/api/tracing"
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

// Add something to the report
func (r *ProcessorReport) Add(i ...interface{}) {
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
			r.nodes = append(r.nodes, x)
		case *sdk.WorkflowNodeRun:
			r.nodes = append(r.nodes, *x)
		case sdk.WorkflowRun:
			r.workflows = append(r.workflows, x)
		case *sdk.WorkflowRun:
			r.workflows = append(r.workflows, *x)
		default:
			log.Warning("ProcessorReport> unknown type %T", w)
		}
	}
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
func (r *ProcessorReport) Merge(r1 *ProcessorReport, err error) (*ProcessorReport, error) {
	if r == nil {
		return r1, err
	}
	if r1 == nil {
		return r, err
	}
	data := r1.All()
	r.Add(data...)
	return r, err
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

// UpdateNodeJobRunStatus Update status of an workflow_node_run_job
// the dbFunc parameter is only used to send status to the repository manager
func UpdateNodeJobRunStatus(ctx context.Context, dbFunc func() *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, job *sdk.WorkflowNodeJobRun, status sdk.Status) (*ProcessorReport, error) {
	var end func()
	ctx, end = tracing.Span(ctx, "workflow.UpdateNodeJobRunStatus",
		tracing.Tag("workflow_node_run_job", job.ID),
		tracing.Tag("workflow_node_run_job_status", status),
	)
	defer end()

	report := new(ProcessorReport)

	log.Debug("UpdateNodeJobRunStatus> job.ID=%d status=%s", job.ID, status.String())

	node, errLoad := LoadNodeRunByID(db, job.WorkflowNodeRunID, LoadRunOptions{})
	if errLoad != nil {
		return nil, sdk.WrapError(errLoad, "workflow.UpdateNodeJobRunStatus> Unable to load node run id %d", job.WorkflowNodeRunID)
	}

	query := `SELECT status FROM workflow_node_run_job WHERE id = $1`
	var currentStatus string
	if err := db.QueryRow(query, job.ID).Scan(&currentStatus); err != nil {
		return nil, sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Cannot select status from workflow_node_run_job node job run %d", job.ID)
	}

	switch status {
	case sdk.StatusBuilding:
		if currentStatus != sdk.StatusWaiting.String() {
			return nil, fmt.Errorf("workflow.UpdateNodeJobRunStatus> Cannot update status of WorkflowNodeJobRun %d to %s, expected current status %s, got %s",
				job.ID, status, sdk.StatusWaiting, currentStatus)
		}
		job.Start = time.Now()
		job.Status = status.String()

	case sdk.StatusFail, sdk.StatusSuccess, sdk.StatusDisabled, sdk.StatusSkipped, sdk.StatusStopped:
		if currentStatus != string(sdk.StatusWaiting) && currentStatus != string(sdk.StatusBuilding) && status != sdk.StatusDisabled && status != sdk.StatusSkipped {
			log.Debug("workflow.UpdateNodeJobRunStatus> Status is %s, cannot update %d to %s", currentStatus, job.ID, status)
			// too late, Nate
			return nil, nil
		}
		job.Done = time.Now()
		job.Status = status.String()

		wf, errLoadWf := LoadRunByID(db, node.WorkflowRunID, LoadRunOptions{})
		if errLoadWf != nil {
			return nil, sdk.WrapError(errLoadWf, "workflow.UpdateNodeJobRunStatus> Unable to load run id %d", node.WorkflowRunID)
		}

		wf.LastExecution = time.Now()
		if err := UpdateWorkflowRun(ctx, db, wf); err != nil {
			return nil, sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Cannot update WorkflowRun %d", wf.ID)
		}
	default:
		return nil, fmt.Errorf("workflow.UpdateNodeJobRunStatus> Cannot update WorkflowNodeJobRun %d to status %v", job.ID, status.String())
	}

	//If the job has been set to building, set the stage to building
	var stageIndex int
	for i := range node.Stages {
		s := &node.Stages[i]
		for _, j := range s.Jobs {
			if j.Action.ID == job.Job.Job.Action.ID {
				stageIndex = i
			}
		}
	}

	if err := UpdateNodeJobRun(ctx, db, store, job); err != nil {
		return nil, sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Cannot update WorkflowNodeJobRun %d", job.ID)
	}

	report.Add(*job)

	if status == sdk.StatusBuilding {
		// Sync job status in noderun
		nodeRun, errNR := LoadNodeRunByID(db, node.ID, LoadRunOptions{})
		if errNR != nil {
			return nil, sdk.WrapError(errNR, "workflow.UpdateNodeJobRunStatus> Cannot LoadNodeRunByID node run %d", node.ID)
		}
		return report.Merge(syncTakeJobInNodeRun(ctx, db, nodeRun, job, stageIndex))
	}

	var errReport error
	report, errReport = report.Merge(execute(ctx, db, store, proj, node))

	wr, err := LoadRunByID(db, node.WorkflowRunID, LoadRunOptions{DisableDetailledNodeRun: true, WithTests: true})
	if err != nil {
		return report, sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Cannot load run by ID %d", node.WorkflowRunID)
	}

	//Start a goroutine to update commit statuses in repositories manager
	go func(wfRun *sdk.WorkflowRun) {
		if sdk.StatusIsTerminated(wfRun.Status) {
			wr.LastExecution = time.Now()
			if err := ResyncCommitStatus(context.Background(), dbFunc(), store, proj, wfRun); err != nil {
				log.Error("workflow.UpdateNodeJobRunStatus> %v", err)
			}
		}
	}(wr)

	return report, errReport
}

// AddSpawnInfosNodeJobRun saves spawn info before starting worker
func AddSpawnInfosNodeJobRun(db gorp.SqlExecutor, jobID int64, infos []sdk.SpawnInfo) error {
	wnjri := &sdk.WorkflowNodeJobRunInfo{
		WorkflowNodeJobRunID: jobID,
		SpawnInfos:           PrepareSpawnInfos(infos),
	}
	if err := insertNodeRunJobInfo(db, wnjri); err != nil {
		return sdk.WrapError(err, "AddSpawnInfosNodeJobRun> Cannot update node job run infos %d", jobID)
	}
	return nil
}

// PrepareSpawnInfos helps yoi to create sdk.SpawnInfo array
func PrepareSpawnInfos(infos []sdk.SpawnInfo) []sdk.SpawnInfo {
	now := time.Now()
	prepared := []sdk.SpawnInfo{}
	for _, info := range infos {
		prepared = append(prepared, sdk.SpawnInfo{
			APITime:    now,
			RemoteTime: info.RemoteTime,
			Message:    info.Message,
		})
	}
	return prepared
}

// TakeNodeJobRun Take an a job run for update
func TakeNodeJobRun(ctx context.Context, dbFunc func() *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, jobID int64, workerModel string, workerName string, workerID string, infos []sdk.SpawnInfo) (*sdk.WorkflowNodeJobRun, *ProcessorReport, error) {
	var end func()
	ctx, end = tracing.Span(ctx, "workflow.TakeNodeJobRun")
	defer end()

	report := new(ProcessorReport)

	// first load without FOR UPDATE WAIT to quick check status
	currentStatus, errS := db.SelectStr(`SELECT status FROM workflow_node_run_job WHERE id = $1`, jobID)
	if errS != nil {
		return nil, nil, sdk.WrapError(errS, "TakeNodeJobRun> Cannot select status from workflow_node_run_job node job run %d", jobID)
	}

	if err := checkStatusWaiting(store, jobID, currentStatus); err != nil {
		return nil, nil, err
	}

	// reload and recheck status
	job, errl := LoadAndLockNodeJobRunNoWait(ctx, db, store, jobID)
	if errl != nil {
		if errPG, ok := errl.(*pq.Error); ok && errPG.Code == "55P03" {
			errl = sdk.ErrJobAlreadyBooked
		}
		return nil, nil, sdk.WrapError(errl, "TakeNodeJobRun> Cannot load node job run (WAIT) %d", jobID)
	}
	if err := checkStatusWaiting(store, jobID, job.Status); err != nil {
		return nil, report, err
	}

	job.Model = workerModel
	job.Job.WorkerName = workerName
	job.Job.WorkerID = workerID
	job.Start = time.Now()

	_, errExec := db.Exec("UPDATE workflow_node_run_job SET worker_id = $2 WHERE id = $1", job.ID, workerID)
	if errExec != nil {
		return nil, nil, sdk.WrapError(errl, "TakeNodeJobRun> Cannot update worker_id in node job run %d", jobID)
	}

	if err := AddSpawnInfosNodeJobRun(db, jobID, PrepareSpawnInfos(infos)); err != nil {
		return nil, nil, sdk.WrapError(err, "TakeNodeJobRun> Cannot save spawn info on node job run %d", jobID)
	}

	var err error
	report, err = report.Merge(UpdateNodeJobRunStatus(ctx, dbFunc, db, store, p, job, sdk.StatusBuilding))
	if err != nil {
		log.Debug("TakeNodeJobRun> call UpdateNodeJobRunStatus on job %d set status from %s to %s", job.ID, job.Status, sdk.StatusBuilding)
		return nil, nil, sdk.WrapError(err, "TakeNodeJobRun>Cannot update node job run %d", jobID)
	}

	return job, report, nil
}

func checkStatusWaiting(store cache.Store, jobID int64, status string) error {
	if status != sdk.StatusWaiting.String() {
		k := keyBookJob(jobID)
		h := sdk.Hatchery{}
		if store.Get(k, &h) {
			return sdk.WrapError(sdk.ErrAlreadyTaken, "checkStatusWaiting> job %d is not waiting status and was booked by hatchery %d. Current status:%s", jobID, h.ID, status)
		}
		return sdk.WrapError(sdk.ErrAlreadyTaken, "checkStatusWaiting> job %d is not waiting status. Current status:%s", jobID, status)
	}
	return nil
}

// LoadNodeJobRunKeys loads all keys for a job run
func LoadNodeJobRunKeys(db gorp.SqlExecutor, store cache.Store, job *sdk.WorkflowNodeJobRun, nodeRun *sdk.WorkflowNodeRun, w *sdk.WorkflowRun, p *sdk.Project) ([]sdk.Parameter, []sdk.Variable, error) {
	params := []sdk.Parameter{}
	secrets := []sdk.Variable{}

	for _, k := range p.Keys {
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
			Type:  "string",
			Value: k.Private,
		})
	}

	//Load node definition
	n := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
	if n == nil {
		return nil, nil, sdk.WrapError(fmt.Errorf("LoadNodeJobRunKeys> Unable to find node %d in workflow", nodeRun.WorkflowNodeID), "LoadNodeJobRunSecrets>")
	}
	if n.Context != nil && n.Context.Application != nil {
		for _, k := range n.Context.Application.Keys {
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

			unBase64, err64 := base64.StdEncoding.DecodeString(k.Private)
			if err64 != nil {
				return nil, nil, sdk.WrapError(err64, "LoadNodeJobRunKeys> Cannot app decode key %s", k.Name)
			}
			decrypted, errD := secret.Decrypt([]byte(unBase64))
			if errD != nil {
				log.Error("LoadNodeJobRunKeys> Unable to decrypt app private key %s/%s: %v", n.Context.Application.Name, k.Name, errD)
			}
			secrets = append(secrets, sdk.Variable{
				Name:  "cds.key." + k.Name + ".priv",
				Type:  "string",
				Value: string(decrypted),
			})
		}
	}

	if n.Context != nil && n.Context.Environment != nil && n.Context.Environment.ID != sdk.DefaultEnv.ID {
		for _, k := range n.Context.Environment.Keys {
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

			unBase64, err64 := base64.StdEncoding.DecodeString(k.Private)
			if err64 != nil {
				return nil, nil, sdk.WrapError(err64, "LoadNodeJobRunKeys> Cannot decode env key %s", k.Name)
			}
			decrypted, errD := secret.Decrypt([]byte(unBase64))
			if errD != nil {
				log.Error("LoadNodeJobRunKeys> Unable to decrypt env private key %s/%s: %v", n.Context.Environment.Name, k.Name, errD)
			}
			secrets = append(secrets, sdk.Variable{
				Name:  "cds.key." + k.Name + ".priv",
				Type:  "string",
				Value: string(decrypted),
			})
		}

	}
	return params, secrets, nil
}

// LoadNodeJobRunSecrets loads all secrets for a job run
func LoadNodeJobRunSecrets(db gorp.SqlExecutor, store cache.Store, job *sdk.WorkflowNodeJobRun, nodeRun *sdk.WorkflowNodeRun, w *sdk.WorkflowRun, pv []sdk.Variable) ([]sdk.Variable, error) {
	var secrets []sdk.Variable

	pv = sdk.VariablesFilter(pv, sdk.SecretVariable, sdk.KeyVariable)
	pv = sdk.VariablesPrefix(pv, "cds.proj.")
	secrets = append(secrets, pv...)

	//Load node definition
	n := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", nodeRun.WorkflowNodeID), "LoadNodeJobRunSecrets>")
	}

	//Application variables
	av := []sdk.Variable{}
	if n.Context != nil && n.Context.Application != nil {
		appv, errA := application.GetAllVariableByID(db, n.Context.Application.ID, application.WithClearPassword())
		if errA != nil {
			return nil, sdk.WrapError(errA, "LoadNodeJobRunSecrets> Cannot load application variables")
		}
		av = sdk.VariablesFilter(appv, sdk.SecretVariable, sdk.KeyVariable)
		av = sdk.VariablesPrefix(av, "cds.app.")

		if err := application.DecryptVCSStrategyPassword(n.Context.Application); err != nil {
			return nil, sdk.WrapError(err, "LoadNodeJobRunSecrets> Cannot decrypt vcs configuration")
		}
		av = append(av, sdk.Variable{
			Name:  "git.http.password",
			Type:  sdk.SecretVariable,
			Value: n.Context.Application.RepositoryStrategy.Password,
		})
	}
	secrets = append(secrets, av...)

	//Environment variables
	ev := []sdk.Variable{}
	if n.Context != nil && n.Context.Environment != nil {
		envv, errE := environment.GetAllVariableByID(db, n.Context.Environment.ID, environment.WithClearPassword())
		if errE != nil {
			return nil, sdk.WrapError(errE, "LoadNodeJobRunSecrets> Cannot load environment variables")
		}
		ev = sdk.VariablesFilter(envv, sdk.SecretVariable, sdk.KeyVariable)
		ev = sdk.VariablesPrefix(ev, "cds.env.")
	}
	secrets = append(secrets, ev...)

	if n.Context.ProjectPlatform != nil {
		pf, err := platform.LoadByID(db, n.Context.ProjectPlatformID, true)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadNodeJobRunSecrets> Cannot load platform %d", n.Context.ProjectPlatformID)
		}

		//Projeft platform variable
		pfv := make([]sdk.Variable, 0, len(pf.Config))
		for k, v := range pf.Config {
			pfv = append(pfv, sdk.Variable{
				Name:  k,
				Type:  v.Type,
				Value: v.Value,
			})
		}
		pfv = sdk.VariablesPrefix(pfv, "cds.platform.")
		pfv = sdk.VariablesFilter(pfv, sdk.SecretVariable)

		if n.Context.Application != nil && n.Context.Application.DeploymentStrategies != nil {
			strats, err := application.LoadDeploymentStrategies(db, n.Context.ApplicationID, true)
			if err != nil {
				return nil, sdk.WrapError(err, "LoadNodeJobRunSecrets> Cannot load application deployment strategies %d", n.Context.ApplicationID)
			}
			strat, has := strats[n.Context.ProjectPlatform.Name]

			//Application deployment strategies variables
			apv := []sdk.Variable{}
			if has {
				for k, v := range strat {
					apv = append(apv, sdk.Variable{
						Name:  k,
						Type:  v.Type,
						Value: v.Value,
					})
				}
			}
			apv = sdk.VariablesPrefix(apv, "cds.platform.")
			apv = sdk.VariablesFilter(apv, sdk.SecretVariable)
			secrets = append(secrets, apv...)
		}
		secrets = append(secrets, pfv...)
	}

	//Decrypt secrets
	for i := range secrets {
		s := &secrets[i]
		if err := secret.DecryptVariable(s); err != nil {
			return nil, sdk.WrapError(err, "LoadNodeJobRunSecrets> Unable to decrypt variables")
		}
	}
	return secrets, nil
}

//BookNodeJobRun  Book a job for a hatchery
func BookNodeJobRun(store cache.Store, id int64, hatchery *sdk.Hatchery) (*sdk.Hatchery, error) {
	k := keyBookJob(id)
	h := sdk.Hatchery{}
	if !store.Get(k, &h) {
		// job not already booked, book it for 2 min
		store.SetWithTTL(k, hatchery, 120)
		return nil, nil
	}
	if h.ID == hatchery.ID {
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrJobAlreadyBooked, "BookNodeJobRun> job %d already booked by %s (%d)", id, h.Name, h.ID)
}

//FreeNodeJobRun  Free a job for a hatchery
func FreeNodeJobRun(store cache.Store, id int64) error {
	k := keyBookJob(id)
	h := sdk.Hatchery{}
	if store.Get(k, &h) {
		// job not already booked, book it for 2 min
		store.Delete(k)
		return nil
	}
	return sdk.WrapError(sdk.ErrJobNotBooked, "BookNodeJobRun> job %d already released", id)
}

//AddNodeJobAttempt add an hatchery attempt to spawn a job
func AddNodeJobAttempt(db gorp.SqlExecutor, id, hatcheryID int64) ([]int64, error) {
	var ids []int64
	query := "UPDATE workflow_node_run_job SET spawn_attempts = array_append(spawn_attempts, $1) WHERE id = $2"
	if _, err := db.Exec(query, hatcheryID, id); err != nil && err != sql.ErrNoRows {
		return ids, sdk.WrapError(err, "AddNodeJobAttempt> cannot update node run job")
	}

	rows, err := db.Query("SELECT DISTINCT unnest(spawn_attempts) FROM workflow_node_run_job WHERE id = $1", id)
	var hID int64
	defer rows.Close()
	for rows.Next() {
		if errS := rows.Scan(&hID); errS != nil {
			return ids, sdk.WrapError(errS, "AddNodeJobAttempt> cannot scan")
		}
		ids = append(ids, hID)
	}

	return ids, err
}

//AddLog adds a build log
func AddLog(db gorp.SqlExecutor, job *sdk.WorkflowNodeJobRun, logs *sdk.Log) error {
	if job != nil {
		logs.PipelineBuildJobID = job.ID
		logs.PipelineBuildID = job.WorkflowNodeRunID
	}

	existingLogs, errLog := LoadStepLogs(db, logs.PipelineBuildJobID, logs.StepOrder)
	if errLog != nil && errLog != sql.ErrNoRows {
		return sdk.WrapError(errLog, "AddLog> Cannot load existing logs")
	}

	if existingLogs == nil {
		if err := insertLog(db, logs); err != nil {
			return sdk.WrapError(err, "AddLog> Cannot insert log")
		}
	} else {
		logbuf := bytes.NewBufferString(existingLogs.Val)
		logbuf.WriteString(logs.Val)
		existingLogs.Val = logbuf.String()
		existingLogs.LastModified = logs.LastModified
		existingLogs.Done = logs.Done
		if err := updateLog(db, existingLogs); err != nil {
			return sdk.WrapError(err, "AddLog> Cannot update log")
		}
	}
	return nil
}

//AddServiceLog adds a service log
func AddServiceLog(db gorp.SqlExecutor, job *sdk.WorkflowNodeJobRun, logs *sdk.ServiceLog) error {
	if job != nil {
		logs.WorkflowNodeJobRunID = job.ID
		logs.WorkflowNodeRunID = job.WorkflowNodeRunID
	}

	existingLogs, errLog := LoadServiceLog(db, logs.WorkflowNodeJobRunID, logs.ServiceRequirementName)
	if errLog != nil && errLog != sql.ErrNoRows {
		return sdk.WrapError(errLog, "AddServiceLog> Cannot load existing logs")
	}

	if existingLogs == nil {
		if err := insertServiceLog(db, logs); err != nil {
			return sdk.WrapError(err, "AddServiceLog> Cannot insert log")
		}
	} else {
		logbuf := bytes.NewBufferString(existingLogs.Val)
		logbuf.WriteString(logs.Val)
		existingLogs.Val = logbuf.String()
		existingLogs.LastModified = logs.LastModified
		if err := updateServiceLog(db, existingLogs); err != nil {
			return sdk.WrapError(err, "AddServiceLog> Cannot update log")
		}
	}
	return nil
}

// RestartWorkflowNodeJob restart all workflow node job and update logs to indicate restart
func RestartWorkflowNodeJob(ctx context.Context, db gorp.SqlExecutor, wNodeJob sdk.WorkflowNodeJobRun) error {
	var end func()
	ctx, end = tracing.Span(ctx, "workflow.RestartWorkflowNodeJob")
	defer end()

	for iS := range wNodeJob.Job.StepStatus {
		step := &wNodeJob.Job.StepStatus[iS]
		if step.Status == sdk.StatusNeverBuilt.String() || step.Status == sdk.StatusSkipped.String() || step.Status == sdk.StatusDisabled.String() {
			continue
		}
		l, errL := LoadStepLogs(db, wNodeJob.ID, int64(step.StepOrder))
		if errL != nil {
			return sdk.WrapError(errL, "RestartWorkflowNodeJob> error while load step logs")
		}
		wNodeJob.Job.Reason = "Killed (Reason: Timeout)\n"
		step.Status = sdk.StatusWaiting.String()
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

	nodeRun, errNR := LoadAndLockNodeRunByID(ctx, db, wNodeJob.WorkflowNodeRunID, true)
	if errNR != nil {
		return sdk.WrapError(errNR, "RestartWorkflowNodeJob> Cannot load node run")
	}

	//Synchronise struct but not in db
	sync, errS := SyncNodeRunRunJob(ctx, db, nodeRun, wNodeJob)
	if errS != nil {
		return sdk.WrapError(errS, "RestartWorkflowNodeJob> error on sync nodeJobRun")
	}
	if !sync {
		log.Warning("RestartWorkflowNodeJob> sync doesn't find a nodeJobRun")
	}

	if errU := UpdateNodeRun(db, nodeRun); errU != nil {
		return sdk.WrapError(errU, "RestartWorkflowNodeJob> Cannot update node run")
	}

	if err := replaceWorkflowJobRunInQueue(db, wNodeJob); err != nil {
		return sdk.WrapError(err, "RestartWorkflowNodeJob> Cannot replace workflow job in queue")
	}

	return nil
}
