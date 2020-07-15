package workflowtemplate

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.WorkflowTemplate{}, "workflow_template", true, "id"),
		gorpmapping.New(sdk.WorkflowTemplateInstance{}, "workflow_template_instance", true, "id"),
		gorpmapping.New(sdk.AuditWorkflowTemplate{}, "workflow_template_audit", true, "id"),
		gorpmapping.New(sdk.AuditWorkflowTemplateInstance{}, "workflow_template_instance_audit", true, "id"),
		gorpmapping.New(sdk.WorkflowTemplateBulk{}, "workflow_template_bulk", true, "id"),
	)
}
