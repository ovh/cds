package workflow

import (
	"time"

	"fmt"

	"sort"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// processWorkflowRun triggers workflow node for every workflow.
// It contains all the logic for triggers and joins processing.
// It calls insertPipelineBuild
func processWorkflowRun(db gorp.SqlExecutor, w *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual, startingFromNode *int64) error {
	t0 := time.Now()
	log.Debug("processWorkflowRun> Begin [#%d]%s", w.Number, w.Workflow.Name)
	defer func() {
		log.Debug("processWorkflowRun> End [#%d]%s - %.3fs", w.Number, w.Workflow.Name, time.Since(t0).Seconds())
	}()

	//Checks startingFromNode
	if startingFromNode != nil {

		start := w.Workflow.GetNode(*startingFromNode)
		if start == nil {
			return sdk.ErrWorkflowNodeNotFound
		}
		//Run the node : manual or from an event
		log.Debug("processWorkflowRun> starting from node %#v", startingFromNode)
		if err := processWorkflowNodeRun(db, w, start, len(w.WorkflowNodeRuns), nil, hookEvent, manual); err != nil {
			return sdk.WrapError(err, "processWorkflowRun> Unable to process workflow node run")
		}
		return nil
	}

	//Checks the root
	if len(w.WorkflowNodeRuns) == 0 {
		log.Debug("processWorkflowRun> starting from the root : %#v", w.Workflow.Root)
		//Run the root: manual or from an event
		if err := processWorkflowNodeRun(db, w, w.Workflow.Root, 0, nil, hookEvent, manual); err != nil {
			return sdk.WrapError(err, "processWorkflowRun> Unable to process workflow node run")
		}
		return nil
	}

	//Checks the triggers
	for i := range w.WorkflowNodeRuns {
		nodeRun := &w.WorkflowNodeRuns[i]
		//Trigger only if the node is over (successfull or not)
		if nodeRun.Status == string(sdk.StatusSuccess) || nodeRun.Status == string(sdk.StatusFail) {
			//Find the node in the workflow
			node := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
			if node == nil {
				return sdk.ErrWorkflowNodeNotFound
			}
			for j := range node.Triggers {
				t := &node.Triggers[j]
				//TODO Check conditions

				//Keep the subnumber of the previous node in the graph
				log.Debug("processWorkflowRun> starting from trigger %#v", t)
				if err := processWorkflowNodeRun(db, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), []int64{nodeRun.ID}, nil, nil); err != nil {
					sdk.WrapError(err, "processWorkflowRun> Unable to process node ID=%d", t.WorkflowDestNode.ID)
				}
			}
		}
	}

	//Checks the joins
	for i := range w.Workflow.Joins {
		j := &w.Workflow.Joins[i]
		sources := map[int64]*sdk.WorkflowNodeRun{}

		//we have to check noderun for every sources
		for _, id := range j.SourceNodeIDs {
			sources[id] = nil
			for x := range w.WorkflowNodeRuns {
				nodeRun := &w.WorkflowNodeRuns[x]
				if nodeRun.WorkflowNodeID == id {
					//We found the source in the list of the noderuns
					sources[id] = nodeRun
				}
			}
		}

		//now checks if all sources have been completed
		var ok = true
		nodeRunIDs := []int64{}
		for _, nodeRun := range sources {
			if nodeRun == nil {
				//One of the sources have not been started
				ok = false
				break
			}
			if nodeRun.Status != string(sdk.StatusSuccess) && nodeRun.Status != string(sdk.StatusFail) {
				//One of the sources have not been completed
				ok = false
				break
			}
			nodeRunIDs = append(nodeRunIDs, nodeRun.ID)
		}

		if ok {
			//Keep a ref to the sources
			nodeRun := sources[j.SourceNodeIDs[0]]
			if nodeRun == nil {
				return fmt.Errorf("This should not append...")
			}

			//All the sources are completed
			//Checks the triggers
			for x := range j.Triggers {
				t := &j.Triggers[x]
				//TODO Check conditions

				//Keep the subnumber of the previous node in the graph
				if err := processWorkflowNodeRun(db, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), nodeRunIDs, nil, nil); err != nil {
					sdk.WrapError(err, "processWorkflowRun> Unable to process node ID=%d", t.WorkflowDestNode.ID)
				}
			}
		}

	}

	if err := updateWorkflowRun(db, w); err != nil {
		sdk.WrapError(err, "processWorkflowRun>")
	}

	return nil
}

func processWorkflowNodeRun(db gorp.SqlExecutor, w *sdk.WorkflowRun, n *sdk.WorkflowNode, subnumber int, sourceNodeRuns []int64, h *sdk.WorkflowNodeRunHookEvent, m *sdk.WorkflowNodeRunManual) error {
	t0 := time.Now()
	log.Debug("processWorkflowNodeRun> Begin [#%d.%d]%s.%d", w.Number, subnumber, w.Workflow.Name, n.ID)
	defer func() {
		log.Debug("processWorkflowNodeRun> End [#%d.%d]%s.%d  - %.3fs", w.Number, subnumber, w.Workflow.Name, n.ID, time.Since(t0).Seconds())
	}()

	run := &sdk.WorkflowNodeRun{
		LastModified:   time.Now(),
		Start:          time.Now(),
		Number:         w.Number,
		SubNumber:      int64(subnumber),
		WorkflowRunID:  w.ID,
		WorkflowNodeID: n.ID,
		Status:         string(sdk.StatusWaiting),
		Stages:         n.Pipeline.Stages,
	}

	for i := range run.Stages {
		run.Stages[i].Status = sdk.StatusWaiting
	}

	run.SourceNodeRuns = sourceNodeRuns
	if sourceNodeRuns != nil {
		//Get all the nodeRun from the sources
		//Merge the payload applying older nodeRun to most recent
		runs := []sdk.WorkflowNodeRun{}
		for _, id := range sourceNodeRuns {
			for _, runID := range w.WorkflowNodeRuns {
				if id == runID.ID {
					runs = append(runs, runID)
				}
			}
		}

		sort.Slice(runs, func(i, j int) bool {
			return runs[i].Start.Before(runs[i].Start)
		})

		m := map[string]string{}
		for _, r := range runs {
			m1 := sdk.ParametersToMap(r.Payload)
			for k, v := range m1 {
				m[k] = v
			}
		}

		run.Payload = sdk.ParametersFromMap(m)
		run.PipelineParameter = n.Context.DefaultPipelineParameters
	}

	run.HookEvent = h
	if h != nil {
		payload, err := dump.ToMap(h.Payload, dump.WithDefaultLowerCaseFormatter())
		if err != nil {
			log.Error("processWorkflowNodeRun> Unable to compute hook payload")
		}
		run.Payload = sdk.ParametersFromMap(payload)
		if len(h.PipelineParameters) != 0 {
			run.PipelineParameter = h.PipelineParameters
		} else {
			run.PipelineParameter = n.Context.DefaultPipelineParameters
		}

	}

	run.Manual = m
	if m != nil {
		payload, err := dump.ToMap(m.Payload, dump.WithDefaultLowerCaseFormatter())
		if err != nil {
			log.Error("processWorkflowNodeRun> Unable to compute hook payload")
		}
		run.Payload = sdk.ParametersFromMap(payload)
		if len(m.PipelineParameters) != 0 {
			run.PipelineParameter = m.PipelineParameters
		} else {
			run.PipelineParameter = n.Context.DefaultPipelineParameters
		}
	}

	if err := insertWorkflowNodeRun(db, run); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to insert run")
	}

	log.Debug("processWorkflowNodeRun> new node run: %#v", run)

	w.WorkflowNodeRuns = append(w.WorkflowNodeRuns, *run)
	if err := updateWorkflowRun(db, w); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to update workflow run")
	}

	//Push the workflow node run in queue
	cache.Enqueue(queueWorkflowNodeRun, run)

	return nil
}
