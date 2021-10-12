package workflow

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/sdk"
)

// AddWorkflowIntegration link a project integration on a workflow
func AddWorkflowIntegration(db gorp.SqlExecutor, integ *sdk.WorkflowProjectIntegration) error {
	dbInteg := dbWorkflowProjectIntegration(*integ)
	if err := gorpmapping.Insert(db, &dbInteg); err != nil {
		return err
	}
	integ.ID = dbInteg.ID
	return nil
}

// LoadWorkflowIntegrationsByWorkflowID load workflow integrations by workflowid
func LoadWorkflowIntegrationsByWorkflowID(ctx context.Context, db gorp.SqlExecutor, id int64) ([]sdk.WorkflowProjectIntegration, error) {
	query := gorpmapping.NewQuery(`
		SELECT workflow_project_integration.*
		FROM workflow_project_integration
		WHERE workflow_project_integration.workflow_id = $1
	`).Args(id)
	return loadAll(ctx, db, query)
}

// RemoveIntegrationFromWorkflow remove a project integration on a workflow
func RemoveIntegrationFromWorkflow(db gorp.SqlExecutor, workflowInteg sdk.WorkflowProjectIntegration) error {
	dbInteg := dbWorkflowProjectIntegration(workflowInteg)
	return gorpmapping.Delete(db, &dbInteg)
}

// DeleteIntegrationsFromWorkflow remove a project integration on a workflow
func DeleteIntegrationsFromWorkflow(db gorp.SqlExecutor, workflowID int64) error {
	query := "DELETE FROM workflow_project_integration WHERE workflow_id = $1"
	if _, err := db.Exec(query, workflowID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func loadAll(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.WorkflowProjectIntegration, error) {
	var integDB []dbWorkflowProjectIntegration
	if err := gorpmapping.GetAll(ctx, db, query, &integDB); err != nil {
		return nil, err
	}
	var integrations = make([]sdk.WorkflowProjectIntegration, 0, len(integDB))
	for _, workflowInteg := range integDB {
		pi, err := integration.LoadProjectIntegrationByID(ctx, db, workflowInteg.ProjectIntegrationID)
		if err != nil {
			return nil, err
		}
		pi.Blur()
		workflowInteg.ProjectIntegration = *pi
		integrations = append(integrations, sdk.WorkflowProjectIntegration(workflowInteg))
	}
	return integrations, nil
}
