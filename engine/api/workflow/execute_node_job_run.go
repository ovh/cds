package workflow

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

/*
	router.Handle("/build/{id}/log", POST(addBuildLogHandler))
	router.Handle("/build/{id}/step", POST(updateStepStatusHandler))
	router.Handle("/queue/{id}/take", POST(takeWorkflowNodeJobRunHandler))
*/

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

	if err := updateWorkflowNodeJobRun(db, job); err != nil {
		return sdk.WrapError(err, "workflow.UpdateNodeJobRunStatus> Cannot update WorkflowNodeJobRun %d", job.ID)
	}

	node, errLoad := loadWorkflowNodeRunByID(db, job.WorkflowNodeRunID)
	if errLoad != nil {
		return errLoad
	}

	event.PublishJobRun(node, job)
	return nil
}
