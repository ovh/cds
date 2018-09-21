package workflow

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processStartFromNode(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, startingFromNode *int64, maxsn int64, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)
	start := mapNodes[*startingFromNode]
	if start == nil {
		return report, false, sdk.ErrWorkflowNodeNotFound
	}

	//Run the node : manual or from an event
	nextSubNumber := maxsn
	nodeRuns, ok := wr.WorkflowNodeRuns[*startingFromNode]
	if ok && len(nodeRuns) > 0 {
		nextSubNumber++
	}
	log.Debug("processWorkflowRun> starting from node %#v", startingFromNode)
	// Find ancestors
	nodeIds := start.Ancestors(wr.Workflow.WorkflowData, mapNodes, false)
	sourceNodesRun := make([]*sdk.WorkflowNodeRun, 0, len(nodeIds))
	for i := range nodeIds {
		nodesRuns, ok := wr.WorkflowNodeRuns[nodeIds[i]]
		if ok && len(nodesRuns) > 0 {
			sourceNodesRun = append(sourceNodesRun, &nodesRuns[0])
		} else {
			return report, false, sdk.ErrWorkflowNodeParentNotRun
		}
	}

	r1, conditionOK, errP := processNodeRun(ctx, db, store, proj, wr, mapNodes, start, int(nextSubNumber), sourceNodesRun, hookEvent, manual)
	if errP != nil {
		return report, conditionOK, sdk.WrapError(errP, "processWorkflowRun> Unable to processNodeRun")
	}

	report, _ = report.Merge(r1, nil)
	wr.Status = sdk.StatusWaiting.String()

	return report, conditionOK, nil
}

func processStartFromRootNode(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	log.Debug("processWorkflowRun> starting from the root : %d (pipeline %s)", wr.Workflow.WorkflowData.Node.ID, wr.Workflow.Pipelines[wr.Workflow.WorkflowData.Node.Context.ID].Name)
	report := new(ProcessorReport)
	//Run the root: manual or from an event
	AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{
		ID: sdk.MsgWorkflowStarting.ID,
		Args: []interface{}{
			wr.Workflow.Name,
			fmt.Sprintf("%d.%d", wr.Number, 0),
		},
	})

	r1, conditionOK, errP := processNodeRun(ctx, db, store, proj, wr, mapNodes, &wr.Workflow.WorkflowData.Node, 0, nil, hookEvent, manual)
	if errP != nil {
		return report, false, sdk.WrapError(errP, "processNodeRun> Unable to process workflow node run")
	}
	report, _ = report.Merge(r1, nil)
	return report, conditionOK, nil
}

func processAllNodesTriggers(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node) (*ProcessorReport, error) {
	report := new(ProcessorReport)
	//Checks the triggers
	for k := range wr.WorkflowNodeRuns {
		// only check the last node run
		nodeRun := &wr.WorkflowNodeRuns[k][0]

		haveToUpdate := false
		//Trigger only if the node is over (successful or not)
		if sdk.StatusIsTerminated(nodeRun.Status) && nodeRun.Status != sdk.StatusNeverBuilt.String() {
			//Find the node in the workflow
			node := mapNodes[nodeRun.WorkflowNodeID]
			r1, _ := processNodeTriggers(ctx, db, store, proj, wr, mapNodes, []*sdk.WorkflowNodeRun{nodeRun}, node, int(nodeRun.SubNumber))
			_, _ = report.Merge(r1, nil)
		}

		if haveToUpdate {
			if err := updateNodeRunStatusAndTriggersRun(db, nodeRun); err != nil {
				return nil, sdk.WrapError(err, "processAllNodesTriggers> Cannot update node run")
			}
		}
	}
	return report, nil
}

func processAllJoins(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, maxsn int64) *ProcessorReport {
	report := new(ProcessorReport)
	//Checks the joins
	for i := range wr.Workflow.WorkflowData.Joins {
		j := &wr.Workflow.WorkflowData.Joins[i]
		sources := make([]*sdk.WorkflowNodeRun, 0)

		//we have to check noderun for every sources
		for _, nodeJoin := range j.JoinContext {
			if _, okF := wr.WorkflowNodeRuns[nodeJoin.ParentID]; okF {
				// Get lastest run on parent
				sources = append(sources, &wr.WorkflowNodeRuns[nodeJoin.ParentID][0])
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

		if len(sources) != len(j.JoinContext) {
			ok = false
		}
		//All the sources are completed
		if ok {
			r1, _ := processNodeTriggers(ctx, db, store, proj, wr, mapNodes, sources, j, int(maxsn))
			_, _ = report.Merge(r1, nil)
		}
	}
	return report
}
