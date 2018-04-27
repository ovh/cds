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
func processWorkflowRun(dbCopy *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual, startingFromNode *int64, chanEvent chan<- interface{}) (bool, error) {
	var nodesRunFailed, nodesRunStopped, nodesRunBuilding, nodesRunSuccess, nodesRunSkipped, nodesRunDisabled int
	t0 := time.Now()
	log.Debug("processWorkflowRun> Begin [#%d]%s", w.Number, w.Workflow.Name)
	defer func() {
		log.Debug("processWorkflowRun> End [#%d]%s - %.3fs", w.Number, w.Workflow.Name, time.Since(t0).Seconds())
	}()
	defer func(oldStatus string, wr *sdk.WorkflowRun, chEvent chan<- interface{}) {
		if oldStatus != wr.Status && chEvent != nil {
			chEvent <- *wr
		}
	}(w.Status, w, chanEvent)

	w.Status = string(sdk.StatusBuilding)
	maxsn := MaxSubNumber(w.WorkflowNodeRuns)
	log.Debug("processWorkflowRun> %s/%s %d.%d", proj.Name, w.Workflow.Name, w.Number, maxsn)
	w.LastSubNumber = maxsn

	//Checks startingFromNode
	if startingFromNode != nil {
		start := w.Workflow.GetNode(*startingFromNode)
		if start == nil {
			return false, sdk.ErrWorkflowNodeNotFound
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
				return false, sdk.ErrWorkflowNodeParentNotRun
			}
		}
		conditionOK, errP := processWorkflowNodeRun(dbCopy, db, store, proj, w, start, int(nextSubNumber), sourceNodesRunID, hookEvent, manual, chanEvent)
		if errP != nil {
			return false, sdk.WrapError(errP, "processWorkflowRun> Unable to process workflow node run")
		}
		w.Status = sdk.StatusWaiting.String()

		return conditionOK, nil
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

		conditionOK, errP := processWorkflowNodeRun(dbCopy, db, store, proj, w, w.Workflow.Root, 0, nil, hookEvent, manual, chanEvent)
		if errP != nil {
			return false, sdk.WrapError(errP, "processWorkflowRun> Unable to process workflow node run")
		}
		return conditionOK, nil
	}

	//Checks the triggers
	for k, v := range w.WorkflowNodeRuns {
		lastCurrentSn := lastSubNumber(w.WorkflowNodeRuns[k])
		// Subversion of workflowNodeRun
		for i := range v {
			nodeRun := &w.WorkflowNodeRuns[k][i]

			haveToUpdate := false

			log.Debug("processWorkflowRun> last current sub number %v nodeRun version %v.%v and status %v", lastCurrentSn, nodeRun.Number, nodeRun.SubNumber, nodeRun.Status)
			// Only the last subversion
			if lastCurrentSn == nodeRun.SubNumber {
				computeRunStatus(nodeRun.Status, &nodesRunSuccess, &nodesRunBuilding, &nodesRunFailed, &nodesRunStopped, &nodesRunSkipped, &nodesRunDisabled)
			}

			//Trigger only if the node is over (successful or not)
			if sdk.StatusIsTerminated(nodeRun.Status) && nodeRun.Status != sdk.StatusNeverBuilt.String() {
				//Find the node in the workflow
				node := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
				if node == nil {
					return false, sdk.ErrWorkflowNodeNotFound
				}
				for j := range node.Triggers {
					t := &node.Triggers[j]

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
						conditionOk, errPwnr := processWorkflowNodeRun(dbCopy, db, store, proj, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), []int64{nodeRun.ID}, nil, nil, chanEvent)
						if errPwnr != nil {
							log.Error("processWorkflowRun> Unable to process node ID=%d: %s", t.WorkflowDestNode.ID, errPwnr)
							AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
								ID:   sdk.MsgWorkflowError.ID,
								Args: []interface{}{errPwnr.Error()},
							})
						}
						if conditionOk {
							nodesRunBuilding++
							if nodeRun.TriggersRun == nil {
								nodeRun.TriggersRun = make(map[int64]sdk.WorkflowNodeTriggerRun)
							}
							triggerStatus := sdk.StatusSuccess.String()
							triggerRun := sdk.WorkflowNodeTriggerRun{
								Status:             triggerStatus,
								WorkflowDestNodeID: t.WorkflowDestNode.ID,
							}
							nodeRun.TriggersRun[t.ID] = triggerRun
							haveToUpdate = true

						} else {
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
					}
				}
			}

			if haveToUpdate {
				if err := UpdateNodeRun(db, nodeRun); err != nil {
					return false, sdk.WrapError(err, "process> Cannot update node run")
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

			if !sdk.StatusIsTerminated(nodeRun.Status) || nodeRun.Status == sdk.StatusNeverBuilt.String() || nodeRun.Status == sdk.StatusStopped.String() || nodeRun.SubNumber < maxsn {
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
					//Keep the subnumber of the previous node in the graph
					conditionOK, err := processWorkflowNodeRun(dbCopy, db, store, proj, w, &t.WorkflowDestNode, int(maxsn), nodeRunIDs, nil, nil, chanEvent)
					if err != nil {
						AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
							ID:   sdk.MsgWorkflowError.ID,
							Args: []interface{}{err.Error()},
						})
						log.Error("processWorkflowRun> Unable to process node ID=%d: %v", t.WorkflowDestNode.ID, err)
					}
					if conditionOK {
						if w.JoinTriggersRun == nil {
							w.JoinTriggersRun = make(map[int64]sdk.WorkflowNodeTriggerRun)
						}
						triggerStatus := sdk.StatusSuccess.String()
						triggerRun := sdk.WorkflowNodeTriggerRun{
							Status:             triggerStatus,
							WorkflowDestNodeID: t.WorkflowDestNode.ID,
						}
						w.JoinTriggersRun[t.ID] = triggerRun
						nodesRunBuilding++
					} else {
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
				}
			}
		}
	}

	// Recompute status counter, it's mandatory to resync
	// the map of workflow node runs of the wornflow run to get the right statuses
	// After resync, recompute all status counter compute the workflow status
	// All of this is useful to get the right workflow status is the last node status is skipped
	if err := syncNodeRuns(db, w, LoadRunOptions{}); err != nil {
		return false, sdk.WrapError(err, "processWorkflowRun> Unable to sync workflow node runs")
	}
	// Reinit the counters
	nodesRunSuccess, nodesRunBuilding, nodesRunFailed, nodesRunStopped, nodesRunSkipped, nodesRunDisabled = 0, 0, 0, 0, 0, 0
	for k, v := range w.WorkflowNodeRuns {
		lastCurrentSn := lastSubNumber(w.WorkflowNodeRuns[k])
		// Subversion of workflowNodeRun
		for i := range v {
			nodeRun := &w.WorkflowNodeRuns[k][i]
			// Compute for the last subnumber only
			if lastCurrentSn == nodeRun.SubNumber {
				computeRunStatus(nodeRun.Status, &nodesRunSuccess, &nodesRunBuilding, &nodesRunFailed, &nodesRunStopped, &nodesRunSkipped, &nodesRunDisabled)
			}
		}
	}

	w.Status = getRunStatus(nodesRunSuccess, nodesRunBuilding, nodesRunFailed, nodesRunStopped, nodesRunSkipped, nodesRunDisabled)
	if sdk.StatusIsTerminated(w.Status) {
		w.LastExecution = time.Now()
		defer func() {
			if err := resyncCommitStatus(dbCopy, store, proj, w); err != nil {
				log.Error("processWorkflowRun> %v", err)
			}
		}()
	}
	if err := UpdateWorkflowRun(db, w); err != nil {
		return false, sdk.WrapError(err, "processWorkflowRun>")
	}

	return true, nil
}

//processWorkflowNodeRun triggers execution of a node run
func processWorkflowNodeRun(dbCopy *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.WorkflowRun, n *sdk.WorkflowNode, subnumber int, sourceNodeRuns []int64, h *sdk.WorkflowNodeRunHookEvent, m *sdk.WorkflowNodeRunManual, chanEvent chan<- interface{}) (bool, error) {
	//TODO: Check user permission
	t0 := time.Now()
	log.Debug("processWorkflowNodeRun> Begin [#%d.%d]%s.%d", w.Number, subnumber, w.Workflow.Name, n.ID)
	defer func() {
		log.Debug("processWorkflowNodeRun> End [#%d.%d]%s.%d  - %.3fs", w.Number, subnumber, w.Workflow.Name, n.ID, time.Since(t0).Seconds())
	}()

	//Recopy stages
	stages := make([]sdk.Stage, len(n.Pipeline.Stages))
	copy(stages, n.Pipeline.Stages)

	run := &sdk.WorkflowNodeRun{
		LastModified:     time.Now(),
		Start:            time.Now(),
		Number:           w.Number,
		SubNumber:        int64(subnumber),
		WorkflowRunID:    w.ID,
		WorkflowNodeID:   n.ID,
		WorkflowNodeName: n.Name,
		Status:           string(sdk.StatusWaiting),
		Stages:           stages,
	}

	runPayload := map[string]string{}

	//If the pipeline has parameter but none are defined on context, use the defaults
	if len(n.Pipeline.Parameter) > 0 && len(n.Context.DefaultPipelineParameters) == 0 {
		n.Context.DefaultPipelineParameters = n.Pipeline.Parameter
	}

	parentStatus := sdk.StatusSuccess.String()
	run.SourceNodeRuns = sourceNodeRuns
	if sourceNodeRuns != nil {
		//Get all the nodeRun from the sources
		runs := []sdk.WorkflowNodeRun{}
		for _, id := range sourceNodeRuns {
			for _, v := range w.WorkflowNodeRuns {
				for _, run := range v {
					if id == run.ID {
						runs = append(runs, run)
						if run.Status == sdk.StatusFail.String() {
							parentStatus = sdk.StatusFail.String()
						}
					}
				}
			}
		}

		//Merge the payloads from all the sources
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
					Args: []interface{}{errm1.Error()},
				})
				log.Error("processWorkflowNodeRun> Unable to compute hook payload: %v", errm1)
			}
			runPayload = sdk.ParametersMapMerge(runPayload, m1)
		}
		run.Payload = runPayload
		run.PipelineParameters = n.Context.DefaultPipelineParameters
	}

	run.HookEvent = h
	if h != nil {
		runPayload = sdk.ParametersMapMerge(runPayload, h.Payload)
		run.Payload = runPayload
		run.PipelineParameters = n.Context.DefaultPipelineParameters
	}

	run.BuildParameters = append(run.BuildParameters, sdk.Parameter{
		Name:  "cds.node",
		Type:  sdk.StringParameter,
		Value: run.WorkflowNodeName,
	})

	run.Manual = m
	if m != nil {
		e := dump.NewDefaultEncoder(new(bytes.Buffer))
		e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		e.ExtraFields.DetailedMap = false
		e.ExtraFields.DetailedStruct = false
		e.ExtraFields.Len = false
		e.ExtraFields.Type = false
		m1, errm1 := e.ToStringMap(m.Payload)
		if errm1 != nil {
			return false, sdk.WrapError(errm1, "processWorkflowNodeRun> Unable to compute payload")
		}
		runPayload = sdk.ParametersMapMerge(runPayload, m1)
		run.Payload = runPayload
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
		}, sdk.Parameter{
			Name:  "cds.manual",
			Type:  sdk.StringParameter,
			Value: "true",
		})
	} else {
		run.BuildParameters = append(run.BuildParameters, sdk.Parameter{
			Name:  "cds.manual",
			Type:  sdk.StringParameter,
			Value: "false",
		})
	}

	cdsStatusParam := sdk.Parameter{
		Name:  "cds.status",
		Type:  sdk.StringParameter,
		Value: parentStatus,
	}
	run.BuildParameters = sdk.ParametersFromMap(
		sdk.ParametersMapMerge(
			sdk.ParametersToMap(run.BuildParameters),
			sdk.ParametersToMap([]sdk.Parameter{cdsStatusParam}),
		),
	)

	// Process parameters for the jobs
	jobParams, errParam := getNodeRunBuildParameters(db, store, p, run)
	if errParam != nil {
		AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errParam.Error()},
		})
		// if there an error -> display it in workflowRunInfo and not stop the launch
		log.Error("processWorkflowNodeRun> getNodeRunBuildParameters failed. Project:%s [#%d.%d]%s.%d with payload %v err:%s", p.Name, w.Number, subnumber, w.Workflow.Name, n.ID, run.Payload, errParam)
	}
	run.BuildParameters = append(run.BuildParameters, jobParams...)

	// Inherit parameter from parent job
	if len(sourceNodeRuns) > 0 {
		parentsParams, errPP := getParentParameters(db, run, sourceNodeRuns, runPayload)
		if errPP != nil {
			return false, sdk.WrapError(errPP, "processWorkflowNodeRun> getParentParameters failed")
		}
		mapBuildParams := sdk.ParametersToMap(run.BuildParameters)
		mapParentParams := sdk.ParametersToMap(parentsParams)

		run.BuildParameters = sdk.ParametersFromMap(sdk.ParametersMapMerge(mapBuildParams, mapParentParams))
	}

	//Parse job params to get the VCS infos
	currentGitValues := map[string]string{}
	for _, param := range jobParams {
		switch param.Name {
		case tagGitHash, tagGitBranch, tagGitTag, tagGitAuthor, tagGitMessage, tagGitRepository, tagGitURL, tagGitHTTPURL:
			currentGitValues[param.Name] = param.Value
		}
	}

	//Parse job params to get the VCS infos
	previousGitValues := map[string]string{}
	for _, param := range run.BuildParameters {
		switch param.Name {
		case tagGitHash, tagGitBranch, tagGitTag, tagGitAuthor, tagGitMessage, tagGitRepository, tagGitURL, tagGitHTTPURL:
			previousGitValues[param.Name] = param.Value
		}
	}

	var isRoot bool
	if n.ID == w.Workflow.Root.ID {
		isRoot = true
	}

	gitValues := currentGitValues
	if previousGitValues[tagGitURL] == currentGitValues[tagGitURL] || previousGitValues[tagGitHTTPURL] == currentGitValues[tagGitHTTPURL] {
		gitValues = previousGitValues
	}

	vcsInfos, errVcs := getVCSInfos(db, store, p, w, gitValues, n, run, !isRoot, previousGitValues[tagGitRepository])
	if errVcs != nil {
		if isRoot {
			return false, sdk.WrapError(errVcs, "processWorkflowNodeRun> Cannot get VCSInfos")
		}
		AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errVcs.Error()},
		})
		return false, nil
	}

	run.VCSRepository = vcsInfos.repository
	run.VCSBranch = vcsInfos.branch
	run.VCSHash = vcsInfos.hash

	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitRepository, sdk.StringParameter, run.VCSRepository)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitBranch, sdk.StringParameter, run.VCSBranch)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHash, sdk.StringParameter, run.VCSHash)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitAuthor, sdk.StringParameter, vcsInfos.author)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitMessage, sdk.StringParameter, vcsInfos.message)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitURL, sdk.StringParameter, vcsInfos.url)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHTTPURL, sdk.StringParameter, vcsInfos.httpurl)

	//Check
	if h != nil {
		hooks := w.Workflow.GetHooks()
		hook, ok := hooks[h.WorkflowNodeHookUUID]
		if !ok {
			return false, sdk.WrapError(sdk.ErrNoHook, "processWorkflowNodeRun> Unable to find hook %s", h.WorkflowNodeHookUUID)
		}

		//Check conditions
		var params = run.BuildParameters
		//Define specific destination parameters
		dest := w.Workflow.GetNode(hook.WorkflowNodeID)
		if dest == nil {
			return false, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "processWorkflowNodeRun> Unable to find node %d", hook.WorkflowNodeID)
		}

		if !checkNodeRunCondition(w, *dest, params) {
			log.Debug("processWorkflowNodeRun> Avoid trigger workflow from hook %s", hook.UUID)
			return false, nil
		}
	} else {
		if !checkNodeRunCondition(w, *n, run.BuildParameters) {
			log.Debug("processWorkflowNodeRun> Condition failed %d/%d", w.ID, n.ID)
			return false, nil
		}
	}

	//Tag VCS infos
	w.Tag(tagGitRepository, run.VCSRepository)
	w.Tag(tagGitBranch, run.VCSBranch)
	if len(run.VCSHash) >= 7 {
		w.Tag(tagGitHash, run.VCSHash[:7])
	} else {
		w.Tag(tagGitHash, run.VCSHash)
	}
	w.Tag(tagGitAuthor, vcsInfos.author)

	// Add env tag
	if n.Context != nil && n.Context.Environment != nil {
		w.Tag(tagEnvironment, n.Context.Environment.Name)
	}

	for _, info := range w.Infos {
		if info.IsError {
			run.Status = string(sdk.StatusFail)
			run.Done = time.Now()
			break
		}
	}

	if err := insertWorkflowNodeRun(db, run); err != nil {
		return true, sdk.WrapError(err, "processWorkflowNodeRun> unable to insert run")
	}
	if chanEvent != nil {
		chanEvent <- *run
	}

	//Update workflow run
	if w.WorkflowNodeRuns == nil {
		w.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}
	w.WorkflowNodeRuns[run.WorkflowNodeID] = append(w.WorkflowNodeRuns[run.WorkflowNodeID], *run)
	w.LastSubNumber = MaxSubNumber(w.WorkflowNodeRuns)

	if err := UpdateWorkflowRun(db, w); err != nil {
		return true, sdk.WrapError(err, "processWorkflowNodeRun> unable to update workflow run")
	}

	//Check the context.mutex to know if we are allowed to run it
	if n.Context.Mutex {
		//Check if there are builing workflownoderun with the same workflow_node_name for the same workflow
		mutexQuery := `select count(1)
		from workflow_node_run
		join workflow_run on workflow_run.id = workflow_node_run.workflow_run_id
		join workflow on workflow.id = workflow_run.workflow_id
		where workflow.id = $1
		and workflow_node_run.id <> $2
		and workflow_node_run.workflow_node_name = $3
		and workflow_node_run.status = $4`
		nbMutex, err := db.SelectInt(mutexQuery, n.WorkflowID, run.ID, n.Name, string(sdk.StatusBuilding))
		if err != nil {
			return false, sdk.WrapError(err, "processWorkflowNodeRun> unable to check mutexes")
		}
		if nbMutex > 0 {
			log.Debug("processWorkflowNodeRun> Noderun %s processed but not executed because of mutex", n.Name)
			AddWorkflowRunInfo(w, false, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowNodeMutex.ID,
				Args: []interface{}{n.Name},
			})

			if err := UpdateWorkflowRun(db, w); err != nil {
				return true, sdk.WrapError(err, "processWorkflowNodeRun> unable to update workflow run")
			}

			//Mutex is locked. exit without error
			return true, nil
		}
		//Mutex is free, continue
	}

	//Execute the node run !
	if err := execute(dbCopy, db, store, p, run, chanEvent); err != nil {
		return true, sdk.WrapError(err, "processWorkflowNodeRun> unable to execute workflow run")
	}

	return true, nil
}

func checkNodeRunCondition(wr *sdk.WorkflowRun, node sdk.WorkflowNode, params []sdk.Parameter) bool {
	//Check conditions
	//Define specific destination parameters
	sdk.AddParameter(&params, "cds.dest.pipeline", sdk.StringParameter, node.Pipeline.Name)
	if node.Context.Application != nil {
		sdk.AddParameter(&params, "cds.dest.application", sdk.StringParameter, node.Context.Application.Name)
	}
	if node.Context.Environment != nil {
		sdk.AddParameter(&params, "cds.dest.environment", sdk.StringParameter, node.Context.Environment.Name)
	}

	var conditionsOK bool
	var errc error
	if node.Context.Conditions.LuaScript == "" {
		conditionsOK, errc = sdk.WorkflowCheckConditions(node.Context.Conditions.PlainConditions, params)
	} else {
		luacheck, err := luascript.NewCheck()
		if err != nil {
			log.Warning("processWorkflowNodeRun> WorkflowCheckConditions error: %s", err)
			AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{err.Error()},
			})
		}
		luacheck.SetVariables(sdk.ParametersToMap(params))
		errc = luacheck.Perform(node.Context.Conditions.LuaScript)
		conditionsOK = luacheck.Result
	}
	if errc != nil {
		log.Warning("processWorkflowNodeRun> WorkflowCheckConditions error: %s", errc)
		AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errc.Error()},
		})
		return false
	}
	return conditionsOK
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

// getRunStatus return the status depending on number of runs in success, building, stopped and fail
func getRunStatus(successStatus, buildingStatus, failStatus, stoppedStatus, skippedStatus, disabledStatus int) string {
	switch {
	case buildingStatus > 0:
		return sdk.StatusBuilding.String()
	case failStatus > 0:
		return sdk.StatusFail.String()
	case stoppedStatus > 0:
		return sdk.StatusStopped.String()
	case successStatus > 0:
		return sdk.StatusSuccess.String()
	case skippedStatus > 0:
		return sdk.StatusSkipped.String()
	case disabledStatus > 0:
		return sdk.StatusDisabled.String()
	default:
		return sdk.StatusNeverBuilt.String()
	}
}

// computeRunStatus is useful to compute number of runs in success, building and fail
func computeRunStatus(status string, success, building, fail, stop, skipped, disabled *int) {
	switch status {
	case sdk.StatusSuccess.String():
		*success++
	case sdk.StatusBuilding.String(), sdk.StatusWaiting.String():
		*building++
	case sdk.StatusFail.String():
		*fail++
	case sdk.StatusStopped.String():
		*stop++
	case sdk.StatusSkipped.String():
		*skipped++
	case sdk.StatusDisabled.String():
		*disabled++
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
