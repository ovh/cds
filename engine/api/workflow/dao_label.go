package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
)

// InsertLabel create new label in database
func InsertLabel(db gorp.SqlExecutor, label *sdk.Label) error {
	if err := label.Validate(); err != nil {
		return sdk.WrapError(err, "InsertLabel>")
	}
	labelDB := Label(*label)
	if err := db.Insert(&labelDB); err != nil {
		return sdk.WrapError(err, "InsertLabel> Cannot insert label %s with color %s and project id %d", label.Name, label.Color, label.ProjectID)
	}
	label.ID = labelDB.ID

	return nil
}

// LabelWorkflow link a label on a workflow given his workflow id
func LabelWorkflow(db gorp.SqlExecutor, labelID, workflowID int64) error {
	if _, err := db.Exec("INSERT INTO workflow_label_workflow (label_id, workflow_id) VALUES ($1, $2)", labelID, workflowID); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == database.ViolateUniqueKeyPGCode {
			return sdk.WrapError(sdk.ErrConflict, "LabelWorkflow> this label %d is already linked to workflow %d", labelID, workflowID)
		}
		return sdk.WrapError(err, "LabelWorkflow> Cannot link label %d to workflow %d", labelID, workflowID)
	}

	return nil
}

// UnLabelWorkflow unlink a label on a workflow given his workflow id
func UnLabelWorkflow(db gorp.SqlExecutor, labelID, workflowID int64) error {
	if _, err := db.Exec("DELETE FROM workflow_label_workflow WHERE label_id = $1 AND workflow_id = $2", labelID, workflowID); err != nil {
		return sdk.WrapError(err, "UnLabelWorkflow> Cannot unlink label %d to workflow %d", labelID, workflowID)
	}

	return nil
}

// LabelByName return a label given his name and project id
func LabelByName(db gorp.SqlExecutor, projectID int64, labelName string) (sdk.Label, error) {
	var label sdk.Label
	err := db.SelectOne(&label, "SELECT workflow_label.* FROM workflow_label WHERE project_id = $1 AND name = $2", projectID, labelName)

	return label, err
}

// Labels return list of labels given a workflow ID
func Labels(db gorp.SqlExecutor, workflowID int64) ([]sdk.Label, error) {
	var labels []sdk.Label
	query := `
	SELECT workflow_label.*
		FROM workflow_label
			JOIN workflow_label_workflow ON workflow_label.id = workflow_label_workflow.label_id
		WHERE workflow_label_workflow.workflow_id = $1
	`
	if _, err := db.Select(&labels, query, workflowID); err != nil {
		if err == sql.ErrNoRows {
			return labels, nil
		}
		return labels, sdk.WrapError(err, "Labels> Cannot load labels")
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
	SELECT workflow_label.*
		FROM workflow_label
			JOIN workflow_label_workflow ON workflow_label.id = workflow_label_workflow.label_id
		WHERE workflow_label.project_id = $1
	`
	if _, err := db.Select(&labels, query, projectID); err != nil {
		if err == sql.ErrNoRows {
			return labels, nil
		}
		return labels, sdk.WrapError(err, "LabelsByProjectID> Cannot load labels")
	}

	return labels, nil
}
