package workflow

import (
	"context"
	"sort"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processNodeOutGoingHook(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, parentNodeRun []*sdk.WorkflowNodeRun, node *sdk.Node, subNumber int) (*ProcessorReport, error) {
	ctx, end := observability.Span(ctx, "workflow.processNodeOutGoingHook")
	defer end()

	report := new(ProcessorReport)

	if wr.WorkflowNodeOutgoingHookRuns == nil {
		wr.WorkflowNodeOutgoingHookRuns = make(map[int64][]sdk.WorkflowNodeOutgoingHookRun)
	}

	//Check if the WorkflowNodeOutgoingHookRun already exist with the same subnumber
	hrs, ok := wr.WorkflowNodeOutgoingHookRuns[node.ID]
	if ok {
		var alreadyProcessed, isWaiting bool
		for _, hr := range hrs {
			if hr.Number == wr.Number && int(hr.SubNumber) == subNumber {
				alreadyProcessed = true
				break
			}
		}
		if alreadyProcessed && isWaiting {
			log.Debug("hook %d already processed", node.ID)
			return nil, nil
		} else if alreadyProcessed {

			log.Debug("hook %d is over, we have to reprocess al the things", node.ID)

			for i := range node.Triggers {
				t := &node.Triggers[i]
				log.Debug("checking trigger %+v", t)
				r1, err := processNodeTriggers(ctx, db, store, proj, wr, mapNodes, parentNodeRun, node, subNumber)
				if err != nil {
					return nil, sdk.WrapError(err, "processNodeOutGoingHook> Unable to process outgoing hook triggers")
				}
				report.Merge(r1, nil) // nolint
			}

			return nil, nil
		}
	}

	srvs, err := services.FindByType(db, services.TypeHooks)
	if err != nil {
		return nil, sdk.WrapError(err, "process> Cannot get hooks service")
	}

	mapParams := map[string]string{}
	for _, p := range parentNodeRun {
		m := sdk.ParametersToMap(p.BuildParameters)
		sdk.ParametersMapMerge(mapParams, m)
	}

	hook := sdk.WorkflowNodeOutgoingHook{
		ID:                  node.ID,
		WorkflowNodeID:      node.ID,
		WorkflowHookModelID: node.OutGoingHookContext.HookModelID,
		Ref:                 node.Ref,
		WorkflowHookModel:   wr.Workflow.OutGoingHookModels[node.OutGoingHookContext.HookModelID],
		Config:              node.OutGoingHookContext.Config,
	}
	var hookRun = sdk.WorkflowNodeOutgoingHookRun{
		WorkflowRunID:              wr.ID,
		HookRunID:                  sdk.UUID(),
		Status:                     sdk.StatusWaiting.String(),
		Number:                     wr.Number,
		SubNumber:                  int64(subNumber),
		WorkflowNodeOutgoingHookID: node.ID,
		Hook:   hook,
		Params: mapParams,
	}

	var taskExecution sdk.TaskExecution
	if _, err := services.DoJSONRequest(ctx, srvs, "POST", "/task/execute", hookRun, &taskExecution); err != nil {
		log.Warning("outgoing hook execution failed: %v", err)
		hookRun.Status = sdk.StatusFail.String()
	}

	hookRun.Status = sdk.StatusWaiting.String()

	if wr.WorkflowNodeOutgoingHookRuns[node.ID] == nil {
		wr.WorkflowNodeOutgoingHookRuns[node.ID] = make([]sdk.WorkflowNodeOutgoingHookRun, 0)
	}
	wr.WorkflowNodeOutgoingHookRuns[node.ID] = append(wr.WorkflowNodeOutgoingHookRuns[node.ID], hookRun)

	sort.Slice(wr.WorkflowNodeOutgoingHookRuns[node.ID], func(i, j int) bool {
		return wr.WorkflowNodeOutgoingHookRuns[node.ID][i].SubNumber > wr.WorkflowNodeOutgoingHookRuns[node.ID][j].SubNumber
	})

	report.Add(hookRun)

	return report, nil
}
