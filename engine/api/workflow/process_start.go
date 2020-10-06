package workflow

import (
	"context"
	"fmt"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processStartFromNode(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project,
	wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, startingFromNode *int64, maxsn int64,
	hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)
	start := mapNodes[*startingFromNode]
	if start == nil {
		return nil, false, sdk.ErrWorkflowNodeNotFound
	}

	//Run the node : manual or from an event
	nextSubNumber := maxsn
	nodeRuns, ok := wr.WorkflowNodeRuns[*startingFromNode]
	if ok && len(nodeRuns) > 0 {
		nextSubNumber++
	}
	log.Debug("processWorkflowRun> starting from node %v", startingFromNode)

	// Find ancestors
	nodeIds := start.Ancestors(wr.Workflow.WorkflowData)
	sourceNodesRun := make([]*sdk.WorkflowNodeRun, 0, len(nodeIds))
	for i := range nodeIds {
		nodesRuns, ok := wr.WorkflowNodeRuns[nodeIds[i]]
		if ok && len(nodesRuns) > 0 {
			sourceNodesRun = append(sourceNodesRun, &nodesRuns[0])
		} else {
			return nil, false, sdk.ErrWorkflowNodeParentNotRun
		}
	}

	r1, conditionOK, errP := processNodeRun(ctx, db, store, proj, wr, mapNodes, start, int(nextSubNumber), sourceNodesRun, hookEvent, manual)
	if errP != nil {
		return nil, conditionOK, sdk.WrapError(errP, "processWorkflowRun> Unable to processNodeRun")
	}

	report.Merge(ctx, r1)
	wr.Status = sdk.StatusWaiting

	return report, conditionOK, nil
}

func processStartFromRootNode(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	log.Debug("processWorkflowRun> starting from the root: %d (pipeline %s)", wr.Workflow.WorkflowData.Node.ID, wr.Workflow.Pipelines[wr.Workflow.WorkflowData.Node.Context.PipelineID].Name)
	report := new(ProcessorReport)
	//Run the root: manual or from an event
	AddWorkflowRunInfo(wr, sdk.SpawnMsgNew(*sdk.MsgWorkflowStarting, wr.Workflow.Name, fmt.Sprintf("%d.%d", wr.Number, 0)))

	r1, conditionOK, errP := processNodeRun(ctx, db, store, proj, wr, mapNodes, &wr.Workflow.WorkflowData.Node, 0, nil, hookEvent, manual)
	if errP != nil {
		return nil, false, sdk.WrapError(errP, "Unable to process workflow node run")
	}
	report.Merge(ctx, r1)
	return report, conditionOK, nil
}

func processAllNodesTriggers(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node) (*ProcessorReport, error) {
	report := new(ProcessorReport)
	//Checks the triggers
	for k := range wr.WorkflowNodeRuns {
		// only check the last node run
		nodeRun := &wr.WorkflowNodeRuns[k][0]

		//Trigger only if the node is over (successful or not)
		if sdk.StatusIsTerminated(nodeRun.Status) && nodeRun.Status != sdk.StatusNeverBuilt {
			//Find the node in the workflow
			node := mapNodes[nodeRun.WorkflowNodeID]
			r1, _ := processNodeTriggers(ctx, db, store, proj, wr, mapNodes, []*sdk.WorkflowNodeRun{nodeRun}, node, int(nodeRun.SubNumber))
			report.Merge(ctx, r1)
		}
	}
	return report, nil
}

func processAllJoins(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node) (*ProcessorReport, error) {
	report := new(ProcessorReport)
	//Checks the joins
	for i := range wr.Workflow.WorkflowData.Joins {
		j := &wr.Workflow.WorkflowData.Joins[i]

		// Find node run
		_, has := wr.WorkflowNodeRuns[j.ID]
		if has {
			continue
		}

		sources := make([]*sdk.WorkflowNodeRun, 0)

		//we have to check noderun for every sources
		for _, nodeJoin := range j.JoinContext {
			if _, okF := wr.WorkflowNodeRuns[nodeJoin.ParentID]; okF {
				// Get latest run on parent
				sources = append(sources, &wr.WorkflowNodeRuns[nodeJoin.ParentID][0])
			}
		}

		//now checks if all sources have been completed
		var ok = true

		sourcesParams := map[string]string{}
		for _, nodeRun := range sources {
			if nodeRun == nil {
				ok = false
				break
			}

			if !sdk.StatusIsTerminated(nodeRun.Status) {
				ok = false
				break
			}

			// If there is no conditions on join, keep default condition ( only continue on success )
			if j.Context == nil || (len(j.Context.Conditions.PlainConditions) == 0 && j.Context.Conditions.LuaScript == "") {
				if nodeRun.Status == sdk.StatusFail || nodeRun.Status == sdk.StatusNeverBuilt || nodeRun.Status == sdk.StatusStopped {
					ok = false
					break
				}
			}

			//Merge build parameters from all sources
			sourcesParams = sdk.ParametersMapMerge(sourcesParams, sdk.ParametersToMap(nodeRun.BuildParameters))
		}

		if len(sources) != len(j.JoinContext) {
			ok = false
		}

		//All the sources are completed
		if ok {
			r1, _, err := processNodeRun(ctx, db, store, proj, wr, mapNodes, j, int(wr.LastSubNumber), sources, nil, nil)
			if err != nil {
				return report, sdk.WrapError(err, "unable to process join node")
			}
			report.Merge(ctx, r1)
		}
	}
	return report, nil
}
