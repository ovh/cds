package workflowtemplate

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.WorkflowTemplate{}, "workflow_templates", true, "id"),
	)
}
