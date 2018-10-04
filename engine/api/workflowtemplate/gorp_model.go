package workflowtemplate

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

const tableWorkflowTemplates = "workflow_templates"

type workflow sdk.WorkflowTemplate

func init() {
	gorpmapping.Register(
		gorpmapping.New(workflow{}, tableWorkflowTemplates, true, "id"),
	)
}
