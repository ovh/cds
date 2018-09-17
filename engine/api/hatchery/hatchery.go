package hatchery

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
)

// CountHatcheries retrieves in database the number of hatcheries
func CountHatcheries(db gorp.SqlExecutor, wfNodeRunID int64) (int64, error) {
	query := `
	SELECT COUNT(1)
		FROM services
		WHERE (
			services.type = 'hatchery'
			AND services.group_id = ANY(
				SELECT DISTINCT(project_group.group_id)
					FROM workflow_node_run
						JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id
						JOIN workflow ON workflow.id = workflow_run.workflow_id
						JOIN project ON project.id = workflow.project_id
						JOIN project_group ON project_group.project_id = project.id
				WHERE workflow_node_run.id = $1
				AND project_group.role >= 5
			)
			OR
			services.group_id = $2
		)
	`
	return db.SelectInt(query, wfNodeRunID, group.SharedInfraGroup.ID)
}

// LoadHatcheriesCountByNodeJobRunID retrieves in database the number of hatcheries given the node job run id
func LoadHatcheriesCountByNodeJobRunID(db gorp.SqlExecutor, wfNodeJobRunID int64) (int64, error) {
	query := `
	SELECT COUNT(1)
		FROM services
		WHERE (
			services.type = 'hatchery'
			AND services.group_id = ANY(
				SELECT DISTINCT(project_group.group_id)
					FROM workflow_node_run_job
						JOIN workflow_node_run ON workflow_node_run.id = workflow_node_run_job.workflow_node_run_id
						JOIN workflow_run ON workflow_run.id = workflow_node_run.workflow_run_id
						JOIN workflow ON workflow.id = workflow_run.workflow_id
						JOIN project ON project.id = workflow.project_id
						JOIN project_group ON project_group.project_id = project.id
				WHERE workflow_node_run.id = $1
				AND project_group.role >= 5
			)
			OR
			services.group_id = $2
		)
	`
	return db.SelectInt(query, wfNodeJobRunID, group.SharedInfraGroup.ID)
}
