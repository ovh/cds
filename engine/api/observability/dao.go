package observability

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// findProjetKeyForNodeRunJob load the project key from a workflow_node_run_job ID
func findProjetKeyForNodeRunJob(db gorp.SqlExecutor, id int64) (string, error) {
	query := `select project.projectkey from project
	join workflow on workflow.project_id = project.id
	join workflow_run on workflow_run.workflow_id = workflow.id
	join workflow_node_run on workflow_node_run.workflow_run_id = workflow_run.id
	join workflow_node_run_job on workflow_node_run_job.workflow_node_run_id = workflow_node_run.id
	where workflow_node_run_job.id = $1`
	pkey, err := db.SelectNullStr(query, id)
	if err != nil {
		return "", sdk.WrapError(err, "FindProjetKeyForNodeRunJob")
	}
	if pkey.Valid {
		return pkey.String, nil
	}
	log.Warning("FindProjetKeyForNodeRunJob> project key not found for node run job %d", id)
	return "", nil
}
