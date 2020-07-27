package workflow

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func processNodeOutGoingHook(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, parentNodeRun []*sdk.WorkflowNodeRun, node *sdk.Node, subNumber int, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	ctx, end := telemetry.Span(ctx, "workflow.processNodeOutGoingHook")
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
		} else if exitingNodeRun != nil && exitingNodeRun.Status != sdk.StatusStopped {
			log.Debug("hook %d is over, we have to reprocess al the things", node.ID)
			r1, _, err := processWorkflowDataRun(ctx, db, store, proj, wr, nil, nil, nil)
			if err != nil {
				return nil, false, sdk.WrapError(err, "unable to process workflow run after outgoing hooks")
			}
			report.Merge(ctx, r1)
			return report, false, nil
		} else if exitingNodeRun != nil && exitingNodeRun.Status == sdk.StatusStopped {
			return report, false, nil
		}
	}

	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeHooks)
	if err != nil {
		return nil, false, sdk.WrapError(err, "cannot get hooks service")
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
		Status:           sdk.StatusWaiting,
		Start:            time.Now(),
		LastModified:     time.Now(),
		SourceNodeRuns:   parentsIDs,
		UUID:             sdk.UUID(),
	}

	var errBP error
	hookRun.BuildParameters, errBP = computeBuildParameters(wr, &hookRun, parentNodeRun, manual)
	if errBP != nil {
		return nil, false, errBP
	}

	// PARENT BUILD PARAMETER
	if len(parentNodeRun) > 0 {
		_, next := telemetry.Span(ctx, "workflow.getParentParameters")
		parentsParams, errPP := getParentParameters(wr, parentNodeRun)
		next()
		if errPP != nil {
			return nil, false, sdk.WrapError(errPP, "processNode> getParentParameters failed")
		}
		mapBuildParams := sdk.ParametersToMap(hookRun.BuildParameters)
		mapParentParams := sdk.ParametersToMap(parentsParams)
		hookRun.BuildParameters = sdk.ParametersFromMap(sdk.ParametersMapMerge(mapBuildParams, mapParentParams))
	}

	if node.OutGoingHookContext != nil {
		hookRun.OutgoingHook = &sdk.NodeOutGoingHook{
			HookModelName: node.OutGoingHookContext.HookModelName,
			HookModelID:   node.OutGoingHookContext.HookModelID,
			NodeID:        node.OutGoingHookContext.NodeID,
			ID:            node.OutGoingHookContext.ID,
			Config:        make(map[string]sdk.WorkflowNodeHookConfigValue, len(node.OutGoingHookContext.Config)),
		}
		for k, v := range node.OutGoingHookContext.Config {
			// If payload run interpolate
			if k == sdk.Payload {
				// Take all parent parameters without any exceptions
				allParentParams := make([]sdk.Parameter, 0, len(hookRun.BuildParameters))
				for _, parentNodeRun := range parentNodeRun {
					allParentParams = append(allParentParams, parentNodeRun.BuildParameters...)
				}

				result, err := interpolate.Do(v.Value, sdk.ParametersToMap(allParentParams))
				if err != nil {
					return nil, true, sdk.WrapError(err, "unable to interpolate payload %s", v.Value)
				}
				v.Value = result
			}
			hookRun.OutgoingHook.Config[k] = v
		}
	}

	if !checkCondition(ctx, wr, node.Context.Conditions, hookRun.BuildParameters) {
		log.Debug("Condition failed on processNodeOutGoingHook %d/%d %+v", wr.ID, node.ID, hookRun.BuildParameters)
		return report, false, nil
	}

	var task sdk.Task
	if _, _, err := services.NewClient(db, srvs).DoJSONRequest(ctx, "POST", "/task/execute", hookRun, &task); err != nil {
		log.Warning(ctx, "outgoing hook execution failed: %v", err)
		hookRun.Status = sdk.StatusFail
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
	}

	if err := UpdateNodeRunBuildParameters(db, hookRun.ID, hookRun.BuildParameters); err != nil {
		return nil, false, sdk.WrapError(err, "unable to update workflow node run build parameters")
	}

	report.Add(ctx, hookRun)

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
