package workflow

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/protobuf/ptypes"

	"github.com/ovh/cds/sdk"
)

// updateServiceLog Update a service log
func updateServiceLog(db gorp.SqlExecutor, log *sdk.ServiceLog) error {
	query := `
	UPDATE requirement_service_logs
		SET workflow_node_run_job_id = $2,
				workflow_node_run_id = $3,
				requirement_service_name = $4,
				start = $5,
				last_modified = $6,
				value = $7
		WHERE id = $1
	`

	if log.Start == nil {
		log.Start, _ = ptypes.TimestampProto(time.Now())
	}
	if log.LastModified == nil {
		log.LastModified, _ = ptypes.TimestampProto(time.Now())
	}

	start, errs := ptypes.Timestamp(log.Start)
	if errs != nil {
		return errs
	}
	lastModified, errm := ptypes.Timestamp(log.LastModified)
	if errm != nil {
		return errm
	}

	_, errU := db.Exec(query, log.Id, log.WorkflowNodeJobRunID, log.WorkflowNodeRunID, log.ServiceRequirementName, start, lastModified, log.Val)

	return errU
}

// insertServiceLog insert service log into database
func insertServiceLog(db gorp.SqlExecutor, log *sdk.ServiceLog) error {
	query := `
	INSERT INTO requirement_service_logs
		(workflow_node_run_job_id, workflow_node_run_id, requirement_service_name, start, last_modified, value)
		VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id
	`

	if log.Start == nil {
		log.Start, _ = ptypes.TimestampProto(time.Now())
	}
	if log.LastModified == nil {
		log.LastModified, _ = ptypes.TimestampProto(time.Now())
	}

	start, errs := ptypes.Timestamp(log.Start)
	if errs != nil {
		return errs
	}
	lastModified, errm := ptypes.Timestamp(log.LastModified)
	if errm != nil {
		return errm
	}

	return db.QueryRow(query, log.WorkflowNodeJobRunID, log.WorkflowNodeRunID, log.ServiceRequirementName, start, lastModified, log.Val).Scan(&log.Id)
}

// ExistsServiceLog returns the size of service log if exists.
func ExistsServiceLog(db gorp.SqlExecutor, nodeRunJobID int64, serviceName string) (bool, int64, error) {
	query := `
    SELECT octet_length(value) as size
    FROM requirement_service_logs
    WHERE workflow_node_run_job_id = $1 AND requirement_service_name = $2
  `

	var size int64
	if err := db.QueryRow(query, nodeRunJobID, serviceName).Scan(&size); err != nil {
		if sdk.Cause(err) != sql.ErrNoRows {
			return false, 0, sdk.WithStack(err)
		}
		return false, 0, nil
	}

	return true, size, nil
}

// LoadServiceLog load logs for the given job and service name
func LoadServiceLog(db gorp.SqlExecutor, nodeRunJobID int64, serviceName string) (*sdk.ServiceLog, error) {
	query := `
		SELECT id, workflow_node_run_job_id, workflow_node_run_id, requirement_service_name, start, last_modified, value
			FROM requirement_service_logs
		WHERE workflow_node_run_job_id = $1 AND requirement_service_name = $2
	`
	var start, lastModified time.Time
	var log sdk.ServiceLog
	err := db.QueryRow(query, nodeRunJobID, serviceName).Scan(&log.Id, &log.WorkflowNodeJobRunID, &log.WorkflowNodeRunID, &log.ServiceRequirementName, &start, &lastModified, &log.Val)
	if err != nil {
		return nil, err
	}
	var errT error
	log.Start, errT = ptypes.TimestampProto(start)
	if errT != nil {
		return nil, errT
	}
	log.LastModified, errT = ptypes.TimestampProto(lastModified)
	if errT != nil {
		return nil, errT
	}

	return &log, nil
}

// LoadServicesLogsByJob retrieves services logs for a run
func LoadServicesLogsByJob(db gorp.SqlExecutor, nodeJobRunID int64) ([]sdk.ServiceLog, error) {
	query := `
		SELECT id, workflow_node_run_job_id, workflow_node_run_id, requirement_service_name, start, last_modified, value
			FROM requirement_service_logs
		WHERE workflow_node_run_job_id = $1
	`
	rows, err := db.Query(query, nodeJobRunID)
	if err != nil {
		return nil, err
	}

	var logs []sdk.ServiceLog
	for rows.Next() {
		var start, lastModified time.Time
		var log sdk.ServiceLog
		errS := rows.Scan(&log.Id, &log.WorkflowNodeJobRunID, &log.WorkflowNodeRunID, &log.ServiceRequirementName, &start, &lastModified, &log.Val)
		if errS != nil {
			return nil, errS
		}

		var errT error
		log.Start, errT = ptypes.TimestampProto(start)
		if errT != nil {
			return nil, errT
		}
		log.LastModified, errT = ptypes.TimestampProto(lastModified)
		if errT != nil {
			return nil, errT
		}

		logs = append(logs, log)
	}

	return logs, nil
}
