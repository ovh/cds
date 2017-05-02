package workflow

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"

	"github.com/ovh/cds/sdk"
)

// Workflow is a gorp wrapper around sdk.Workflow
type Workflow sdk.Workflow

// Node is a gorp wrapper around sdk.WorkflowNode
type Node sdk.WorkflowNode

// NodeContext is a gorp wrapper around sdk.WorkflowNodeContext
type NodeContext sdk.WorkflowNodeContext

// NodeTrigger is a gorp wrapper around sdk.WorkflowNodeTrigger
type NodeTrigger sdk.WorkflowNodeTrigger

func init() {
	gorpmapping.Register(gorpmapping.New(Workflow{}, "workflow", true, "id"))
	gorpmapping.Register(gorpmapping.New(Node{}, "workflow_node", true, "id"))
	gorpmapping.Register(gorpmapping.New(NodeTrigger{}, "workflow_node_trigger", true, "id"))
	gorpmapping.Register(gorpmapping.New(NodeContext{}, "workflow_node_context", true, "id"))
}
