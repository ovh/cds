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
	Application        sdk.Application
	Pipeline           sdk.Pipeline
	Environment        sdk.Environment
	ProjectIntegration sdk.ProjectIntegration
	NodeGroups         []sdk.GroupPermission
}

func processWorkflowDataRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, wr *sdk.WorkflowRun, hookEvent *sdk.WorkflowNodeRunHookEvent, manual *sdk.WorkflowNodeRunManual, startingFromNode *int64) (*ProcessorReport, bool, error) {
	//TRACEABILITY
	var end func()
	ctx, end = observability.Span(ctx, "workflow.processWorkflowDataRun",
		observability.Tag(observability.TagWorkflowRun, wr.Number),
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
	)
	defer end()

	if wr.Header == nil {
		wr.Header = sdk.WorkflowRunHeaders{}
	}
	wr.Header.Set(sdk.WorkflowRunHeader, strconv.FormatInt(wr.Number, 10))
	wr.Header.Set(sdk.WorkflowHeader, wr.Workflow.Name)
	wr.Header.Set(sdk.ProjectKeyHeader, proj.Key)

	// Push data in header to allow tracing
	if observability.Current(ctx).SpanContext().IsSampled() {
		wr.Header.Set(tracingutils.SampledHeader, "1")
		wr.Header.Set(tracingutils.TraceIDHeader, fmt.Sprintf("%v", observability.Current(ctx).SpanContext().TraceID))
	}
	//////

	//// Process Report
	report := new(ProcessorReport)
	defer func(oldStatus string, wr *sdk.WorkflowRun) {
		if oldStatus != wr.Status {
			report.Add(*wr)
		}
	}(wr.Status, wr)
	////

	wr.Status = string(sdk.StatusBuilding)
	maxsn := MaxSubNumber(wr.WorkflowNodeRuns)
	wr.LastSubNumber = maxsn

	mapNodes := wr.Workflow.WorkflowData.Maps()

	//Checks startingFromNode
	if startingFromNode != nil {
		r1, conditionOK, err := processStartFromNode(ctx, db, store, proj, wr, mapNodes, startingFromNode, maxsn, hookEvent, manual)
		if err != nil {
			return nil, false, sdk.WrapError(err, "processWorkflowDataRun> Unable to processStartFromNode")
		}
		report, _ = report.Merge(r1, nil)
		return report, conditionOK, nil
	}

	//Checks the root
	if len(wr.WorkflowNodeRuns) == 0 {
		r1, conditionOK, err := processStartFromRootNode(ctx, db, store, proj, wr, mapNodes, hookEvent, manual)
		if err != nil {
			return nil, false, sdk.WrapError(err, "processWorkflowDataRun> Unable to processStartFromRootNode")
		}
		report, _ = report.Merge(r1, nil)
		return report, conditionOK, nil
	}

	r1, errT := processAllNodesTriggers(ctx, db, store, proj, wr, mapNodes)
	if errT != nil {
		return nil, false, errT
	}
	report, _ = report.Merge(r1, nil)

	r2, errJ := processAllJoins(ctx, db, store, proj, wr, mapNodes)
	if errJ != nil {
		return nil, false, errJ
	}
	report, _ = report.Merge(r2, nil)

	r1, err := computeAndUpdateWorkflowRunStatus(ctx, db, wr)
	if err != nil {
		return nil, false, sdk.WrapError(err, "processWorkflowDataRun> unable to compute workflow run status")
	}
	report.Merge(r1, nil) // nolint

	return report, true, nil
}

func computeAndUpdateWorkflowRunStatus(ctx context.Context, db gorp.SqlExecutor, wr *sdk.WorkflowRun) (*ProcessorReport, error) {
	report := new(ProcessorReport)
	// Recompute status counter, it's mandatory to resync
	// the map of workflow node runs of the workflow run to get the right statuses
	// After resync, recompute all status counter compute the workflow status
	// All of this is useful to get the right workflow status is the last node status is skipped
	_, next := observability.Span(ctx, "workflow.computeAndUpdateWorkflowRunStatus")
	if err := syncNodeRuns(db, wr, LoadRunOptions{}); err != nil {
		next()
		return report, sdk.WrapError(err, "computeAndUpdateWorkflowRunStatus> Unable to sync workflow node runs")
	}
	next()

	// Reinit the counters
	var counterStatus statusCounter
	for k, v := range wr.WorkflowNodeRuns {
		lastCurrentSn := lastSubNumber(wr.WorkflowNodeRuns[k])
		// Subversion of workflowNodeRun
		for i := range v {
			nodeRun := &wr.WorkflowNodeRuns[k][i]
			// Compute for the last subnumber only
			if lastCurrentSn == nodeRun.SubNumber {
				computeRunStatus(nodeRun.Status, &counterStatus)
			}
		}
	}
	newStatus := getRunStatus(counterStatus)
	if wr.Status == newStatus {
		return report, nil
	}
	wr.Status = newStatus
	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return report, sdk.WrapError(err, "computeAndUpdateWorkflowRunStatus>")
	}
	return report, nil
}
