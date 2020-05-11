package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// LabelWorkflow link a label on a workflow given his workflow id
func LabelWorkflow(db gorp.SqlExecutor, labelID, workflowID int64) error {
	log.Debug("LabelWorkflow> %d %d", labelID, workflowID)
	if _, err := db.Exec("INSERT INTO project_label_workflow (label_id, workflow_id) VALUES ($1, $2)", labelID, workflowID); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == gorpmapping.ViolateUniqueKeyPGCode {
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

type dbLabel struct {
	sdk.Label
	WorkflowID int64 `db:"workflow_id"`
}

// LoadLabels return list of labels given a workflow ID
func LoadLabels(db gorp.SqlExecutor, workflowIDs ...int64) ([]sdk.Label, error) {
	var labels []dbLabel
	query := `
	SELECT project_label.*, project_label_workflow.workflow_id
	FROM project_label
	JOIN project_label_workflow ON project_label.id = project_label_workflow.label_id
	WHERE project_label_workflow.workflow_id = ANY($1)`

	if _, err := db.Select(&labels, query, pq.Int64Array(workflowIDs)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot load labels")
	}

	log.Debug("=> %+v", labels)

	var result = make([]sdk.Label, 0, len(labels))
	for i := range labels {
		labels[i].Label.WorkflowID = labels[i].WorkflowID
		result = append(result, labels[i].Label)
	}

	return result, nil
}
