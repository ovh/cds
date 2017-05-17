package workflow

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const processWorkflowQueue = "process_workflow"

func Do() {
	for {
		w := &sdk.WorkflowRun{}
		cache.Dequeue(processWorkflowQueue, w)
		db := database.DBMap(database.DB())
		if db != nil {
			if err := processWorkflowRun(db, w); err != nil {
				retryWorkflowRunProcess(w, err)
			}
			continue
		}
		retryWorkflowRunProcess(w, fmt.Errorf("Database unavailable"))
	}
}

// processWorkflowRun triggers workflow node for every workflow.
// It contains all the logic for triggers and joins processing.
// It calls insertPipelineBuild
func processWorkflowRun(db gorp.SqlExecutor, w *sdk.WorkflowRun) error {
	t0 := time.Now()
	log.Debug("processWorkflowRun> Begin %s/%s", w.ProjectKey, w.WorkflowName)
	defer func() {
		log.Debug("processWorkflowRun> End %s/%s - %.3fs", w.ProjectKey, w.WorkflowName, time.Since(t0).Seconds())
	}()

	//Checks the triggers
	for _, nodeRun := range w.WorkflowNodeRuns {
		//Load the pipeline build
		var errbuild error
		nodeRun.PipelineBuild, errbuild = pipeline.LoadPipelineBuildByID(db, nodeRun.PipelineBuildID)
		if errbuild != nil {
			log.Warning("processWorkflowRun> Unable to load pipeline build ID=%d", nodeRun.PipelineBuildID)
			return errbuild
		}

		//Find the node in the workflow
		node := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
		if node == nil {
			log.Warning("processWorkflowRun> Unable to find node ID=%d", nodeRun.PipelineBuildID)
			return sdk.ErrWorkflowNodeNotFound
		}

		for _, t := range node.Triggers {
			//Check conditions
			if err := processWorkflowNodeRun(db, w, node, t.ID, nil, nil); err != nil {
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
