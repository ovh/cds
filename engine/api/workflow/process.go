package workflow

import (
	"bytes"
	"fmt"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/luascript"
)

// processWorkflowRun triggers workflow node for every workflow.
// It contains all the logic for triggers and joins processing.
func processWorkflowRun(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual, startingFromNode *int64) error {
	var nodesRunFailed, nodesRunStopped, nodesRunBuilding, nodesRunSuccess int
	t0 := time.Now()
	w.Status = string(sdk.StatusBuilding)
	log.Debug("processWorkflowRun> Begin [#%d]%s", w.Number, w.Workflow.Name)
	defer func() {
		log.Debug("processWorkflowRun> End [#%d]%s - %.3fs", w.Number, w.Workflow.Name, time.Since(t0).Seconds())
	}()

	maxsn := MaxSubNumber(w.WorkflowNodeRuns)
	log.Info("processWorkflowRun> %d.%d", w.Number, maxsn)
	w.LastSubNumber = maxsn

	//Checks startingFromNode
	if startingFromNode != nil {
		start := w.Workflow.GetNode(*startingFromNode)
		if start == nil {
			return sdk.ErrWorkflowNodeNotFound
		}
		//Run the node : manual or from an event
		nextSubNumber := maxsn
		nodeRuns, ok := w.WorkflowNodeRuns[*startingFromNode]
		if ok && len(nodeRuns) > 0 {
			nextSubNumber++
		}

		log.Debug("processWorkflowRun> starting from node %#v", startingFromNode)
		// Find ancestors
		nodeIds := start.Ancestors(&w.Workflow, false)
		sourceNodesRunID := make([]int64, len(nodeIds))
		for i := range nodeIds {
			nodesRuns, ok := w.WorkflowNodeRuns[nodeIds[i]]
			if ok && len(nodesRuns) > 0 {
				sourceNodesRunID[i] = nodesRuns[0].ID
			} else {
				return sdk.ErrWorkflowNodeParentNotRun
			}
		}
		if err := processWorkflowNodeRun(db, store, p, w, start, int(nextSubNumber), sourceNodesRunID, hookEvent, manual); err != nil {
			return sdk.WrapError(err, "processWorkflowRun> Unable to process workflow node run")
		}
		return nil
	}

	//Checks the root
	if len(w.WorkflowNodeRuns) == 0 {
		log.Debug("processWorkflowRun> starting from the root : %d (pipeline %s)", w.Workflow.Root.ID, w.Workflow.Root.Pipeline.Name)
		//Run the root: manual or from an event
		AddWorkflowRunInfo(w, false, sdk.SpawnMsg{
			ID: sdk.MsgWorkflowStarting.ID,
			Args: []interface{}{
				w.Workflow.Name,
				fmt.Sprintf("%d.%d", w.Number, 0),
			},
		})

		if err := processWorkflowNodeRun(db, store, p, w, w.Workflow.Root, 0, nil, hookEvent, manual); err != nil {
			return sdk.WrapError(err, "processWorkflowRun> Unable to process workflow node run")
		}
		return nil
	}

	//Checks the triggers
	for k, v := range w.WorkflowNodeRuns {
		lastCurrentSn := lastSubNumber(w.WorkflowNodeRuns[k])
		// Subversion of workflowNodeRun
		for i := range v {
			nodeRun := &w.WorkflowNodeRuns[k][i]

			haveToUpdate := false

			log.Debug("last current sub number %v nodeRun version %v.%v and status %v", lastCurrentSn, nodeRun.Number, nodeRun.SubNumber, nodeRun.Status)
			// Only the last subversion
			if lastCurrentSn == nodeRun.SubNumber {
				updateNodesRunStatus(nodeRun.Status, &nodesRunSuccess, &nodesRunBuilding, &nodesRunFailed, &nodesRunStopped)
			}

			//Trigger only if the node is over (successfull or not)
			if nodeRun.Status == sdk.StatusSuccess.String() || nodeRun.Status == sdk.StatusFail.String() {
				//Find the node in the workflow
				node := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
				if node == nil {
					return sdk.ErrWorkflowNodeNotFound
				}
				for j := range node.Triggers {
					t := &node.Triggers[j]

					if t.Manual {
						continue
					}

					if !t.ContinueOnError && nodeRun.Status == sdk.StatusFail.String() {
						continue
					}

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

					var conditionsOK bool
					var errc error
					if t.Conditions.LuaScript == "" {
						conditionsOK, errc = sdk.WorkflowCheckConditions(t.Conditions.PlainConditions, params)
					} else {
						luacheck := luascript.NewCheck()
						luacheck.SetVariables(sdk.ParametersToMap(params))
						errc = luacheck.Perform(t.Conditions.LuaScript)
						conditionsOK = luacheck.Result
					}

					if errc != nil {
						log.Warning("processWorkflowRun> WorkflowCheckConditions error: %s", errc)
						AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
							ID:   sdk.MsgWorkflowError.ID,
							Args: []interface{}{errc},
						})
					}

					if !conditionsOK {
						if nodeRun.TriggersRun == nil {
							nodeRun.TriggersRun = make(map[int64]sdk.WorkflowNodeTriggerRun)
						}
						wntr, ok := nodeRun.TriggersRun[t.ID]
						if !ok || wntr.Status != sdk.StatusFail.String() {
							triggerRun := sdk.WorkflowNodeTriggerRun{
								Status:             sdk.StatusFail.String(),
								WorkflowDestNodeID: t.WorkflowDestNode.ID,
							}
							nodeRun.TriggersRun[t.ID] = triggerRun
							haveToUpdate = true
						}

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
						if nodeRun.TriggersRun == nil {
							nodeRun.TriggersRun = make(map[int64]sdk.WorkflowNodeTriggerRun)
						}
						triggerStatus := sdk.StatusSuccess.String()
						if t.ContinueOnError && nodeRun.Status == sdk.StatusFail.String() {
							triggerStatus = sdk.StatusWarning.String()
						}
						triggerRun := sdk.WorkflowNodeTriggerRun{
							Status:             triggerStatus,
							WorkflowDestNodeID: t.WorkflowDestNode.ID,
						}
						nodeRun.TriggersRun[t.ID] = triggerRun
						haveToUpdate = true
						//Keep the subnumber of the previous node in the graph
						if err := processWorkflowNodeRun(db, store, p, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), []int64{nodeRun.ID}, nil, nil); err != nil {
							log.Error("processWorkflowRun> Unable to process node ID=%d: %s", t.WorkflowDestNode.ID, err)
							AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
								ID:   sdk.MsgWorkflowError.ID,
								Args: []interface{}{err},
							})
						} else {
							nodesRunBuilding++
						}
					}
				}
			}

			if haveToUpdate {
				if err := UpdateNodeRun(db, nodeRun); err != nil {
					return sdk.WrapError(err, "process> Cannot update node run")
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
						if sources[id] == nil {
							sources[id] = nodeRun
							continue
						}
						//We found the source in the list of the noderuns
						if sources[id].SubNumber < nodeRun.SubNumber {
							sources[id] = nodeRun
						}
					}
				}
			}
		}

		//now checks if all sources have been completed
		var ok = true
		nodeRunIDs := []int64{}
		sourcesParams := map[string]string{}
		sourcesFail := 0
		for _, nodeRun := range sources {
			if nodeRun == nil {
				ok = false
				break
			}

			log.Debug("Checking source %s (#%d.%d) status = %s", w.Workflow.GetNode(nodeRun.WorkflowNodeID).Name, nodeRun.Number, nodeRun.SubNumber, nodeRun.Status)

			if (nodeRun.Status != string(sdk.StatusSuccess) && nodeRun.Status != string(sdk.StatusFail)) || nodeRun.SubNumber < maxsn {
				//One of the sources have not been completed
				ok = false
				break
			}

			if nodeRun.Status == sdk.StatusFail.String() {
				sourcesFail++
			}

			nodeRunIDs = append(nodeRunIDs, nodeRun.ID)
			//Merge build parameters from all sources
			sourcesParams = sdk.ParametersMapMerge(sourcesParams, sdk.ParametersToMap(nodeRun.BuildParameters))
		}

		//All the sources are completed
		if ok {
			//Checks the triggers
			for x := range j.Triggers {
				t := &j.Triggers[x]
				if t.Manual {
					continue
				}

				if !t.ContinueOnError && sourcesFail > 0 {
					continue
				}

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

				var errc error
				var conditionsOK bool
				if t.Conditions.LuaScript == "" {
					conditionsOK, errc = sdk.WorkflowCheckConditions(t.Conditions.PlainConditions, params)
				} else {
					luacheck := luascript.NewCheck()
					luacheck.SetVariables(sdk.ParametersToMap(params))
					errc = luacheck.Perform(t.Conditions.LuaScript)
					conditionsOK = luacheck.Result
				}

				if errc != nil {
					AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
						ID:   sdk.MsgWorkflowError.ID,
						Args: []interface{}{errc},
					})
				}
				//If conditions are not met, skip this trigger
				if !conditionsOK {
					if w.JoinTriggersRun == nil {
						w.JoinTriggersRun = make(map[int64]sdk.WorkflowNodeTriggerRun)
					}
					wntr, ok := w.JoinTriggersRun[t.ID]
					if !ok || wntr.Status != sdk.StatusFail.String() {
						triggerRun := sdk.WorkflowNodeTriggerRun{
							Status:             sdk.StatusFail.String(),
							WorkflowDestNodeID: t.WorkflowDestNode.ID,
						}
						w.JoinTriggersRun[t.ID] = triggerRun
					}
					continue
				}

				// check if the destination node already exists on w.WorkflowNodeRuns with the same subnumber
				var abortTrigger bool
			previousJoinRuns:
				for _, previousRunArray := range w.WorkflowNodeRuns {
					for _, previousRun := range previousRunArray {
						if previousRun.WorkflowNodeID == t.WorkflowDestNode.ID && previousRun.SubNumber == maxsn {
							abortTrigger = true
							break previousJoinRuns
						}
					}
				}

				if !abortTrigger {
					if w.JoinTriggersRun == nil {
						w.JoinTriggersRun = make(map[int64]sdk.WorkflowNodeTriggerRun)
					}
					triggerStatus := sdk.StatusSuccess.String()
					if t.ContinueOnError && sourcesFail > 0 {
						triggerStatus = sdk.StatusWarning.String()
					}
					triggerRun := sdk.WorkflowNodeTriggerRun{
						Status:             triggerStatus,
						WorkflowDestNodeID: t.WorkflowDestNode.ID,
					}
					w.JoinTriggersRun[t.ID] = triggerRun

					//Keep the subnumber of the previous node in the graph
					if err := processWorkflowNodeRun(db, store, p, w, &t.WorkflowDestNode, int(maxsn), nodeRunIDs, nil, nil); err != nil {
						AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
							ID:   sdk.MsgWorkflowError.ID,
							Args: []interface{}{err},
						})
						log.Error("processWorkflowRun> Unable to process node ID=%d: %v", t.WorkflowDestNode.ID, err)
					} else {
						nodesRunBuilding++
					}
				}
			}
		}
	}

	w.Status = getWorkflowRunStatus(nodesRunSuccess, nodesRunBuilding, nodesRunFailed, nodesRunStopped)
	if err := updateWorkflowRun(db, w); err != nil {
		return sdk.WrapError(err, "processWorkflowRun>")
	}

	return nil
}

//processWorkflowNodeRun triggers execution of a node run
func processWorkflowNodeRun(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.WorkflowRun, n *sdk.WorkflowNode, subnumber int, sourceNodeRuns []int64, h *sdk.WorkflowNodeRunHookEvent, m *sdk.WorkflowNodeRunManual) error {
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
			e := dump.NewDefaultEncoder(new(bytes.Buffer))
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false
			m1, errm1 := e.ToStringMap(r.Payload)
			if errm1 != nil {
				AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
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
		run.PipelineParameters = n.Context.DefaultPipelineParameters
	}

	run.Manual = m
	if m != nil {
		m1, errm1 := dump.ToMap(m.Payload, dump.WithDefaultLowerCaseFormatter())
		if errm1 != nil {
			AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{errm1},
			})
			return sdk.WrapError(errm1, "processWorkflowNodeRun> Unable to compute payload")
		}
		run.Payload = m1
		run.PipelineParameters = m.PipelineParameters
		run.BuildParameters = append(run.BuildParameters, sdk.Parameter{
			Name:  "cds.triggered_by.email",
			Type:  sdk.StringParameter,
			Value: m.User.Email,
		}, sdk.Parameter{
			Name:  "cds.triggered_by.fullname",
			Type:  sdk.StringParameter,
			Value: m.User.Fullname,
		}, sdk.Parameter{
			Name:  "cds.triggered_by.username",
			Type:  sdk.StringParameter,
			Value: m.User.Username,
		})
	}

	// Process parameters for the jobs
	jobParams, errParam := getNodeRunBuildParameters(db, p, run)
	if errParam != nil {
		AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errParam},
		})
		// if there an error -> display it in workflowRunInfo and not stop the launch
		log.Error("processWorkflowNodeRun> getNodeRunBuildParameters failed. Project:%s Begin [#%d.%d]%s.%d err:%s", p.Name, w.Number, subnumber, w.Workflow.Name, n.ID, errParam)
	}
	run.BuildParameters = append(run.BuildParameters, jobParams...)

	// Inherit parameter from parent job
	if len(sourceNodeRuns) > 0 {
		parentsParams, errPP := getParentParameters(db, run, sourceNodeRuns)
		if errPP != nil {
			return sdk.WrapError(errPP, "processWorkflowNodeRun> getParentParameters failed")
		}
		mapBuildParams := sdk.ParametersToMap(run.BuildParameters)
		mapParentParams := sdk.ParametersToMap(parentsParams)

		run.BuildParameters = sdk.ParametersFromMap(sdk.ParametersMapMerge(mapBuildParams, mapParentParams))
	}
	for _, p := range jobParams {
		switch p.Name {
		case tagGitHash, tagGitBranch, tagGitTag, tagGitAuthor:
			w.Tag(p.Name, p.Value)
		}
	}

	// Add env tag
	if n.Context != nil && n.Context.Environment != nil {
		w.Tag(tagEnvironment, n.Context.Environment.Name)
	}

	//Check
	if h != nil {
		hooks := w.Workflow.GetHooks()
		hook, ok := hooks[h.WorkflowNodeHookUUID]
		if !ok {
			return sdk.WrapError(sdk.ErrNoHook, "processWorkflowNodeRun> Unable to find hook %s", h.WorkflowNodeHookUUID)
		}

		//Check conditions
		var params = run.BuildParameters
		//Define specific destination parameters
		dest := w.Workflow.GetNode(hook.WorkflowNodeID)
		if dest == nil {
			return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "processWorkflowNodeRun> Unable to find node %d", hook.WorkflowNodeID)
		}
		sdk.AddParameter(&params, "cds.dest.pipeline", sdk.StringParameter, dest.Pipeline.Name)
		if dest.Context.Application != nil {
			sdk.AddParameter(&params, "cds.dest.application", sdk.StringParameter, dest.Context.Application.Name)
		}
		if dest.Context.Environment != nil {
			sdk.AddParameter(&params, "cds.dest.environment", sdk.StringParameter, dest.Context.Environment.Name)
		}

		var conditionsOK bool
		var errc error
		if hook.Conditions.LuaScript == "" {
			conditionsOK, errc = sdk.WorkflowCheckConditions(hook.Conditions.PlainConditions, params)
		} else {
			luacheck := luascript.NewCheck()
			luacheck.SetVariables(sdk.ParametersToMap(params))
			errc = luacheck.Perform(hook.Conditions.LuaScript)
			conditionsOK = luacheck.Result
		}

		if errc != nil {
			log.Warning("processWorkflowNodeRun> WorkflowCheckConditions error: %s", errc)
			AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{errc},
			})
		}
		if !conditionsOK {
			log.Info("processWorkflowNodeRun> Avoid trigger workflow from hook %s", hook.UUID)
			return nil
		}
	}

	for _, info := range w.Infos {
		if info.IsError {
			run.Status = string(sdk.StatusFail)
			break
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
	w.LastSubNumber = MaxSubNumber(w.WorkflowNodeRuns)
	if err := updateWorkflowRun(db, w); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to update workflow run")
	}

	//Execute the node run !
	if err := execute(db, store, p, run); err != nil {
		return sdk.WrapError(err, "processWorkflowNodeRun> unable to execute workflow run")
	}

	return nil
}

// AddWorkflowRunInfo add WorkflowRunInfo on a WorkflowRun
func AddWorkflowRunInfo(run *sdk.WorkflowRun, isError bool, infos ...sdk.SpawnMsg) {
	for _, i := range infos {
		run.Infos = append(run.Infos, sdk.WorkflowRunInfo{
			APITime: time.Now(),
			Message: i,
			IsError: isError,
		})
	}
}

// getWorkflowRunStatus return the status depending on number of workflowNodeRuns in success, building, stopped and fail
func getWorkflowRunStatus(nodesRunSuccess, nodesRunBuilding, nodesRunFailed, nodesRunStopped int) string {
	switch {
	case nodesRunBuilding > 0:
		return string(sdk.StatusBuilding)
	case nodesRunFailed > 0:
		return string(sdk.StatusFail)
	case nodesRunStopped > 0:
		return string(sdk.StatusStopped)
	case nodesRunSuccess > 0:
		return string(sdk.StatusSuccess)
	default:
		return string(sdk.StatusNeverBuilt)
	}
}

// updateNodesRunStatus is useful to compute number of nodeRun in success, building and fail
func updateNodesRunStatus(status string, success, building, fail, stop *int) {
	switch status {
	case string(sdk.StatusSuccess):
		*success++
	case string(sdk.StatusBuilding), string(sdk.StatusWaiting):
		*building++
	case string(sdk.StatusFail):
		*fail++
	case string(sdk.StatusStopped):
		*stop++
	}
}

// MaxSubNumber returns the MaxSubNumber of workflowNodeRuns
func MaxSubNumber(workflowNodeRuns map[int64][]sdk.WorkflowNodeRun) int64 {
	var maxsn int64
	for _, wNodeRuns := range workflowNodeRuns {
		for _, wNodeRun := range wNodeRuns {
			if maxsn < wNodeRun.SubNumber {
				maxsn = wNodeRun.SubNumber
			}
		}
	}

	return maxsn
}

func lastSubNumber(workflowNodeRuns []sdk.WorkflowNodeRun) int64 {
	var lastSn int64
	for _, wNodeRun := range workflowNodeRuns {
		if lastSn < wNodeRun.SubNumber {
			lastSn = wNodeRun.SubNumber
		}
	}

	return lastSn
}
