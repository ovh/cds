package scheduler

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk/log"
)

// PurgeRun to purge all history of workflow run
func PurgeRun(db *gorp.DbMap) error {
	log.Debug("PurgeRun> Deleting all workflow run marked to delete...")

	return workflow.DeleteWorkflowRunsHistory(db)
}
