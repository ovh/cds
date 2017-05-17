package queue

import "github.com/ovh/cds/sdk"

var (
	chanWorkflowNodeRun = make(chan *sdk.WorkflowNodeRun)
)

func RunWorkflow(w *sdk.Workflow, n *sdk.WorkflowNodeRun) {

}
