package workflow

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processNodeOutGoingHook(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, parentNodeRun []*sdk.WorkflowNodeRun, node *sdk.Node, subNumber int) (*ProcessorReport, bool, error) {
	ctx, end := observability.Span(ctx, "workflow.processNodeOutGoingHook")
	defer end()

	report := new(ProcessorReport)

	//Check if the WorkflowNodeOutgoingHookRun already exist with the same subnumber
	nrs, ok := wr.WorkflowNodeRuns[node.ID]
	if ok {
		var exitingNodeRun *sdk.WorkflowNodeRun
		for i := range nrs {
			if nrs[i].Number == wr.Number && int(nrs[i].SubNumber) == subNumber {
				exitingNodeRun = &nrs[i]
				break
			}
		}
		// If the hookrun is at status terminated, let's trigger outgoing children
		if exitingNodeRun != nil && !sdk.StatusIsTerminated(exitingNodeRun.Status) {
			log.Debug("hook %d already processed", node.ID)
			return nil, false, nil
		} else if exitingNodeRun != nil && exitingNodeRun.Status != sdk.StatusStopped.String() {
			log.Debug("hook %d is over, we have to reprocess al the things", node.ID)
			for i := range node.Triggers {
				t := &node.Triggers[i]
				log.Debug("checking trigger %+v", t)
				r1, err := processNodeTriggers(ctx, db, store, proj, wr, mapNodes, []*sdk.WorkflowNodeRun{exitingNodeRun}, node, subNumber)
				if err != nil {
					return nil, false, sdk.WrapError(err, "Unable to process outgoing hook triggers")
				}
				report.Merge(r1, nil) // nolint
			}
			return report, false, nil
		} else if exitingNodeRun != nil && exitingNodeRun.Status == sdk.StatusStopped.String() {
			return report, false, nil
		}
	}

	//FIX: For the moment, we trigger outgoing hooks on success
	for _, p := range parentNodeRun {
		if p.Status != sdk.StatusSuccess.String() {
			return report, false, nil
		}
	}

	srvs, err := services.FindByType(db, services.TypeHooks)
	if err != nil {
		return nil, false, sdk.WrapError(err, "Cannot get hooks service")
	}

	mapParams := map[string]string{}
	for _, p := range parentNodeRun {
		m := sdk.ParametersToMap(p.BuildParameters)
		sdk.ParametersMapMerge(mapParams, m)
	}

	node.OutGoingHookContext.Config[sdk.HookConfigModelName] = sdk.WorkflowNodeHookConfigValue{
		Value:        wr.Workflow.OutGoingHookModels[node.OutGoingHookContext.HookModelID].Name,
		Configurable: false,
		Type:         sdk.HookConfigTypeString,
	}
	node.OutGoingHookContext.Config[sdk.HookConfigModelType] = sdk.WorkflowNodeHookConfigValue{
		Value:        wr.Workflow.OutGoingHookModels[node.OutGoingHookContext.HookModelID].Type,
		Configurable: false,
		Type:         sdk.HookConfigTypeString,
	}

	parentsIDs := make([]int64, 0, len(parentNodeRun))
	for _, r := range parentNodeRun {
		parentsIDs = append(parentsIDs, r.ID)
	}
	var hookRun = sdk.WorkflowNodeRun{
		WorkflowRunID:    wr.ID,
		WorkflowID:       wr.Workflow.ID,
		WorkflowNodeID:   node.ID,
		WorkflowNodeName: node.Name,
		Number:           wr.Number,
		SubNumber:        int64(subNumber),
		Status:           sdk.StatusWaiting.String(),
		Start:            time.Now(),
		LastModified:     time.Now(),
		SourceNodeRuns:   parentsIDs,
		BuildParameters:  sdk.ParametersFromMap(mapParams),
		UUID:             sdk.UUID(),
		OutgoingHook:     node.OutGoingHookContext,
	}

	if !checkNodeRunCondition(wr, node.Context.Conditions, hookRun.BuildParameters) {
		log.Debug("Condition failed %d/%d %+v", wr.ID, node.ID, hookRun.BuildParameters)
		return report, false, nil
	}

	var task sdk.Task
	if _, err := services.DoJSONRequest(ctx, db, srvs, "POST", "/task/execute", hookRun, &task); err != nil {
		log.Warning("outgoing hook execution failed: %v", err)
		hookRun.Status = sdk.StatusFail.String()
	}

	if len(task.Executions) > 0 {
		hookRun.HookExecutionID = task.Executions[0].UUID
		hookRun.HookExecutionTimeStamp = task.Executions[0].Timestamp
	}

	if err := insertWorkflowNodeRun(db, &hookRun); err != nil {
		return nil, true, sdk.WrapError(err, "unable to insert run (node id : %d, node name : %s, subnumber : %d)", hookRun.WorkflowNodeID, hookRun.WorkflowNodeName, hookRun.SubNumber)
	}
	wr.LastExecution = time.Now()

	buildParameters := sdk.ParametersToMap(hookRun.BuildParameters)
	_, okID := buildParameters["cds.node.id"]
	if !okID {
		if !okID {
			sdk.AddParameter(&hookRun.BuildParameters, "cds.node.id", sdk.StringParameter, fmt.Sprintf("%d", hookRun.ID))
		}

		if err := UpdateNodeRunBuildParameters(db, hookRun.ID, hookRun.BuildParameters); err != nil {
			return nil, true, sdk.WrapError(err, "unable to update workflow node run build parameters")
		}
	}

	report.Add(hookRun)

	//Update workflow run
	if wr.WorkflowNodeRuns == nil {
		wr.WorkflowNodeRuns = make(map[int64][]sdk.WorkflowNodeRun)
	}
	if wr.WorkflowNodeRuns[node.ID] == nil {
		wr.WorkflowNodeRuns[node.ID] = make([]sdk.WorkflowNodeRun, 0)
	}
	wr.WorkflowNodeRuns[node.ID] = append(wr.WorkflowNodeRuns[node.ID], hookRun)

	sort.Slice(wr.WorkflowNodeRuns[node.ID], func(i, j int) bool {
		return wr.WorkflowNodeRuns[node.ID][i].SubNumber > wr.WorkflowNodeRuns[node.ID][j].SubNumber
	})

	wr.LastSubNumber = MaxSubNumber(wr.WorkflowNodeRuns)

	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return nil, true, sdk.WrapError(err, "unable to update workflow run")
	}

	return report, true, nil
}
