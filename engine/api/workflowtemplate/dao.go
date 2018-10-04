package workflowtemplate

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// GetAll returns all existing templates.
func GetAll(db gorp.SqlExecutor) ([]sdk.WorkflowTemplate, error) {
	ws := []workflow{}

	if _, err := db.Select(&ws, "SELECT * FROM workflow_templates"); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound // TODO withstack
		}
		return nil, sdk.WrapError(err, "Cannot get workflow")
	}

	wsP := make([]*workflow, len(ws))
	for i := 0; i < len(ws); i++ {
		wsP[i] = &ws[i]
	}

	if err := aggregateWorkflowTemplateJSONBFields(db, wsP...); err != nil {
		return nil, sdk.WrapError(err, "Cannot aggregate json fields on workflow")
	}

	wts := make([]sdk.WorkflowTemplate, len(ws))
	for i := 0; i < len(ws); i++ {
		wts[i] = sdk.WorkflowTemplate(ws[i])
	}

	return wts, nil
}

// GetByID returns the workflow template for given id.
func GetByID(db gorp.SqlExecutor, id int64) (*sdk.WorkflowTemplate, error) {
	w := workflow{}

	if err := db.SelectOne(&w, "SELECT * FROM workflow_templates WHERE id=$1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNotFound // TODO withstack
		}
		return nil, sdk.WrapError(err, "Cannot get workflow")
	}

	if err := aggregateWorkflowTemplateJSONBFields(db, &w); err != nil {
		return nil, sdk.WrapError(err, "Cannot aggregate json fields on workflow")
	}

	wt := sdk.WorkflowTemplate(w)
	return &wt, nil
}

func idsToQueryString(ids []int64) []string {
	res := make([]string, len(ids))
	for i, id := range ids {
		res[i] = fmt.Sprintf("%d", id)
	}
	return res
}

func aggregateWorkflowTemplateJSONBFields(db gorp.SqlExecutor, wts ...*workflow) error {
	if len(wts) == 0 {
		return nil
	}

	ids := make([]int64, len(wts))
	for i, wt := range wts {
		ids[i] = wt.ID
	}

	var query string
	var args interface{}
	if len(wts) > 1 {
		query = "SELECT id, pipelines, parameters FROM workflow_templates WHERE id = ANY(string_to_array($1, ',')::int[])"
		args = strings.Join(idsToQueryString(ids), ",")
	} else {
		query = "SELECT id, pipelines, parameters FROM workflow_templates WHERE id=$1"
		args = ids[0]
	}

	rows, err := db.Query(query, args)
	if err != nil {
		return err // TODO withstack
	}
	defer rows.Close()

	m := map[int64]*workflow{}
	for _, wt := range wts {
		m[wt.ID] = wt
	}

	for rows.Next() {
		var id int64
		var pipelines, parameters []byte
		if err := rows.Scan(&id, &pipelines, &parameters); err != nil {
			return err // TODO withstack
		}
		if wt, ok := m[id]; ok {
			if err := json.Unmarshal(pipelines, &wt.Pipelines); err != nil {
				return err // TODO withstack
			}
			if err := json.Unmarshal(parameters, &wt.Parameters); err != nil {
				return err // TODO withstack
			}
		}
	}

	return nil
}

// InsertWorkflow template in database.
func InsertWorkflow(db gorp.SqlExecutor, wt *sdk.WorkflowTemplate) error {
	wtdb := workflow(*wt)

	if err := db.Insert(&wtdb); err != nil {
		return sdk.WrapError(err, "Unable to insert workflow template %s", wt.Name)
	}

	*wt = sdk.WorkflowTemplate(wtdb)

	return nil
}

// PostInsert is a db hook on workflow
func (w *workflow) PostInsert(s gorp.SqlExecutor) error { return w.PostUpdate(s) }

// PostUpdate is a db hook on workflow
func (w *workflow) PostUpdate(s gorp.SqlExecutor) error {
	pipelines, err := gorpmapping.JSONToNullString(w.Pipelines)
	if err != nil {
		return sdk.WrapError(err, "Unable to stringify pipelines")
	}

	parameters, err := gorpmapping.JSONToNullString(w.Parameters)
	if err != nil {
		return sdk.WrapError(err, "Unable to stringify parameters")
	}

	query := "UPDATE workflow_templates SET pipelines=$1, parameters=$2 WHERE id=$3"
	_, err = s.Exec(query, pipelines, parameters, w.ID)
	return sdk.WrapError(err, "Unable to update pipelines and parameters")
}
