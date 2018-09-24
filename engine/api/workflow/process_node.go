package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processNodeTriggers(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, parentNodeRun []*sdk.WorkflowNodeRun, node *sdk.Node, parentSubNumber int) (*ProcessorReport, error) {
	report := new(ProcessorReport)

	for j := range node.Triggers {
		t := &node.Triggers[j]

		var abortTrigger bool
		if previousRunArray, ok := wr.WorkflowNodeRuns[t.ChildNode.ID]; ok {
			for _, previousRun := range previousRunArray {
				if int(previousRun.SubNumber) == parentSubNumber {
					abortTrigger = true
					break
				}
			}
		}

		if !abortTrigger {
			//Keep the subnumber of the previous node in the graph
			r1, _, errPwnr := processNodeRun(ctx, db, store, proj, wr, mapNodes, &t.ChildNode, int(parentSubNumber), parentNodeRun, nil, nil)
			if errPwnr != nil {
				log.Error("processWorkflowRun> Unable to process node ID=%d: %s", t.ChildNode.ID, errPwnr)
				AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
					ID:   sdk.MsgWorkflowError.ID,
					Args: []interface{}{errPwnr.Error()},
				})
			}
			_, _ = report.Merge(r1, nil)
			continue
		}
	}
	return report, nil
}

func processNodeRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, mapNodes map[int64]*sdk.Node, n *sdk.Node, subNumber int, parentNodeRuns []*sdk.WorkflowNodeRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual) (*ProcessorReport, bool, error) {
	report := new(ProcessorReport)
	exist, errN := nodeRunExist(db, n.ID, wr.Number, subNumber)
	if errN != nil {
		return nil, false, sdk.WrapError(errN, "processNodeRun> unable to check if node run exist")
	}
	if exist {
		return nil, false, nil
	}

	var end func()
	ctx, end = observability.Span(ctx, "workflow.processWorkflowNodeRun",
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
		observability.Tag(observability.TagWorkflowRun, wr.Number),
		observability.Tag(observability.TagWorkflowNode, n.Name),
	)
	defer end()

	switch n.Type {
	case sdk.NodeTypeFork:
		r1, errT := processNodeTriggers(ctx, db, store, proj, wr, mapNodes, parentNodeRuns, n, subNumber)
		_, _ = report.Merge(r1, nil)
		return report, true, sdk.WrapError(errT, "processNodeRun> Unable to processNodeTriggers")
	case sdk.NodeTypeOutGoingHook:
		r1, errO := processNodeOutGoingHook(ctx, db, store, proj, wr, mapNodes, parentNodeRuns, n, subNumber)
		_, _ = report.Merge(r1, nil)
		return report, true, sdk.WrapError(errO, "processNodeRun> Unable to processNodeOutGoingHook")
	case sdk.NodeTypePipeline:
		parentsIDs := make([]int64, len(parentNodeRuns))
		for i := range parentNodeRuns {
			parentsIDs[i] = parentNodeRuns[i].ID
		}
		r1, conditionOk, errN := processNodeRunPipeline(ctx, db, store, proj, wr, mapNodes, n, subNumber, parentsIDs, hookEvent, manual)
		_, _ = report.Merge(r1, nil)
		return r1, conditionOk, sdk.WrapError(errN, "processNodeRun> unable to processNodeRunPipeline")
	}
	return nil, false, nil
}
