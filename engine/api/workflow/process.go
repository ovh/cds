package workflow

import (
	"fmt"
	"sort"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

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
	for k, v := range w.WorkflowNodeRuns {
		for i := range v {
			nodeRun := &w.WorkflowNodeRuns[k][i]

			//Trigger only if the node is over (successfull or not)
			if nodeRun.Status == string(sdk.StatusSuccess) || nodeRun.Status == string(sdk.StatusFail) {
				//Find the node in the workflow
				node := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
				if node == nil {
					return sdk.ErrWorkflowNodeNotFound
				}
				for j := range node.Triggers {
					t := &node.Triggers[j]

					//Check conditions
					var params = nodeRun.BuildParameters
					//Define specific desitination parameters
					sdk.AddParameter(&params, "cds.dest.pip", sdk.StringParameter, t.WorkflowDestNode.Pipeline.Name)
					if t.WorkflowDestNode.Context.Application != nil {
						sdk.AddParameter(&params, "cds.dest.app", sdk.StringParameter, t.WorkflowDestNode.Context.Application.Name)
					}
					if t.WorkflowDestNode.Context.Environment != nil {
						sdk.AddParameter(&params, "cds.dest.env", sdk.StringParameter, t.WorkflowDestNode.Context.Environment.Name)
					}

					conditionsOK, err := sdk.WorkflowCheckConditions(t.Conditions, params)
					if err != nil {
						//TODO do something like spawn info on  workflow run
						return err
					}

					if !conditionsOK {
						continue
					}

					// check if the destination node already exists on w.WorkflowNodeRuns with the same subnumber
					var abortTrigger bool
				previousRuns:
					for _, previousRunArray := range w.WorkflowNodeRuns {
						for _, previousRun := range previousRunArray {
							if previousRun.WorkflowNodeID == t.WorkflowDestNode.ID && previousRun.SubNumber == nodeRun.SubNumber {
								abortTrigger = true
								break previousRuns
							}
						}
					}

					if !abortTrigger {
						//Keep the subnumber of the previous node in the graph
						log.Debug("processWorkflowRun> starting from trigger %#v", t)
						if err := processWorkflowNodeRun(db, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), []int64{nodeRun.ID}, nil, nil); err != nil {
							sdk.WrapError(err, "processWorkflowRun> Unable to process node ID=%d", t.WorkflowDestNode.ID)
						}
					}
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
			for k, v := range w.WorkflowNodeRuns {
				for x := range v {
					nodeRun := &w.WorkflowNodeRuns[k][x]
					if nodeRun.WorkflowNodeID == id {
						//We found the source in the list of the noderuns
						sources[id] = nodeRun
					}
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

				// check if the destination node already exists on w.WorkflowNodeRuns with the same subnumber
				var abortTrigger bool
			previousJoinRuns:
				for _, previousRunArray := range w.WorkflowNodeRuns {
					for _, previousRun := range previousRunArray {
						if previousRun.WorkflowNodeID == t.WorkflowDestNode.ID && previousRun.SubNumber == nodeRun.SubNumber {
							abortTrigger = true
							break previousJoinRuns
						}
					}
				}

				if !abortTrigger {
					//Keep the subnumber of the previous node in the graph
					if err := processWorkflowNodeRun(db, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), nodeRunIDs, nil, nil); err != nil {
						sdk.WrapError(err, "processWorkflowRun> Unable to process node ID=%d", t.WorkflowDestNode.ID)
					}
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

	//Process parameters for the jobs
	jobParams, errParam := getNodeRunParameters(db, run)
	if errParam != nil {
		return errParam
	}
	run.BuildParameters = jobParams

	run.SourceNodeRuns = sourceNodeRuns
	if sourceNodeRuns != nil {
		//Get all the nodeRun from the sources
		//Merge the payload applying older nodeRun to most recent
		runs := []sdk.WorkflowNodeRun{}
		for _, id := range sourceNodeRuns {
			for _, v := range w.WorkflowNodeRuns {
				for _, run := range v {
					if id == run.ID {
						runs = append(runs, run)
					}
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
		run.PipelineParameters = n.Context.DefaultPipelineParameters
	}

	run.HookEvent = h
	if h != nil {
		payload, err := dump.ToMap(h.Payload, dump.WithDefaultLowerCaseFormatter())
		if err != nil {
			log.Error("processWorkflowNodeRun> Unable to compute hook payload")
		}
		run.Payload = sdk.ParametersFromMap(payload)
		if len(h.PipelineParameters) != 0 {
			run.PipelineParameters = h.PipelineParameters
		} else {
			run.PipelineParameters = n.Context.DefaultPipelineParameters
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
			run.PipelineParameters = m.PipelineParameters
		} else {
			run.PipelineParameters = n.Context.DefaultPipelineParameters
		}
	}

	if err := insertWorkflowNodeRun(db, run); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to insert run")
	}

	log.Debug("processWorkflowNodeRun> new node run: %#v", run)

	if w.WorkflowNodeRuns == nil {
		w.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}
	w.WorkflowNodeRuns[run.WorkflowNodeID] = append(w.WorkflowNodeRuns[run.WorkflowNodeID], *run)
	if err := updateWorkflowRun(db, w); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to update workflow run")
	}

	if err := execute(db, run); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to execute workflow run")
	}

	return nil
}
