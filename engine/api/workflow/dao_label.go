package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// LabelWorkflow link a label on a workflow given his workflow id
func LabelWorkflow(db gorp.SqlExecutor, labelID, workflowID int64) error {
	if _, err := db.Exec("INSERT INTO project_label_workflow (label_id, workflow_id) VALUES ($1, $2)", labelID, workflowID); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == database.ViolateUniqueKeyPGCode {
			return sdk.WrapError(sdk.ErrConflict, "LabelWorkflow> this label %d is already linked to workflow %d", labelID, workflowID)
		}
		return sdk.WrapError(err, "Cannot link label %d to workflow %d", labelID, workflowID)
	}

	return nil
}

// UnLabelWorkflow unlink a label on a workflow given his workflow id
func UnLabelWorkflow(db gorp.SqlExecutor, labelID, workflowID int64) error {
	if _, err := db.Exec("DELETE FROM project_label_workflow WHERE label_id = $1 AND workflow_id = $2", labelID, workflowID); err != nil {
		return sdk.WrapError(err, "Cannot unlink label %d to workflow %d", labelID, workflowID)
	}

	return nil
}

// Labels return list of labels given a workflow ID
func Labels(db gorp.SqlExecutor, workflowID int64) ([]sdk.Label, error) {
	var labels []sdk.Label
	query := `
	SELECT project_label.*
		FROM project_label
			JOIN project_label_workflow ON project_label.id = project_label_workflow.label_id
		WHERE project_label_workflow.workflow_id = $1
	`
	if _, err := db.Select(&labels, query, workflowID); err != nil {
		if err == sql.ErrNoRows {
			return labels, nil
		}
		return labels, sdk.WrapError(err, "Cannot load labels")
	}
	for i := range labels {
		labels[i].WorkflowID = workflowID
	}

	return labels, nil
}

// LabelsByProjectID return list of labels given a project ID
func LabelsByProjectID(db gorp.SqlExecutor, projectID int64) ([]sdk.Label, error) {
	var labels []sdk.Label
	query := `
	SELECT project_label.*
		FROM project_label
			JOIN project_label_workflow ON project_label.id = project_label_workflow.label_id
		WHERE project_label.project_id = $1
	`
	if _, err := db.Select(&labels, query, projectID); err != nil {
		if err == sql.ErrNoRows {
			return labels, nil
		}
		return labels, sdk.WrapError(err, "Cannot load labels")
	}

	return labels, nil
}
