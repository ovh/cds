package audit

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type auditWorkflow sdk.AuditWorklflow

func init() {
	gorpmapping.Register(
		gorpmapping.New(warning{}, "workflow_audit", true, "id"),
	)
}
