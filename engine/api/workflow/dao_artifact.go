package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadArtifactByIDs Load artifact by workflow ID and artifact ID
func LoadArtifactByIDs(db gorp.SqlExecutor, workflowID, artifactID int64) (*sdk.WorkflowNodeRunArtifact, error) {
	var artGorp NodeRunArtifact
	query := `
		SELECT *
		FROM workflow_node_run_artifacts
		JOIN workflow_run ON workflow_run.id = workflow_node_run_artifacts.workflow_run_id
		WHERE workflow_run.workflow_id = $1 AND workflow_node_run_artifacts.id = $2

	`
	if err := db.SelectOne(&artGorp, query, workflowID, artifactID); err != nil {
		return nil, err
	}
	art := sdk.WorkflowNodeRunArtifact(artGorp)
	return &art, nil
}

func loadArtifactByNodeRunID(db gorp.SqlExecutor, nodeRunID int64) ([]sdk.WorkflowNodeRunArtifact, error) {
	var artifactsGorp []NodeRunArtifact
	if _, err := db.Select(&artifactsGorp, "SELECT * FROM workflow_node_run_artifacts WHERE workflow_node_run_id = $1", nodeRunID); err != nil {
		return nil, err
	}

	artifacts := make([]sdk.WorkflowNodeRunArtifact, len(artifactsGorp))
	for i := range artifactsGorp {
		artifacts[i] = sdk.WorkflowNodeRunArtifact(artifactsGorp[i])
	}
	return artifacts, nil
}

//insertArtifact insert in table workflow_artifacts
func InsertArtifact(db gorp.SqlExecutor, a *sdk.WorkflowNodeRunArtifact) error {
	wArtifactDB := NodeRunArtifact(*a)
	if err := db.Insert(&wArtifactDB); err != nil {
		return err
	}
	a.ID = wArtifactDB.ID
	return nil
}
