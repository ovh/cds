package workflow

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// UpdateNodeJobRunStatus Update status of an workflow_node_run_job
func UpdateNodeJobRunStatus(db gorp.SqlExecutor, job *sdk.WorkflowNodeJobRun, status sdk.Status) error {
	var query string
	query = `SELECT status FROM workflow_node_run_job WHERE id = $1 FOR UPDATE`
	var currentStatus string
	if err := db.QueryRow(query, job.ID).Scan(&currentStatus); err != nil {
		return sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Cannot lock node job run %d: %s", job.ID, err)
	}

	switch status {
	case sdk.StatusBuilding:
		if currentStatus != sdk.StatusWaiting.String() {
			return fmt.Errorf("workflow.UpdateNodeJobRunStatus> Cannot update status of WorkflowNodeJobRun %d to %s, expected current status %s, got %s",
				job.ID, status, sdk.StatusWaiting, currentStatus)
		}
		job.Start = time.Now()
		job.Status = status.String()

	case sdk.StatusFail, sdk.StatusSuccess, sdk.StatusDisabled, sdk.StatusSkipped:
		if currentStatus != string(sdk.StatusWaiting) && currentStatus != string(sdk.StatusBuilding) && status != sdk.StatusDisabled && status != sdk.StatusSkipped {
			log.Debug("workflow.UpdateNodeJobRunStatus> Status is %s, cannot update %d to %s", currentStatus, job.ID, status)
			// too late, Nate
			return nil
		}
		job.Done = time.Now()
		job.Status = status.String()
	default:
		return fmt.Errorf("workflow.UpdateNodeJobRunStatus> Cannot update WorkflowNodeJobRun %d to status %v", job.ID, status.String())
	}

	if err := UpdateNodeJobRun(db, job); err != nil {
		return sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Cannot update WorkflowNodeJobRun %d", job.ID)
	}

	node, errLoad := LoadNodeRunByID(db, job.WorkflowNodeRunID)
	if errLoad != nil {
		return errLoad
	}

	event.PublishJobRun(node, job)
	//call workflow.execute
	cache.Enqueue(queueWorkflowNodeRun, node)

	return nil
}

// AddSpawnInfosNodeJobRun saves spawn info before starting worker
func AddSpawnInfosNodeJobRun(db gorp.SqlExecutor, id int64, infos []sdk.SpawnInfo) (*sdk.WorkflowNodeJobRun, error) {
	j, err := LoadAndLockNodeJobRun(db, id)
	if err != nil {
		return nil, sdk.WrapError(err, "AddSpawnInfosNodeJobRun> Cannot load node job run")
	}
	if err := prepareSpawnInfos(j, infos); err != nil {
		return nil, sdk.WrapError(err, "AddSpawnInfosNodeJobRun> Cannot prepare spawn infos")
	}
	if err := UpdateNodeJobRun(db, j); err != nil {
		return nil, sdk.WrapError(err, "AddSpawnInfosNodeJobRun> Cannot update node job run")
	}
	return j, nil
}

func prepareSpawnInfos(j *sdk.WorkflowNodeJobRun, infos []sdk.SpawnInfo) error {
	now := time.Now()
	for _, info := range infos {
		j.SpawnInfos = append(j.SpawnInfos, sdk.SpawnInfo{
			APITime:    now,
			RemoteTime: info.RemoteTime,
			Message:    info.Message,
		})
	}
	return nil
}

// TakeNodeJobRun Take an a job run for update
func TakeNodeJobRun(db gorp.SqlExecutor, id int64, workerModel string, workerName string, infos []sdk.SpawnInfo) (*sdk.WorkflowNodeJobRun, error) {
	job, err := LoadAndLockNodeJobRun(db, id)
	if err != nil {
		return nil, sdk.WrapError(err, "TakeNodeJobRun> Cannot load node job run")
	}
	if job.Status != sdk.StatusWaiting.String() {
		k := keyBookJob(id)
		h := sdk.Hatchery{}
		if cache.Get(k, &h) {
			return nil, sdk.WrapError(sdk.ErrAlreadyTaken, "TakeNodeJobRun> job %d is not waiting status and was booked by hatchery %d. Current status:%s", id, h.ID, job.Status)
		}
		return nil, sdk.WrapError(sdk.ErrAlreadyTaken, "TakeNodeJobRun> job %d is not waiting status. Current status:%s", id, job.Status)
	}

	job.Model = workerModel
	job.Job.WorkerName = workerName
	job.Start = time.Now()
	job.Status = sdk.StatusBuilding.String()

	if err := prepareSpawnInfos(job, infos); err != nil {
		return nil, sdk.WrapError(err, "TakeNodeJobRun> Cannot prepare spawn infos")
	}

	if err := UpdateNodeJobRun(db, job); err != nil {
		return nil, sdk.WrapError(err, "TakeNodeJobRun>Cannot update node job run")
	}

	return job, nil
}

// LoadNodeJobRunSecrets loads all secrets for a job run
func LoadNodeJobRunSecrets(db gorp.SqlExecutor, job *sdk.WorkflowNodeJobRun) ([]sdk.Variable, error) {
	//Load workflow node run
	node, err := LoadNodeRunByID(db, job.WorkflowNodeRunID)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNodeJobRunSecrets> Unable to load node run")
	}

	//Load workflow run
	w, err := loadRunByID(db, node.WorkflowRunID)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadNodeJobRunSecrets> Unable to load workflow run")
	}

	var secrets []sdk.Variable

	// Load project secrets
	pv, err := project.GetAllVariableInProject(db, w.Workflow.ProjectID, project.WithClearPassword())
	if err != nil {
		return nil, err
	}
	pv = sdk.VariablesFilter(pv, sdk.SecretVariable, sdk.KeyVariable)
	pv = sdk.VariablesPrefix(pv, "cds.proj")
	secrets = append(secrets, pv...)

	//Load node definition
	n := w.Workflow.GetNode(node.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", node.WorkflowNodeID), "LoadNodeJobRunSecrets>")
	}

	//Application variables
	av := []sdk.Variable{}
	if n.Context != nil && n.Context.Application != nil {
		av = sdk.VariablesFilter(n.Context.Application.Variable, sdk.SecretVariable, sdk.KeyVariable)
		av = sdk.VariablesPrefix(pv, "cds.app")
	}
	secrets = append(secrets, av...)

	//Environment variables
	ev := []sdk.Variable{}
	if n.Context != nil && n.Context.Environment != nil {
		ev = sdk.VariablesFilter(n.Context.Environment.Variable, sdk.SecretVariable, sdk.KeyVariable)
		ev = sdk.VariablesPrefix(pv, "cds.env")
	}
	secrets = append(secrets, ev...)

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
func BookNodeJobRun(id int64, hatchery *sdk.Hatchery) (*sdk.Hatchery, error) {
	k := keyBookJob(id)
	h := sdk.Hatchery{}
	if !cache.Get(k, &h) {
		// job not already booked, book it for 2 min
		cache.SetWithTTL(k, hatchery, 120)
		return nil, nil
	}
	return &h, sdk.WrapError(sdk.ErrJobAlreadyBooked, "BookNodeJobRun> job %d already booked by %s (%d)", id, h.Name, h.ID)
}

//AddLog adds a build log
func AddLog(db gorp.SqlExecutor, job *sdk.WorkflowNodeJobRun, logs *sdk.Log) error {
	logs.PipelineBuildJobID = job.ID
	logs.PipelineBuildID = job.WorkflowNodeRunID

	existingLogs, errLog := LoadStepLogs(db, logs.PipelineBuildJobID, logs.StepOrder)
	if errLog != nil && errLog != sql.ErrNoRows {
		return sdk.WrapError(errLog, "AddLog> Cannot load existing logs")
	}

	if existingLogs == nil {
		if err := InsertLog(db, logs); err != nil {
			return sdk.WrapError(err, "AddLog> Cannot insert log")
		}
	} else {
		existingLogs.Val += logs.Val
		existingLogs.LastModified = logs.LastModified
		existingLogs.Done = logs.Done
		if err := UpdateLog(db, existingLogs); err != nil {
			return sdk.WrapError(err, "AddLog> Cannot update log")
		}
	}
	return nil
}
