package hatchery

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
)

// TODO yesnault to delete

// LoadHatchery fetch hatchery info from database given UID
// func LoadHatchery(db gorp.SqlExecutor, uid, name string) (*sdk.Hatchery, error) {
// 	var hatchery dbHatchery
// 	query := `SELECT * FROM hatchery WHERE uid = $1 AND name = $2`
// 	if err := db.SelectOne(&hatchery, query, uid, name); err != nil {
// 		if err != sql.ErrNoRows {
// 			return nil, sdk.WrapError(err, "LoadHatchery> unable to load hachery %s with uid: %s", name, uid)
// 		}
// 		return nil, sdk.ErrNotFound
// 	}
// 	h := sdk.Hatchery(hatchery)
// 	return &h, nil
// }

// // LoadHatcheryByName fetch hatchery info from database given name
// func LoadHatcheryByName(db gorp.SqlExecutor, name string) (*sdk.Hatchery, error) {
// 	var hatchery dbHatchery
// 	query := `SELECT * FROM hatchery WHERE name = $1`
// 	if err := db.SelectOne(&hatchery, query, name); err != nil {
// 		if err != sql.ErrNoRows {
// 			return nil, sdk.WrapError(err, "LoadHatcheryByName> unable to load hachery %s", name)
// 		}
// 		return nil, sdk.ErrNotFound
// 	}
// 	h := sdk.Hatchery(hatchery)
// 	return &h, nil
// }

// CountHatcheries retrieves in database the number of hatcheries
func CountHatcheries(db gorp.SqlExecutor, wfNodeRunID int64) (int64, error) {
	query := `
	SELECT COUNT(1)
		FROM hatchery
		WHERE (
			hatchery.group_id = ANY(
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
			hatchery.group_id = $2
		)
	`
	return db.SelectInt(query, wfNodeRunID, group.SharedInfraGroup.ID)
}

// LoadHatcheriesCountByNodeJobRunID retrieves in database the number of hatcheries given the node job run id
func LoadHatcheriesCountByNodeJobRunID(db gorp.SqlExecutor, wfNodeJobRunID int64) (int64, error) {
	query := `
	SELECT COUNT(1)
		FROM hatchery
		WHERE (
			hatchery.group_id = ANY(
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
			hatchery.group_id = $2
		)
	`
	return db.SelectInt(query, wfNodeJobRunID, group.SharedInfraGroup.ID)
}
