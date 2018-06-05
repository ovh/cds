package audit

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertWorkflowAudit insert a workflow audit
func InsertWorkflowAudit(db gorp.SqlExecutor, a sdk.AuditWorklflow) error {
	audit := auditWorkflow(a)
	return db.Insert(&audit)
}
