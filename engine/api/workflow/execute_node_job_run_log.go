package workflow

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

func LoadStepLogs(db gorp.SqlExecutor, id int64, order int64) (*sdk.Log, error) {
	query := `
		SELECT id, workflow_node_run_job_id, workflow_node_run_id, start, last_modified, done, step_order, value
		FROM workflow_node_run_job_logs
		WHERE workflow_node_run_job_id = $1 AND step_order = $2`
	logs := &sdk.Log{}
	if err := db.QueryRow(query, id, order).Scan(&logs.Id, &logs.PipelineBuildJobID, &logs.PipelineBuildID, &logs.Start, &logs.LastModified, &logs.Done, &logs.StepOrder, &logs.Val); err != nil {
		return nil, err
	}
	return logs, nil
}

func LoadLogs(db gorp.SqlExecutor, id int64) ([]sdk.Log, error) {
	query := `
		SELECT id, workflow_node_run_job_id, workflow_node_run_id, start, last_modified, done, step_order, value
		FROM workflow_node_run_job_logs
		WHERE workflow_node_run_job_id = $1
		ORDER BY id`
	rows, err := db.Query(query, id)
	if err != nil {
		return nil, err
	}
	var logs []sdk.Log
	for rows.Next() {
		l := &sdk.Log{}
		if err := rows.Scan(&l.Id, &l.PipelineBuildJobID, &l.PipelineBuildID, &l.Start, &l.LastModified, &l.Done, &l.StepOrder, &l.Val); err != nil {
			return nil, err
		}
		logs = append(logs, *l)
	}
	return logs, nil
}

func InsertLog(db gorp.SqlExecutor, logs *sdk.Log) error {

	query := `
		INSERT INTO workflow_node_run_job_logs (workflow_node_run_job_id, workflow_node_run_id, start, last_modified, done, step_order, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ID 
	`

	if err := db.QueryRow(query, logs.PipelineBuildJobID, logs.PipelineBuildID, logs.Start, logs.LastModified, logs.Done, logs.StepOrder, logs.Val).Scan(logs.Id); err != nil {

	}

	return nil
}

func UpdateLog(db gorp.SqlExecutor, logs *sdk.Log) error {
	return nil
}
