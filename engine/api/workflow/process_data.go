package workflow

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/tracingutils"
)

type nodeRunContext struct {
	Application     sdk.Application
	Pipeline        sdk.Pipeline
	Environment     sdk.Environment
	ProjectPlatform sdk.ProjectPlatform
}

func processWorkflowDataRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual, startingFromNode *int64) (*ProcessorReport, bool, error) {
	//TRACABILITY
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
	//////

	//// Process Report
	report := new(ProcessorReport)
	defer func(oldStatus string, wr *sdk.WorkflowRun) {
		if oldStatus != wr.Status {
			report.Add(*wr)
		}
	}(w.Status, w)
	////

	w.Status = string(sdk.StatusBuilding)
	maxsn := MaxSubNumber(w.WorkflowNodeRuns)
	w.LastSubNumber = maxsn

	mapNodes := w.Workflow.WorkflowData.Maps()

	//Checks startingFromNode
	if startingFromNode != nil {
		r1, conditionOK, err := processStartFromNode(ctx, db, store, proj, w, mapNodes, startingFromNode, maxsn, hookEvent, manual)
		if err != nil {
			return report, false, sdk.WrapError(err, "processWorkflow2Run> Unable to processStartFromNode")
		}
		report, _ = report.Merge(r1, nil)
		return report, conditionOK, nil
	}

	//Checks the root
	if len(w.WorkflowNodeRuns) == 0 && len(w.WorkflowNodeOutgoingHookRuns) == 0 {
		r1, conditionOK, err := processStartFromRootNode(ctx, db, store, proj, w, mapNodes, hookEvent, manual)
		if err != nil {
			return report, false, sdk.WrapError(err, "processWorkflow2Run> Unable to processStartFromRootNode")
		}
		report, _ = report.Merge(r1, nil)
		return report, conditionOK, err
	}

	r1, errT := processAllNodesTriggers(ctx, db, store, proj, w, mapNodes)
	if errT != nil {
		return report, false, errT
	}
	report, _ = report.Merge(r1, nil)

	r2 := processAllJoins(ctx, db, store, proj, w, mapNodes, maxsn)
	report, _ = report.Merge(r2, nil)

	// Recompute status counter, it's mandatory to resync
	// the map of workflow node runs of the workflow run to get the right statuses
	// After resync, recompute all status counter compute the workflow status
	// All of this is useful to get the right workflow status is the last node status is skipped
	_, next := observability.Span(ctx, "workflow.syncNodeRuns")
	if err := syncNodeRuns(db, w, LoadRunOptions{}); err != nil {
		next()
		return report, false, sdk.WrapError(err, "processWorkflow2Run> Unable to sync workflow node runs")
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
		return report, false, sdk.WrapError(err, "processWorkflow2Run>")
	}

	return report, true, nil
}
