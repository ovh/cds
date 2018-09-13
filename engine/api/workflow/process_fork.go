package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func processWorkflowNodeFork(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, parentNodeRun *sdk.WorkflowNodeRun, f sdk.WorkflowNodeFork) *ProcessorReport {
	report := new(ProcessorReport)
	for j := range f.Triggers {
		t := &f.Triggers[j]

		// check if the destination node already exists on w.WorkflowNodeRuns with the same subnumber
		var abortTrigger bool
		if previousRunArray, ok := wr.WorkflowNodeRuns[t.WorkflowDestNode.ID]; ok {
			for _, previousRun := range previousRunArray {
				if previousRun.SubNumber == parentNodeRun.SubNumber {
					abortTrigger = true
					break
				}
			}
		}

		if !abortTrigger {
			//Keep the subnumber of the previous node in the graph
			r1, _, errPwnr := processWorkflowNodeRun(ctx, db, store, proj, wr, &t.WorkflowDestNode, int(parentNodeRun.SubNumber), []int64{parentNodeRun.ID}, nil, nil)
			if errPwnr != nil {
				log.Error("processWorkflowRun> Unable to process node ID=%d: %s", t.WorkflowDestNode.ID, errPwnr)
				AddWorkflowRunInfo(wr, true, sdk.SpawnMsg{
					ID:   sdk.MsgWorkflowError.ID,
					Args: []interface{}{errPwnr.Error()},
				})
			}
			_, _ = report.Merge(r1, nil)
			continue
		}
	}
	return report
}
