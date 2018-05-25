package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadWorkfowArtifactByHash retrieves an artiface using its download hash
func LoadWorkfowArtifactByHash(db gorp.SqlExecutor, hash string) (*sdk.WorkflowNodeRunArtifact, error) {
	var artGorp NodeRunArtifact
	query := `SELECT
				id,
				name,
				tag,
				workflow_node_run_id,
				download_hash,
				size,
				perm,
				md5sum,
				object_path,
				created,
				workflow_run_id,
				coalesce(sha512sum, '')
		  FROM workflow_node_run_artifacts
		  WHERE workflow_node_run_artifacts.download_hash = $1`
	if err := db.SelectOne(&artGorp, query, hash); err != nil {
		return nil, err
	}
	art := sdk.WorkflowNodeRunArtifact(artGorp)
	return &art, nil

}

// LoadArtifactByIDs Load artifact by workflow ID and artifact ID
func LoadArtifactByIDs(db gorp.SqlExecutor, workflowID, artifactID int64) (*sdk.WorkflowNodeRunArtifact, error) {
	var artGorp NodeRunArtifact
	query := `
		SELECT
			workflow_node_run_artifacts.id,
			workflow_node_run_artifacts.name,
			workflow_node_run_artifacts.tag,
			workflow_node_run_artifacts.workflow_node_run_id,
			workflow_node_run_artifacts.download_hash,
			workflow_node_run_artifacts.size,
			workflow_node_run_artifacts.perm,
			workflow_node_run_artifacts.md5sum,
			workflow_node_run_artifacts.object_path,
			workflow_node_run_artifacts.created,
			workflow_node_run_artifacts.workflow_run_id,
			workflow_node_run_artifacts.coalesce(sha512sum, '')
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
	if _, err := db.Select(&artifactsGorp, `SELECT
			id,
			name,
			tag,
			workflow_node_run_id,
			download_hash,
			size,
			perm,
			md5sum,
			object_path,
			created,
			workflow_run_id,
			coalesce(sha512sum, '')
		FROM workflow_node_run_artifacts WHERE workflow_node_run_id = $1`, nodeRunID); err != nil {
		return nil, err
	}

	artifacts := make([]sdk.WorkflowNodeRunArtifact, len(artifactsGorp))
	for i := range artifactsGorp {
		artifacts[i] = sdk.WorkflowNodeRunArtifact(artifactsGorp[i])
	}
	return artifacts, nil
}

// InsertArtifact insert in table workflow_artifacts
func InsertArtifact(db gorp.SqlExecutor, a *sdk.WorkflowNodeRunArtifact) error {
	wArtifactDB := NodeRunArtifact(*a)
	if err := db.Insert(&wArtifactDB); err != nil {
		return err
	}
	a.ID = wArtifactDB.ID
	return nil
}
