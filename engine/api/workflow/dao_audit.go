package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertAudit insert a workflow audit
func InsertAudit(db gorp.SqlExecutor, a *sdk.AuditWorklflow) error {
	audit := auditWorkflow(*a)
	if err := db.Insert(&audit); err != nil {
		return sdk.WrapError(err, "Unable to insert audit")
	}
	a.ID = audit.ID
	return nil
}

// LoadAudits Load audits for the given workflow
func LoadAudits(db gorp.SqlExecutor, workflowID int64) ([]sdk.AuditWorklflow, error) {
	query := `
		SELECT * FROM workflow_audit WHERE workflow_id = $1
	`
	var audits []auditWorkflow
	if _, err := db.Select(&audits, query, workflowID); err != nil {
		return nil, sdk.WrapError(err, "Unable to load audits")
	}

	workflowAudits := make([]sdk.AuditWorklflow, len(audits), len(audits))
	for i := range audits {
		workflowAudits[i] = sdk.AuditWorklflow(audits[i])
	}
	return workflowAudits, nil
}

// LoadAudit Load audit for the given workflow
func LoadAudit(db gorp.SqlExecutor, auditID int64, workflowID int64) (sdk.AuditWorklflow, error) {
	var audit auditWorkflow
	if err := db.SelectOne(&audit, "SELECT * FROM workflow_audit WHERE id = $1 AND workflow_id = $2", auditID, workflowID); err != nil {
		return sdk.AuditWorklflow{}, sdk.WrapError(err, "Unable to load audit")
	}

	return sdk.AuditWorklflow(audit), nil
}
