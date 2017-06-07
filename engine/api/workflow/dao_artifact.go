package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func loadArtifactByNodeRunID(db gorp.SqlExecutor, nodeRunID int64) ([]sdk.WorkflowNodeRunArtifact, error) {
	var artifactsGorp []NodeRunArtifact
	if _, err := db.Select(&artifactsGorp, "SELECT * FROM workflow_run_artifacts WHERE workflow_node_run_id = $1", nodeRunID); err != nil {
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
