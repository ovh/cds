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
