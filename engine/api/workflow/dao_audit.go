package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// insertAudit insert a workflow audit
func insertAudit(db gorp.SqlExecutor, a sdk.AuditWorklflow) error {
	audit := auditWorkflow(a)
	return db.Insert(&audit)
}

// LoadAudits Load audits for the given workflow
func LoadAudits(db gorp.SqlExecutor, workflowID int64) ([]sdk.AuditWorklflow, error) {
	query := `
		SELECT * FROM workflow_audit WHERE workflow_id = $1
	`
	var audits []auditWorkflow
	if _, err := db.Select(&audits, query, workflowID); err != nil {
		return nil, sdk.WrapError(err, "workflow.loadAudits> Unable to load audits")
	}

	workflowAudits := make([]sdk.AuditWorklflow, len(audits), len(audits))
	for i := range audits {
		workflowAudits[i] = sdk.AuditWorklflow(audits[i])
	}
	return workflowAudits, nil
}

// LoadAudit Load audit for the given workflow
func LoadAudit(db gorp.SqlExecutor, auditID int64) (sdk.AuditWorklflow, error) {
	var audit auditWorkflow
	if err := db.SelectOne(&audit, "SELECT * FROM workflow_audit WHERE id = $1", auditID); err != nil {
		return sdk.AuditWorklflow{}, sdk.WrapError(err, "workflow.LoadAudit> Unable to load audit")
	}

	return sdk.AuditWorklflow(audit), nil
}
