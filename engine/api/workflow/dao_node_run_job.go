package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// LoadNodeJobRunQueue load all workflow_node_run_job accessible
func LoadNodeJobRunQueue(db gorp.SqlExecutor, groupsID []int64, since *time.Time, statuses ...string) ([]sdk.WorkflowNodeJobRun, error) {
	if since == nil {
		since = new(time.Time)
	}

	if len(statuses) == 0 {
		statuses = []string{sdk.StatusWaiting.String()}
	}

	query := `select workflow_node_run_job.* 
	from workflow_node_run_job
	join workflow_node_run on workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
	join workflow_node on workflow_node.id = workflow_node_run.workflow_node_id
	join workflow on workflow.id = workflow_node.workflow_id
	join project on project.id = workflow.project_id
	join project_group on project_group.project_id = project.id
	where project_group.group_id = ANY(string_to_array($1, ',')::int[])
	and workflow_node_run_job.queued >= $2
	and workflow_node_run_job.status = ANY(string_to_array($3, ','))`

	var groupID string
	for i, g := range groupsID {
		if i == 0 {
			groupID = fmt.Sprintf("%d", g)
		} else {
			groupID += "," + fmt.Sprintf("%d", g)
		}
	}

	sqlJobs := []JobRun{}
	if _, err := db.Select(&sqlJobs, query, groupID, *since, strings.Join(statuses, ",")); err != nil {
		return nil, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs")
	}

	jobs := make([]sdk.WorkflowNodeJobRun, len(sqlJobs))
	for i := range sqlJobs {
		if err := sqlJobs[i].PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "workflow.LoadNodeJobRun> Unable to load job runs")
		}
		jobs[i] = sdk.WorkflowNodeJobRun(sqlJobs[i])
	}

	return jobs, nil
}

//LoadNodeJobRun load a NodeJobRun given its ID
func LoadNodeJobRun(db gorp.SqlExecutor, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1`
	if err := db.SelectOne(&j, query, id); err != nil {
		return nil, err
	}
	job := sdk.WorkflowNodeJobRun(j)
	return &job, nil
}

//LoadAndLockNodeJobRun load for update a NodeJobRun given its ID
func LoadAndLockNodeJobRun(db gorp.SqlExecutor, id int64) (*sdk.WorkflowNodeJobRun, error) {
	j := JobRun{}
	query := `select workflow_node_run_job.* from workflow_node_run_job where id = $1 for update nowait`
	if err := db.SelectOne(&j, query, id); err != nil {
		return nil, err
	}
	job := sdk.WorkflowNodeJobRun(j)
	return &job, nil
}

func insertWorkflowNodeJobRun(db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	dbj := JobRun(*j)
	if err := db.Insert(&dbj); err != nil {
		return err
	}
	j.ID = dbj.ID
	return nil
}

//DeleteNodeJobRuns deletes all workflow_node_run_job for a given workflow_node_run
func DeleteNodeJobRuns(db gorp.SqlExecutor, nodeID int64) error {
	query := `delete from workflow_node_run_job where workflow_node_run_id = $1`
	_, err := db.Exec(query, nodeID)
	return err
}

//UpdateNodeJobRun updates a workflow_node_run_job
func UpdateNodeJobRun(db gorp.SqlExecutor, j *sdk.WorkflowNodeJobRun) error {
	dbj := JobRun(*j)
	if _, err := db.Update(&dbj); err != nil {
		return err
	}
	return nil
}

func keyBookJob(id int64) string {
	return cache.Key("book", "job", strconv.FormatInt(id, 10))
}

// PostInsert is a db hook on workflow_node_run_job
func (j *JobRun) PostInsert(s gorp.SqlExecutor) error {
	return j.PostUpdate(s)
}

// PostUpdate is a db hook on workflow_node_run_job
func (j *JobRun) PostUpdate(s gorp.SqlExecutor) error {
	jobJSON, err := json.Marshal(j.Job)
	if err != nil {
		return err
	}

	paramsJSON, errP := json.Marshal(j.Parameters)
	if errP != nil {
		return errP
	}

	spawnJSON, errJ := json.Marshal(j.SpawnInfos)
	if errJ != nil {
		return errJ
	}

	query := "update workflow_node_run_job set job = $2, variables = $3, spawninfos = $4 where id = $1"
	if n, err := s.Exec(query, j.ID, jobJSON, paramsJSON, spawnJSON); err != nil {
		return err
	} else if n, _ := n.RowsAffected(); n == 0 {
		return fmt.Errorf("Unable to update workflow_node_run_job id = %d", j.ID)
	}

	return nil
}

// PostGet is a db hook on workflow_node_run_job
func (j *JobRun) PostGet(s gorp.SqlExecutor) error {
	h := sdk.Hatchery{}
	if cache.Get(keyBookJob(j.ID), &h) {
		j.BookedBy = h
	}

	query := "SELECT job, variables, spawninfos FROM workflow_node_run_job WHERE id = $1"
	var params, job, spawn []byte
	if err := s.QueryRow(query, j.ID).Scan(&job, &params, &spawn); err != nil {
		return err
	}

	if err := json.Unmarshal(job, &j.Job); err != nil {
		return err
	}
	if err := json.Unmarshal(params, &j.Parameters); err != nil {
		return err
	}
	if err := json.Unmarshal(spawn, &j.SpawnInfos); err != nil {
		return err
	}

	j.QueuedSeconds = time.Now().Unix() - j.Queued.Unix()

	return nil
}
