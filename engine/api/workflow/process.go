package workflow

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/luascript"
	"github.com/ovh/cds/sdk/tracingutils"
)

// processWorkflowRun triggers workflow node for every workflow.
// It contains all the logic for triggers and joins processing.
func processWorkflowRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual, startingFromNode *int64) (*ProcessorReport, bool, error) {
	if w.Version == 2 {
		return processWorkflowDataRun(ctx, db, store, proj, w, hookEvent, manual, startingFromNode)
	}

	// Erase workflow data with old struct for ui compatibility
	if w.Workflow.Root.ID != w.Workflow.WorkflowData.Node.ID {
		data := w.Workflow.Migrate(true)
		w.Workflow.WorkflowData = &data
	}

	var end func()
	ctx, end = observability.Span(ctx, "workflow.processWorkflowRun",
		observability.Tag(observability.TagWorkflowRun, w.Number),
		observability.Tag(observability.TagWorkflow, w.Workflow.Name),
	)
	defer end()

	if w.Header == nil {
		w.Header = sdk.WorkflowRunHeaders{}
	}
	w.Header.Set(sdk.WorkflowRunHeader, strconv.FormatInt(w.Number, 10))
	w.Header.Set(sdk.WorkflowHeader, w.Workflow.Name)
	w.Header.Set(sdk.ProjectKeyHeader, proj.Key)

	// Push data in header to allow tracing
	if observability.Current(ctx).SpanContext().IsSampled() {
		w.Header.Set(tracingutils.SampledHeader, "1")
		w.Header.Set(tracingutils.TraceIDHeader, fmt.Sprintf("%v", observability.Current(ctx).SpanContext().TraceID))
	}

	report := new(ProcessorReport)
	defer func(oldStatus string, wr *sdk.WorkflowRun) {
		if oldStatus != wr.Status {
			report.Add(*wr)
		}
	}(w.Status, w)

	w.Status = string(sdk.StatusBuilding)
	maxsn := MaxSubNumber(w.WorkflowNodeRuns)
	w.LastSubNumber = maxsn

	//Checks startingFromNode
	if startingFromNode != nil {
		start := w.Workflow.GetNode(*startingFromNode)
		if start == nil {
			return report, false, sdk.ErrWorkflowNodeNotFound
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
				return report, false, sdk.ErrWorkflowNodeParentNotRun
			}
		}
		r1, conditionOK, errP := processWorkflowNodeRun(ctx, db, store, proj, w, start, int(nextSubNumber), sourceNodesRunID, nil, hookEvent, manual)
		if errP != nil {
			return report, false, sdk.WrapError(errP, "processWorkflowRun> Unable to process workflow node run")
		}
		report, _ = report.Merge(r1, nil)
		w.Status = sdk.StatusWaiting.String()

		return report, conditionOK, nil
	}

	//Checks the root
	if len(w.WorkflowNodeRuns) == 0 {
		log.Debug("processWorkflowRun> starting from the root : %d (pipeline %s)", w.Workflow.Root.ID, w.Workflow.Root.PipelineName)
		//Run the root: manual or from an event
		AddWorkflowRunInfo(w, false, sdk.SpawnMsg{
			ID: sdk.MsgWorkflowStarting.ID,
			Args: []interface{}{
				w.Workflow.Name,
				fmt.Sprintf("%d.%d", w.Number, 0),
			},
		})

		r1, conditionOK, errP := processWorkflowNodeRun(ctx, db, store, proj, w, w.Workflow.Root, 0, nil, nil, hookEvent, manual)
		if errP != nil {
			return report, false, sdk.WrapError(errP, "processWorkflowRun> Unable to process workflow node run")
		}
		report, _ = report.Merge(r1, nil)
		return report, conditionOK, nil
	}

	//Checks the triggers
	for k, v := range w.WorkflowNodeRuns {
		// Subversion of workflowNodeRun
		for i := range v {
			nodeRun := &w.WorkflowNodeRuns[k][i]

			haveToUpdate := false
			//Trigger only if the node is over (successful or not)
			if sdk.StatusIsTerminated(nodeRun.Status) && nodeRun.Status != sdk.StatusNeverBuilt.String() {
				//Find the node in the workflow
				if nodeRun.OutgoingHook == nil {
					node := w.Workflow.GetNode(nodeRun.WorkflowNodeID)
					if node == nil {
						return report, false, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "processWorkflowRun")
					}
					for j := range node.Forks {
						r1 := processWorkflowNodeFork(ctx, db, store, proj, w, nodeRun, node.Forks[j])
						report.Merge(r1, nil) //nolint
					}

					for j := range node.Triggers {
						t := &node.Triggers[j]
						var r1 *ProcessorReport
						r1, haveToUpdate = processWorklowNodeTrigger(ctx, db, store, proj, w, nodeRun, t)
						report.Merge(r1, nil) // nolint
					}

					// Execute the outgoing hooks (asynchronously)
					for j := range node.OutgoingHooks {
						// Checks if the hooks as already been executed
						// If not instanciante trigger a task on "hooks" uService and store the execution UUID
						// Later the hooks uService will call back the API with the task execution status
						// This will reprocess all the things
						h := node.OutgoingHooks[j]
						var err error
						report, err = report.Merge(processWorkflowNodeOutgoingHook(ctx, db, store, proj, w, nodeRun, &h))
						if err != nil {
							return nil, false, sdk.WrapError(err, "process> Cannot update node run")
						}
					}

				}
			}

			if haveToUpdate {
				if err := updateNodeRunStatusAndTriggersRun(db, nodeRun); err != nil {
					return nil, false, sdk.WrapError(err, "Cannot update node run")
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
			if v, okF := w.WorkflowNodeRuns[id]; okF {
				for x := range v {
					nodeRun := &w.WorkflowNodeRuns[id][x]
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

			// check status and subnumber
			if !sdk.StatusIsTerminated(nodeRun.Status) || nodeRun.Status == sdk.StatusNeverBuilt.String() || nodeRun.Status == sdk.StatusStopped.String() || nodeRun.SubNumber < maxsn {
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
				r1 := processWorklowNodeJoinTrigger(ctx, db, store, proj, w, nodeRunIDs, maxsn, t)
				report.Merge(r1, nil) // nolint
			}
		}
	}

	// Recompute status counter, it's mandatory to resync
	// the map of workflow node runs of the workflow run to get the right statuses
	// After resync, recompute all status counter compute the workflow status
	// All of this is useful to get the right workflow status is the last node status is skipped
	_, next := observability.Span(ctx, "workflow.syncNodeRuns")
	if err := syncNodeRuns(db, w, LoadRunOptions{}); err != nil {
		next()
		return report, false, sdk.WrapError(err, "Unable to sync workflow node runs")
	}
	next()

	// Reinit the counters
	var counterStatus statusCounter
	for k, v := range w.WorkflowNodeRuns {
		lastCurrentSn := lastSubNumber(w.WorkflowNodeRuns[k])
		// Subversion of workflowNodeRun
		for i := range v {
			nodeRun := &w.WorkflowNodeRuns[k][i]
			// Compute for the last subnumber only
			if lastCurrentSn == nodeRun.SubNumber {
				computeRunStatus(nodeRun.Status, &counterStatus)
			}
		}
	}

	w.Status = getRunStatus(counterStatus)
	if err := UpdateWorkflowRun(ctx, db, w); err != nil {
		return report, false, sdk.WithStack(err)
	}

	return report, true, nil
}

func processWorklowNodeTrigger(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun, t *sdk.WorkflowNodeTrigger) (*ProcessorReport, bool) {
	report := new(ProcessorReport)
	var haveToUpdate bool

	// check if the destination node already exists on w.WorkflowNodeRuns with the same subnumber
	if previousRunArray, ok := w.WorkflowNodeRuns[t.WorkflowDestNode.ID]; ok {
		for _, previousRun := range previousRunArray {
			if previousRun.SubNumber == nodeRun.SubNumber {
				return report, false
			}
		}
	}

	//Keep the subnumber of the previous node in the graph
	r1, conditionOk, errPwnr := processWorkflowNodeRun(ctx, db, store, proj, w, &t.WorkflowDestNode, int(nodeRun.SubNumber), []int64{nodeRun.ID}, nil, nil, nil)
	if errPwnr != nil {
		log.Error("processWorklowNodeTrigger> Unable to process node ID=%d: %s", t.WorkflowDestNode.ID, errPwnr)
		AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errPwnr.Error()},
		})
	}
	report.Merge(r1, nil) // nolint

	if nodeRun.TriggersRun == nil {
		nodeRun.TriggersRun = make(map[int64]sdk.WorkflowNodeTriggerRun)
	}

	if conditionOk {
		triggerStatus := sdk.StatusSuccess.String()
		triggerRun := sdk.WorkflowNodeTriggerRun{
			Status:             triggerStatus,
			WorkflowDestNodeID: t.WorkflowDestNode.ID,
		}
		nodeRun.TriggersRun[t.ID] = triggerRun
		haveToUpdate = true
	} else {
		wntr, ok := nodeRun.TriggersRun[t.ID]
		if !ok || wntr.Status != sdk.StatusFail.String() {
			triggerRun := sdk.WorkflowNodeTriggerRun{
				Status:             sdk.StatusFail.String(),
				WorkflowDestNodeID: t.WorkflowDestNode.ID,
			}
			nodeRun.TriggersRun[t.ID] = triggerRun
			haveToUpdate = true
		}
	}

	return report, haveToUpdate
}

func processWorklowNodeJoinTrigger(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.WorkflowRun, nodeRunIDs []int64, maxsn int64, t *sdk.WorkflowNodeJoinTrigger) *ProcessorReport {
	report := new(ProcessorReport)

	// check if the destination node already exists on w.WorkflowNodeRuns with the same subnumber
	if previousRunArray, okF := w.WorkflowNodeRuns[t.WorkflowDestNode.ID]; okF {
		for _, previousRun := range previousRunArray {
			if previousRun.SubNumber == maxsn {
				return report
			}
		}
	}

	//Keep the subnumber of the previous node in the graph
	r1, _, err := processWorkflowNodeRun(ctx, db, store, proj, w, &t.WorkflowDestNode, int(maxsn), nodeRunIDs, nil, nil, nil)
	if err != nil {
		AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{err.Error()},
		})
		log.Error("processWorklowNodeJoinTrigger> Unable to process node ID=%d: %v", t.WorkflowDestNode.ID, err)
	}
	report.Merge(r1, nil) // nolint

	return report
}

func processWorklowOutgoingHookTrigger(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.WorkflowRun, subnumber int64, hookRunID string, t *sdk.WorkflowNodeOutgoingHookTrigger) *ProcessorReport {
	report := new(ProcessorReport)

	// check if the destination node already exists on w.WorkflowNodeRuns with the same subnumber
	if previousRunArray, okF := w.WorkflowNodeRuns[t.WorkflowDestNode.ID]; okF {
		for _, previousRun := range previousRunArray {
			if previousRun.SubNumber == subnumber {
				return report
			}
		}
	}

	//Keep the subnumber of the previous node in the graph
	r1, _, errPwnr := processWorkflowNodeRun(ctx, db, store, proj, w, &t.WorkflowDestNode, int(subnumber), nil, &hookRunID, nil, nil)
	if errPwnr != nil {
		log.Error("processWorklowOutgoingHookTrigger> Unable to process node ID=%d: %s", t.WorkflowDestNode.ID, errPwnr)
		AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{errPwnr.Error()},
		})
	}
	report.Merge(r1, nil) // nolint

	return report
}

func processWorkflowNodeOutgoingHook(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.WorkflowRun, nodeRun *sdk.WorkflowNodeRun, hook *sdk.WorkflowNodeOutgoingHook) (*ProcessorReport, error) {
	ctx, end := observability.Span(ctx, "workflow.processWorkflowNodeOutgoingHook")
	defer end()

	report := new(ProcessorReport)

	if w.WorkflowNodeRuns == nil {
		w.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}

	//FIX: For the moment, we trigger outgoing hooks on success
	if nodeRun.Status != sdk.StatusSuccess.String() {
		return report, nil
	}

	//Check if the WorkflowNodeOutgoingHookRun already exist with the same subnumber
	hrs, ok := w.WorkflowNodeRuns[hook.ID]
	if ok {
		var hookNodeRun *sdk.WorkflowNodeRun
		for i := range hrs {
			if hrs[i].Number == w.Number && hrs[i].SubNumber == nodeRun.SubNumber {
				hookNodeRun = &hrs[i]
				break
			}
		}
		// If the hookrun is at status terminated, let's trigger outgoing children
		if hookNodeRun != nil && !sdk.StatusIsTerminated(hookNodeRun.Status) {
			log.Debug("hook %d already processed", hook.ID)
			return nil, nil
		} else if hookNodeRun != nil && hookNodeRun.Status != sdk.StatusStopped.String() {
			log.Debug("hook %d is over, we have to reprocess al the things", hook.ID)
			for i := range hook.Triggers {
				t := &hook.Triggers[i]
				log.Debug("checking trigger %+v", t)
				r1 := processWorklowOutgoingHookTrigger(ctx, db, store, p, w, nodeRun.SubNumber, hookNodeRun.UUID, t)
				report.Merge(r1, nil) // nolint
			}
			return report, nil
		} else if hookNodeRun != nil && hookNodeRun.Status == sdk.StatusStopped.String() {
			return report, nil
		}
	}

	srvs, err := services.FindByType(db, services.TypeHooks)
	if err != nil {
		return nil, sdk.WrapError(err, "process> Cannot get hooks service")
	}

	ogHook := &sdk.NodeOutGoingHook{
		HookModelID: hook.WorkflowHookModelID,
		Config:      hook.Config,
		NodeID:      hook.ID,
	}

	if w.Workflow.OutGoingHookModels == nil {
		w.Workflow.OutGoingHookModels = make(map[int64]sdk.WorkflowHookModel)
	}
	model, has := w.Workflow.OutGoingHookModels[hook.WorkflowHookModelID]
	if !has {
		m, errM := LoadOutgoingHookModelByID(db, hook.WorkflowHookModelID)
		if errM != nil {
			return nil, sdk.WrapError(err, "process> Cannot load outgoing hook model")
		}
		model = *m
		w.Workflow.OutGoingHookModels[hook.WorkflowHookModelID] = *m
	}

	ogHook.Config[sdk.HookConfigModelName] = sdk.WorkflowNodeHookConfigValue{
		Value:        model.Name,
		Configurable: false,
		Type:         sdk.HookConfigTypeString,
	}
	ogHook.Config[sdk.HookConfigModelType] = sdk.WorkflowNodeHookConfigValue{
		Value:        model.Type,
		Configurable: false,
		Type:         sdk.HookConfigTypeString,
	}

	var hookRun = sdk.WorkflowNodeRun{
		WorkflowRunID:    w.ID,
		WorkflowNodeID:   hook.ID,
		WorkflowID:       w.Workflow.ID,
		WorkflowNodeName: hook.Name,
		OutgoingHook: &sdk.NodeOutGoingHook{
			HookModelID: hook.WorkflowHookModelID,
			Config:      hook.Config,
			NodeID:      hook.ID,
		},
		UUID:            sdk.UUID(),
		Status:          sdk.StatusWaiting.String(),
		Number:          w.Number,
		SubNumber:       nodeRun.SubNumber,
		BuildParameters: nodeRun.BuildParameters,
		Callback: &sdk.WorkflowNodeOutgoingHookRunCallback{
			Start:  time.Now(),
			Status: sdk.StatusWaiting.String(),
		},
		Start:          time.Now(),
		LastModified:   time.Now(),
		SourceNodeRuns: []int64{nodeRun.ID},
	}

	var task sdk.Task
	if _, err := services.DoJSONRequest(ctx, srvs, "POST", "/task/execute", hookRun, &task); err != nil {
		log.Warning("outgoing hook execution failed: %v", err)
		hookRun.Status = sdk.StatusFail.String()
	}

	if len(task.Executions) > 0 {
		hookRun.HookExecutionID = task.Executions[0].UUID
	}

	if err := insertWorkflowNodeRun(db, &hookRun); err != nil {
		return report, sdk.WrapError(err, "processWorkflowNodeOutgoingHook> unable to insert node run")
	}

	if w.WorkflowNodeRuns[hook.ID] == nil {
		w.WorkflowNodeRuns[hook.ID] = make([]sdk.WorkflowNodeRun, 0)
	}
	w.WorkflowNodeRuns[hook.ID] = append(w.WorkflowNodeRuns[hook.ID], hookRun)

	sort.Slice(w.WorkflowNodeRuns[hook.ID], func(i, j int) bool {
		return w.WorkflowNodeRuns[hook.ID][i].SubNumber > w.WorkflowNodeRuns[hook.ID][j].SubNumber
	})

	report.Add(hookRun)

	return report, nil
}

//processWorkflowNodeRun triggers execution of a node run
func processWorkflowNodeRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.WorkflowRun, n *sdk.WorkflowNode, subnumber int, sourceNodeRuns []int64, sourceOutgoingHookRun *string, h *sdk.WorkflowNodeRunHookEvent, m *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	exist, errN := nodeRunExist(db, n.ID, w.Number, subnumber)
	if errN != nil {
		return nil, true, sdk.WrapError(errN, "processWorkflowNodeRun> unable to check if node run exist")
	}
	if exist {
		return nil, true, nil
	}

	var end func()
	ctx, end = observability.Span(ctx, "workflow.processWorkflowNodeRun",
		observability.Tag(observability.TagWorkflow, w.Workflow.Name),
		observability.Tag(observability.TagWorkflowRun, w.Number),
		observability.Tag(observability.TagWorkflowNode, n.Name),
	)
	defer end()

	report := new(ProcessorReport)

	//TODO: Check user for manual done but check permission also for automatic trigger and hooks (with system to authenticate a webhook)

	//Recopy stages
	pip, has := w.Workflow.Pipelines[n.PipelineID]
	if !has {
		return nil, false, fmt.Errorf("pipeline %d not found in workflow", n.PipelineID)
	}
	stages := make([]sdk.Stage, len(pip.Stages))
	copy(stages, pip.Stages)

	run := &sdk.WorkflowNodeRun{
		WorkflowID:       w.WorkflowID,
		LastModified:     time.Now(),
		Start:            time.Now(),
		Number:           w.Number,
		SubNumber:        int64(subnumber),
		WorkflowRunID:    w.ID,
		WorkflowNodeID:   n.ID,
		WorkflowNodeName: n.Name,
		Status:           string(sdk.StatusWaiting),
		Stages:           stages,
		Header:           w.Header,
	}
	if run.SubNumber >= w.LastSubNumber {
		w.LastSubNumber = run.SubNumber
	}
	if n.Context != nil && n.Context.ApplicationID != 0 {
		run.ApplicationID = n.Context.ApplicationID
	} else if n.Context != nil && n.Context.Application != nil {
		run.ApplicationID = n.Context.Application.ID
	}

	runPayload := map[string]string{}

	//If the pipeline has parameter but none are defined on context, use the defaults
	if len(pip.Parameter) > 0 && len(n.Context.DefaultPipelineParameters) == 0 {
		n.Context.DefaultPipelineParameters = pip.Parameter
	}

	parentStatus := sdk.StatusSuccess.String()
	run.SourceNodeRuns = sourceNodeRuns
	runs := []*sdk.WorkflowNodeRun{}
	if sourceNodeRuns != nil {
		//Get all the nodeRun from the sources
		for _, id := range sourceNodeRuns {
			for _, v := range w.WorkflowNodeRuns {
				for _, run := range v {
					if id == run.ID {
						runs = append(runs, &run)
						if run.Status == sdk.StatusFail.String() || run.Status == sdk.StatusStopped.String() {
							parentStatus = run.Status
						}
					}
				}
			}
		}

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
		run.PipelineParameters = sdk.ParametersMerge(pip.Parameter, n.Context.DefaultPipelineParameters)
	}

	run.HookEvent = h
	if h != nil {
		runPayload = sdk.ParametersMapMerge(runPayload, h.Payload)
		run.Payload = runPayload
		run.PipelineParameters = sdk.ParametersMerge(pip.Parameter, n.Context.DefaultPipelineParameters)
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
			return report, false, sdk.WrapError(errm1, "processWorkflowNodeRun> Unable to compute payload")
		}
		runPayload = sdk.ParametersMapMerge(runPayload, m1)
		run.Payload = runPayload
		run.PipelineParameters = sdk.ParametersMerge(n.Context.DefaultPipelineParameters, m.PipelineParameters)
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

	runContext := nodeRunContext{
		Pipeline: pip,
	}
	app, has := n.Application()
	if has {
		runContext.Application = app
	}
	env, has := n.Environment()
	if has {
		runContext.Environment = env
	}
	prjPlat, has := n.ProjectPlatform()
	if has {
		runContext.ProjectPlatform = prjPlat
	}

	// Process parameters for the jobs
	jobParams, errParam := getNodeRunBuildParameters(ctx, p, w, run, runContext)

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
	if len(runs) > 0 {
		parentsParams, errPP := getParentParameters(w, runs, runPayload)
		if errPP != nil {
			return report, false, sdk.WrapError(errPP, "processWorkflowNodeRun> getParentParameters failed")
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

	var vcsInfos vcsInfos
	var errVcs error
	vcsServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
	vcsInfos, errVcs = getVCSInfos(ctx, db, store, vcsServer, gitValues, app.Name, app.VCSServer, app.RepositoryFullname, !isRoot, previousGitValues[tagGitRepository])
	if errVcs != nil {
		if strings.Contains(errVcs.Error(), "branch has been deleted") {
			AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowRunBranchDeleted.ID,
				Args: []interface{}{vcsInfos.Branch},
			})
		} else {
			AddWorkflowRunInfo(w, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{errVcs.Error()},
			})
		}
		if isRoot {
			return report, false, sdk.WrapError(errVcs, "processWorkflowNodeRun> Cannot get VCSInfos")
		}

		return nil, false, nil
	}

	// only if it's the root pipeline, we put the git... in the build parameters
	// this allow user to write some run conditions with .git.var on the root pipeline
	if isRoot {
		setValuesGitInBuildParameters(run, vcsInfos)
	}

	// Check Run Conditions
	if h != nil {
		hooks := w.Workflow.GetHooks()
		hook, ok := hooks[h.WorkflowNodeHookUUID]
		if !ok {
			return report, false, sdk.WrapError(sdk.ErrNoHook, "processWorkflowNodeRun> Unable to find hook %s", h.WorkflowNodeHookUUID)
		}

		// Check conditions
		var params = run.BuildParameters
		// Define specific destination parameters
		dest := w.Workflow.GetNode(hook.WorkflowNodeID)
		if dest == nil {
			return report, false, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "processWorkflowNodeRun> Unable to find node %d", hook.WorkflowNodeID)
		}

		if !checkNodeRunCondition(w, dest.Context.Conditions, params) {
			log.Debug("processWorkflowNodeRun> Avoid trigger workflow from hook %s", hook.UUID)
			return report, false, nil
		}
	} else {
		if !checkNodeRunCondition(w, n.Context.Conditions, run.BuildParameters) {
			log.Debug("processWorkflowNodeRun> Condition failed %d/%d", w.ID, n.ID)
			return report, false, nil
		}
	}

	if !isRoot {
		setValuesGitInBuildParameters(run, vcsInfos)
	}

	// Tag VCS infos : add in tag only if it does not exist
	if !w.TagExists(tagGitRepository) {
		w.Tag(tagGitRepository, run.VCSRepository)
		if run.VCSBranch != "" && run.VCSTag == "" {
			w.Tag(tagGitBranch, run.VCSBranch)
		}
		if run.VCSTag != "" {
			w.Tag(tagGitTag, run.VCSTag)
		}
		if len(run.VCSHash) >= 7 {
			w.Tag(tagGitHash, run.VCSHash[:7])
		} else {
			w.Tag(tagGitHash, run.VCSHash)
		}
		w.Tag(tagGitAuthor, vcsInfos.Author)
	}

	// Add env tag
	if n.Context != nil && n.Context.Environment != nil {
		w.Tag(tagEnvironment, n.Context.Environment.Name)
	}

	for _, info := range w.Infos {
		if info.IsError && info.SubNumber == w.LastSubNumber {
			run.Status = string(sdk.StatusFail)
			run.Done = time.Now()
			break
		}
	}

	if err := insertWorkflowNodeRun(db, run); err != nil {
		return report, true, sdk.WrapError(err, "unable to insert run (node id : %d, node name : %s, subnumber : %d)", run.WorkflowNodeID, run.WorkflowNodeName, run.SubNumber)
	}
	w.LastExecution = time.Now()

	buildParameters := sdk.ParametersToMap(run.BuildParameters)
	_, okUI := buildParameters["cds.ui.pipeline.run"]
	_, okID := buildParameters["cds.node.id"]
	if !okUI || !okID {
		if !okUI {
			uiRunURL := fmt.Sprintf("%s/project/%s/workflow/%s/run/%s/node/%d?name=%s", baseUIURL, buildParameters["cds.project"], buildParameters["cds.workflow"], buildParameters["cds.run.number"], run.ID, buildParameters["cds.workflow"])
			sdk.AddParameter(&run.BuildParameters, "cds.ui.pipeline.run", sdk.StringParameter, uiRunURL)
		}
		if !okID {
			sdk.AddParameter(&run.BuildParameters, "cds.node.id", sdk.StringParameter, fmt.Sprintf("%d", run.ID))
		}

		if err := UpdateNodeRunBuildParameters(db, run.ID, run.BuildParameters); err != nil {
			return report, true, sdk.WrapError(err, "unable to update workflow node run build parameters")
		}
	}

	report.Add(*run)

	//Update workflow run
	if w.WorkflowNodeRuns == nil {
		w.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}
	w.WorkflowNodeRuns[run.WorkflowNodeID] = append(w.WorkflowNodeRuns[run.WorkflowNodeID], *run)
	w.LastSubNumber = MaxSubNumber(w.WorkflowNodeRuns)

	if err := UpdateWorkflowRun(ctx, db, w); err != nil {
		return report, true, sdk.WrapError(err, "unable to update workflow run")
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
			return report, false, sdk.WrapError(err, "unable to check mutexes")
		}
		if nbMutex > 0 {
			log.Debug("processWorkflowNodeRun> Noderun %s processed but not executed because of mutex", n.Name)
			AddWorkflowRunInfo(w, false, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowNodeMutex.ID,
				Args: []interface{}{n.Name},
			})

			if err := UpdateWorkflowRun(ctx, db, w); err != nil {
				return report, true, sdk.WrapError(err, "unable to update workflow run")
			}

			//Mutex is locked. exit without error
			return report, true, nil
		}
		//Mutex is free, continue
	}

	//Execute the node run !
	r1, err := execute(ctx, db, store, p, run, runContext)
	if err != nil {
		return report, true, sdk.WrapError(err, "unable to execute workflow run")
	}
	_, _ = report.Merge(r1, nil)
	return report, true, nil
}

func setValuesGitInBuildParameters(run *sdk.WorkflowNodeRun, vcsInfos vcsInfos) {
	run.VCSRepository = vcsInfos.Repository
	run.VCSBranch = vcsInfos.Branch
	run.VCSTag = vcsInfos.Tag
	run.VCSHash = vcsInfos.Hash
	run.VCSServer = vcsInfos.Server

	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitRepository, sdk.StringParameter, run.VCSRepository)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitBranch, sdk.StringParameter, run.VCSBranch)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitTag, sdk.StringParameter, run.VCSTag)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHash, sdk.StringParameter, run.VCSHash)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitAuthor, sdk.StringParameter, vcsInfos.Author)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitMessage, sdk.StringParameter, vcsInfos.Message)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitURL, sdk.StringParameter, vcsInfos.URL)
	sdk.ParameterAddOrSetValue(&run.BuildParameters, tagGitHTTPURL, sdk.StringParameter, vcsInfos.HTTPUrl)
}

func checkNodeRunCondition(wr *sdk.WorkflowRun, conditions sdk.WorkflowNodeConditions, params []sdk.Parameter) bool {

	var conditionsOK bool
	var errc error
	if conditions.LuaScript == "" {
		conditionsOK, errc = sdk.WorkflowCheckConditions(conditions.PlainConditions, params)
	} else {
		luacheck, err := luascript.NewCheck()
		if err != nil {
			log.Warning("processWorkflowNodeRun> WorkflowCheckConditions error: %s", err)
			AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
				ID:   sdk.MsgWorkflowError.ID,
				Args: []interface{}{fmt.Sprintf("Error init LUA System: %v", err)},
			})
		}
		luacheck.SetVariables(sdk.ParametersToMap(params))
		errc = luacheck.Perform(conditions.LuaScript)
		conditionsOK = luacheck.Result
	}
	if errc != nil {
		log.Warning("processWorkflowNodeRun> WorkflowCheckConditions error: %s", errc)
		AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{fmt.Sprintf("Error on LUA Condition: %v", errc)},
		})
		return false
	}
	return conditionsOK
}

// AddWorkflowRunInfo add WorkflowRunInfo on a WorkflowRun
func AddWorkflowRunInfo(run *sdk.WorkflowRun, isError bool, infos ...sdk.SpawnMsg) {
	for _, i := range infos {
		run.Infos = append(run.Infos, sdk.WorkflowRunInfo{
			APITime:   time.Now(),
			Message:   i,
			IsError:   isError,
			SubNumber: run.LastSubNumber,
		})
	}
}

// computeRunStatus is useful to compute number of runs in success, building and fail
type statusCounter struct {
	success, building, failed, stoppped, skipped, disabled int
}

// getRunStatus return the status depending on number of runs in success, building, stopped and fail
func getRunStatus(counter statusCounter) string {
	switch {
	case counter.building > 0:
		return sdk.StatusBuilding.String()
	case counter.failed > 0:
		return sdk.StatusFail.String()
	case counter.stoppped > 0:
		return sdk.StatusStopped.String()
	case counter.success > 0:
		return sdk.StatusSuccess.String()
	case counter.skipped > 0:
		return sdk.StatusSkipped.String()
	case counter.disabled > 0:
		return sdk.StatusDisabled.String()
	default:
		return sdk.StatusNeverBuilt.String()
	}
}

func computeRunStatus(status string, counter *statusCounter) {
	switch status {
	case sdk.StatusSuccess.String():
		counter.success++
	case sdk.StatusBuilding.String(), sdk.StatusWaiting.String():
		counter.building++
	case sdk.StatusFail.String():
		counter.failed++
	case sdk.StatusStopped.String():
		counter.stoppped++
	case sdk.StatusSkipped.String():
		counter.skipped++
	case sdk.StatusDisabled.String():
		counter.disabled++
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
