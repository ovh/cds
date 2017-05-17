package workflow

import (
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// processWorkflowRun triggers workflow node for every workflow.
// It contains all the logic for triggers and joins processing.
// It calls insertPipelineBuild
func processWorkflowRun(db gorp.SqlExecutor, w *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) error {
	t0 := time.Now()
	log.Debug("processWorkflowRun> Begin [#%d]%s", w.Number, w.Workflow.Name)
	defer func() {
		log.Debug("processWorkflowRun> End [#%d]%s - %.3fs", w.Number, w.Workflow.Name, time.Since(t0).Seconds())
	}()

	//Checks the root
	if len(w.WorkflowNodeRuns) == 0 {
		//Run the root
		if err := processWorkflowNodeRun(db, w, w.Workflow.Root, nil, hookEvent, manual); err != nil {
			return sdk.WrapError(err, "processWorkflowRun> Unable to process workflow node run")
		}
	}

	//Checks the triggers
	for _, nodeRun := range w.WorkflowNodeRuns {

		//Find the node in the workflow
		node := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
		if node == nil {
			return sdk.ErrWorkflowNodeNotFound
		}

		for _, t := range node.Triggers {
			//Check conditions
			if err := processWorkflowNodeRun(db, w, node, &t.ID, nil, nil); err != nil {
				log.Warning("processWorkflowRun> Unable to process node ID=%d", node.ID)
			}
		}
	}

	return nil
}

func processWorkflowNodeRun(db gorp.SqlExecutor, w *sdk.WorkflowRun, n *sdk.WorkflowNode, triggerID *int64, h *sdk.WorkflowNodeRunHookEvent, m *sdk.WorkflowNodeRunManual) error {

	run := sdk.WorkflowNodeRun{
		LastModified:   time.Now(),
		Start:          time.Now(),
		Number:         w.Number,
		SubNumber:      0, //Manage it
		WorkflowRunID:  w.ID,
		WorkflowNodeID: n.ID,
	}

	if triggerID != nil {
		run.TriggerID = *triggerID
	} else if h != nil {
		run.HookEvent = h
	} else if m != nil {
		run.Manual = m
	}

	return nil
}

func retryWorkflowRunProcess(w *sdk.WorkflowRun, e error) {
	//Retry
}
