package workflowtemplate

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.WorkflowTemplate{}, "workflow_template", true, "id"),
		gorpmapping.New(sdk.WorkflowTemplateInstance{}, "workflow_template_instance", true, "id"),
		gorpmapping.New(sdk.AuditWorkflowTemplate{}, "workflow_template_audit", true, "id"),
	)
}
