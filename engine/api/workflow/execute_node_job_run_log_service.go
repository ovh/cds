package workflow

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

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

	var now = time.Now()

	if log.Start == nil {
		log.Start = &now
	}
	if log.LastModified == nil {
		log.LastModified = &now
	}

	_, errU := db.Exec(query, log.ID, log.WorkflowNodeJobRunID, log.WorkflowNodeRunID, log.ServiceRequirementName, log.Start, log.LastModified, log.Val)

	return sdk.WithStack(errU)
}

// insertServiceLog insert service log into database
func insertServiceLog(db gorp.SqlExecutor, log *sdk.ServiceLog) error {
	query := `
	INSERT INTO requirement_service_logs
		(workflow_node_run_job_id, workflow_node_run_id, requirement_service_name, start, last_modified, value)
		VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id
	`

	var now = time.Now()
	if log.Start == nil {
		log.Start = &now
	}
	if log.LastModified == nil {
		log.LastModified = &now
	}

	return db.QueryRow(query, log.WorkflowNodeJobRunID, log.WorkflowNodeRunID, log.ServiceRequirementName, log.Start, log.LastModified, log.Val).Scan(&log.ID)
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
	var log sdk.ServiceLog
	var s, m pq.NullTime
	if err := db.QueryRow(query, nodeRunJobID, serviceName).Scan(&log.ID, &log.WorkflowNodeJobRunID, &log.WorkflowNodeRunID, &log.ServiceRequirementName, &s, &m, &log.Val); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if s.Valid {
		log.Start = &s.Time
	}
	if m.Valid {
		log.LastModified = &m.Time
	}

	return &log, nil
}
