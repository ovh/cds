package workflow

import (
	"database/sql"
	"time"

	"github.com/lib/pq"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ExistsStepLog returns the size of step log if exists.
func ExistsStepLog(db gorp.SqlExecutor, id int64, order int64) (bool, int64, error) {
	query := `
    SELECT octet_length(value) as size
    FROM workflow_node_run_job_logs
    WHERE workflow_node_run_job_id = $1 AND step_order = $2
  `

	var size int64
	if err := db.QueryRow(query, id, order).Scan(&size); err != nil {
		if sdk.Cause(err) != sql.ErrNoRows {
			return false, 0, sdk.WithStack(err)
		}
		return false, 0, nil
	}

	return true, size, nil
}

//LoadStepLogs load logs (workflow_node_run_job_logs) for a job (workflow_node_run_job) for a specific step_order
func LoadStepLogs(db gorp.SqlExecutor, id int64, order int64) (*sdk.Log, error) {
	log.Debug("LoadStepLogs> workflow_node_run_job_id = %d", id)
	query := `
		SELECT id, workflow_node_run_job_id, workflow_node_run_id, start, last_modified, done, step_order, value
		FROM workflow_node_run_job_logs
		WHERE workflow_node_run_job_id = $1 AND step_order = $2`
	logs := &sdk.Log{}
	var s, m, d pq.NullTime
	if err := db.QueryRow(query, id, order).Scan(&logs.ID, &logs.JobID, &logs.NodeRunID, &s, &m, &d, &logs.StepOrder, &logs.Val); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if s.Valid {
		logs.Start = &s.Time
	}
	if m.Valid {
		logs.LastModified = &m.Time
	}
	if d.Valid {
		logs.Done = &d.Time
	}
	return logs, nil
}

//LoadLogs load logs (workflow_node_run_job_logs) for a job (workflow_node_run_job)
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
		var s, m, d pq.NullTime

		if err := rows.Scan(&l.ID, &l.JobID, &l.NodeRunID, &s, &m, &d, &l.StepOrder, &l.Val); err != nil {
			return nil, err
		}

		if s.Valid {
			l.Start = &s.Time
		}
		if m.Valid {
			l.LastModified = &m.Time
		}
		if d.Valid {
			l.Done = &d.Time
		}

		logs = append(logs, *l)
	}
	return logs, nil
}

func insertLog(db gorp.SqlExecutor, logs *sdk.Log) error {
	query := `
		INSERT INTO workflow_node_run_job_logs (workflow_node_run_job_id, workflow_node_run_id, start, last_modified, done, step_order, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ID `
	return sdk.WithStack(db.QueryRow(query, logs.JobID, logs.NodeRunID, logs.Start, logs.LastModified, logs.Done, logs.StepOrder, logs.Val).Scan(&logs.ID))
}

func updateLog(db gorp.SqlExecutor, logs *sdk.Log) error {
	now := time.Now()
	if logs.Start == nil {
		logs.Start = &now
	}
	if logs.LastModified == nil {
		logs.LastModified = &now
	}
	if logs.Done == nil {
		logs.Done = &now
	}

	query := `
		UPDATE workflow_node_run_job_logs set
			workflow_node_run_job_id = $1,
			workflow_node_run_id = $2,
			start = $3,
			last_modified = $4,
			done = $5,
			step_order = $6,
			value = $7
		where id = $8`

	if _, err := db.Exec(query, logs.JobID, logs.NodeRunID, logs.Start, logs.LastModified, logs.Done, logs.StepOrder, logs.Val, logs.ID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
