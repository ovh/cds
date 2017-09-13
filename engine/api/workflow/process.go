package workflow

import (
	"fmt"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// processWorkflowRun triggers workflow node for every workflow.
// It contains all the logic for triggers and joins processing.
func processWorkflowRun(db gorp.SqlExecutor, store cache.Store, w *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual, startingFromNode *int64) error {
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
		if err := processWorkflowNodeRun(db, store, w, start, len(w.WorkflowNodeRuns), nil, hookEvent, manual); err != nil {
			return sdk.WrapError(err, "processWorkflowRun> Unable to process workflow node run")
		}
		return nil
	}

	//Checks the root
	if len(w.WorkflowNodeRuns) == 0 {
		log.Debug("processWorkflowRun> starting from the root : %d (pipeline %s)", w.Workflow.Root.ID, w.Workflow.Root.Pipeline.Name)
		//Run the root: manual or from an event
		AddWorkflowRunInfo(w, sdk.SpawnMsg{
			ID: sdk.MsgWorkflowStarting.ID,
			Args: []interface{}{
				w.Workflow.Name,
				fmt.Sprintf("%d.%d", w.Number, 0),
			},
		})

		if err := processWorkflowNodeRun(db, store, w, w.Workflow.Root, 0, nil, hookEvent, manual); err != nil {
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
					//Define specific destination parameters
					sdk.AddParameter(&params, "cds.dest.pipeline", sdk.StringParameter, t.WorkflowDestNode.Pipeline.Name)
					if t.WorkflowDestNode.Context.Application != nil {
						sdk.AddParameter(&params, "cds.dest.application", sdk.StringParameter, t.WorkflowDestNode.Context.Application.Name)
					}
					if t.WorkflowDestNode.Context.Environment != nil {
						sdk.AddParameter(&params, "cds.dest.environment", sdk.StringParameter, t.WorkflowDestNode.Context.Environment.Name)
					}

					conditionsOK, errc := sdk.WorkflowCheckConditions(t.Conditions, params)
					if errc != nil {
						log.Warning("processWorkflowRun> WorkflowCheckConditions error: %s", errc)
						AddWorkflowRunInfo(w, sdk.SpawnMsg{
							ID:   sdk.MsgWorkflowError.ID,
							Args: []interface{}{errc},
						})
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
						if err := processWorkflowNodeRun(db, store, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), []int64{nodeRun.ID}, nil, nil); err != nil {
							log.Error("processWorkflowRun> Unable to process node ID=%d: %s", t.WorkflowDestNode.ID, err)
							AddWorkflowRunInfo(w, sdk.SpawnMsg{
								ID:   sdk.MsgWorkflowError.ID,
								Args: []interface{}{err},
							})
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
		sourcesParams := map[string]string{}
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
			//Merge build parameters from all sources
			sourcesParams = sdk.ParametersMapMerge(sourcesParams, sdk.ParametersToMap(nodeRun.BuildParameters))
		}

		//All the sources are completed
		if ok {
			//Keep a ref to the sources
			nodeRun := sources[j.SourceNodeIDs[0]]
			if nodeRun == nil {
				return fmt.Errorf("processWorkflowRun> this should not append %#v", w)
			}

			//All the sources are completed
			//Checks the triggers
			for x := range j.Triggers {
				t := &j.Triggers[x]

				//Check conditions
				params := sdk.ParametersFromMap(sourcesParams)
				//Define specific desitination parameters
				sdk.AddParameter(&params, "cds.dest.pipeline", sdk.StringParameter, t.WorkflowDestNode.Pipeline.Name)
				if t.WorkflowDestNode.Context.Application != nil {
					sdk.AddParameter(&params, "cds.dest.application", sdk.StringParameter, t.WorkflowDestNode.Context.Application.Name)
				}
				if t.WorkflowDestNode.Context.Environment != nil {
					sdk.AddParameter(&params, "cds.dest.environment", sdk.StringParameter, t.WorkflowDestNode.Context.Environment.Name)
				}

				conditionOK, errc := sdk.WorkflowCheckConditions(t.Conditions, params)
				if errc != nil {
					AddWorkflowRunInfo(w, sdk.SpawnMsg{
						ID:   sdk.MsgWorkflowError.ID,
						Args: []interface{}{errc},
					})
				}
				//If conditions are not met, skip this trigger
				if !conditionOK {
					continue
				}

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
					if err := processWorkflowNodeRun(db, store, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), nodeRunIDs, nil, nil); err != nil {
						AddWorkflowRunInfo(w, sdk.SpawnMsg{
							ID:   sdk.MsgWorkflowError.ID,
							Args: []interface{}{err},
						})
						log.Error("processWorkflowRun> Unable to process node ID=%d: %v", t.WorkflowDestNode.ID, err)
					}
				}
			}
		}

	}

	if err := updateWorkflowRun(db, w); err != nil {
		return sdk.WrapError(err, "processWorkflowRun>")
	}

	return nil
}

//processWorkflowNodeRun triggers execution of a node run
func processWorkflowNodeRun(db gorp.SqlExecutor, store cache.Store, w *sdk.WorkflowRun, n *sdk.WorkflowNode, subnumber int, sourceNodeRuns []int64, h *sdk.WorkflowNodeRunHookEvent, m *sdk.WorkflowNodeRunManual) error {
	t0 := time.Now()
	log.Debug("processWorkflowNodeRun> Begin [#%d.%d]%s.%d", w.Number, subnumber, w.Workflow.Name, n.ID)
	defer func() {
		log.Debug("processWorkflowNodeRun> End [#%d.%d]%s.%d  - %.3fs", w.Number, subnumber, w.Workflow.Name, n.ID, time.Since(t0).Seconds())
	}()

	//Recopy stages
	stages := make([]sdk.Stage, len(n.Pipeline.Stages))
	copy(stages, n.Pipeline.Stages)

	run := &sdk.WorkflowNodeRun{
		LastModified:   time.Now(),
		Start:          time.Now(),
		Number:         w.Number,
		SubNumber:      int64(subnumber),
		WorkflowRunID:  w.ID,
		WorkflowNodeID: n.ID,
		Status:         string(sdk.StatusWaiting),
		Stages:         stages,
	}

	run.SourceNodeRuns = sourceNodeRuns
	if sourceNodeRuns != nil {
		//Get all the nodeRun from the sources
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

		//Merge the payloads from all the sources
		m := map[string]string{}
		for _, r := range runs {
			m1, errm1 := dump.ToMap(r.Payload, dump.WithDefaultLowerCaseFormatter())
			if errm1 != nil {
				AddWorkflowRunInfo(w, sdk.SpawnMsg{
					ID:   sdk.MsgWorkflowError.ID,
					Args: []interface{}{errm1},
				})
				log.Error("processWorkflowNodeRun> Unable to compute hook payload: %v", errm1)
			}
			m = sdk.ParametersMapMerge(m, m1)
		}

		run.Payload = m
		run.PipelineParameters = n.Context.DefaultPipelineParameters
	}

	run.HookEvent = h
	if h != nil {
		run.Payload = h.Payload
		run.PipelineParameters = h.PipelineParameters
	}

	run.Manual = m
	if m != nil {
		run.Payload = m.Payload
		run.PipelineParameters = m.PipelineParameters
	}

	// Process parameters for the jobs
	jobParams, errParam := getNodeRunBuildParameters(db, store, run)
	if errParam != nil {
		AddWorkflowRunInfo(w, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errParam},
		})
		return sdk.WrapError(errParam, "processWorkflowNodeRun> getNodeRunBuildParameters failed")
	}
	run.BuildParameters = jobParams

	// Inherit parameter from parent job
	if len(sourceNodeRuns) > 0 {
		parentsParams, errPP := getParentParameters(db, run, sourceNodeRuns)
		if errPP != nil {
			return sdk.WrapError(errPP, "processWorkflowNodeRun> getParentParameters failed")
		}
		run.BuildParameters = append(run.BuildParameters, parentsParams...)
	}
	for _, p := range jobParams {
		switch p.Name {
		case "git.hash", "git.branch", "git.tag", "git.author":
			w.Tag(p.Name, p.Value)
		}
	}

	if err := insertWorkflowNodeRun(db, run); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to insert run")
	}

	//Update workflow run
	if w.WorkflowNodeRuns == nil {
		w.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}
	w.WorkflowNodeRuns[run.WorkflowNodeID] = append(w.WorkflowNodeRuns[run.WorkflowNodeID], *run)

	//Update the workflow run
	if err := updateWorkflowRun(db, w); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to update workflow run")
	}

	//Execute the node run !
	if err := execute(db, store, run); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to execute workflow run")
	}

	return nil
}

// AddWorkflowRunInfo add WorkflowRunInfo on a WorkflowRun
func AddWorkflowRunInfo(run *sdk.WorkflowRun, infos ...sdk.SpawnMsg) {
	for _, i := range infos {
		run.Infos = append(run.Infos, sdk.WorkflowRunInfo{
			APITime: time.Now(),
			Message: i,
		})
	}
}
