package workflow

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/protobuf/ptypes"
	"github.com/ovh/cds/sdk"
)

//LoadStepLogs load logs (workflow_node_run_job_logs) for a job (workflow_node_run_job) for a specific step_order
func LoadStepLogs(db gorp.SqlExecutor, id int64, order int64) (*sdk.Log, error) {
	query := `
		SELECT id, workflow_node_run_job_id, workflow_node_run_id, start, last_modified, done, step_order, value
		FROM workflow_node_run_job_logs
		WHERE workflow_node_run_job_id = $1 AND step_order = $2`
	logs := &sdk.Log{}
	var s, m, d time.Time
	if err := db.QueryRow(query, id, order).Scan(&logs.Id, &logs.PipelineBuildJobID, &logs.PipelineBuildID, &s, &m, &d, &logs.StepOrder, &logs.Val); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var err error
	logs.Start, err = ptypes.TimestampProto(s)
	if err != nil {
		return nil, err
	}
	logs.LastModified, err = ptypes.TimestampProto(m)
	if err != nil {
		return nil, err
	}
	logs.Done, err = ptypes.TimestampProto(d)
	if err != nil {
		return nil, err
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
		var s, m, d time.Time

		if err := rows.Scan(&l.Id, &l.PipelineBuildJobID, &l.PipelineBuildID, &s, &m, &d, &l.StepOrder, &l.Val); err != nil {
			return nil, err
		}

		var err error
		l.Start, err = ptypes.TimestampProto(s)
		if err != nil {
			return nil, err
		}
		l.LastModified, err = ptypes.TimestampProto(m)
		if err != nil {
			return nil, err
		}
		l.Done, err = ptypes.TimestampProto(d)
		if err != nil {
			return nil, err
		}

		logs = append(logs, *l)
	}
	return logs, nil
}

func insertLog(db gorp.SqlExecutor, logs *sdk.Log) error {
	if logs.Start == nil {
		logs.Start, _ = ptypes.TimestampProto(time.Now())
	}
	if logs.LastModified == nil {
		logs.LastModified, _ = ptypes.TimestampProto(time.Now())
	}
	if logs.Done == nil {
		logs.Done, _ = ptypes.TimestampProto(time.Now())
	}
	query := `
		INSERT INTO workflow_node_run_job_logs (workflow_node_run_job_id, workflow_node_run_id, start, last_modified, done, step_order, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ID `
	s, errs := ptypes.Timestamp(logs.Start)
	if errs != nil {
		return errs
	}
	m, errm := ptypes.Timestamp(logs.LastModified)
	if errm != nil {
		return errm
	}
	d, errd := ptypes.Timestamp(logs.Done)
	if errd != nil {
		return errd
	}

	return db.QueryRow(query, logs.PipelineBuildJobID, logs.PipelineBuildID, s, m, d, logs.StepOrder, logs.Val).Scan(&logs.Id)
}

func updateLog(db gorp.SqlExecutor, logs *sdk.Log) error {
	if logs.Start == nil {
		logs.Start, _ = ptypes.TimestampProto(time.Now())
	}
	if logs.LastModified == nil {
		logs.LastModified, _ = ptypes.TimestampProto(time.Now())
	}
	if logs.Done == nil {
		logs.Done, _ = ptypes.TimestampProto(time.Now())
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

	s, errs := ptypes.Timestamp(logs.Start)
	if errs != nil {
		return errs
	}
	m, errm := ptypes.Timestamp(logs.LastModified)
	if errm != nil {
		return errm
	}
	d, errd := ptypes.Timestamp(logs.Done)
	if errd != nil {
		return errd
	}

	if _, err := db.Exec(query, logs.PipelineBuildJobID, logs.PipelineBuildID, s, m, d, logs.StepOrder, logs.Val, logs.Id); err != nil {
		return err
	}
	return nil
}
