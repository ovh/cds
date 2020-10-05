package workflow

import (
	"context"
	"time"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	tagTriggeredBy   = "triggered_by"
	tagEnvironment   = "environment"
	tagGitHash       = "git.hash"
	tagGitHashShort  = "git.hash.short"
	tagGitRepository = "git.repository"
	tagGitBranch     = "git.branch"
	tagGitTag        = "git.tag"
	tagGitAuthor     = "git.author"
	tagGitMessage    = "git.message"
	tagGitURL        = "git.url"
	tagGitHTTPURL    = "git.http_url"
	tagGitServer     = "git.server"
)

//RunFromHook is the entry point to trigger a workflow from a hook
func runFromHook(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunHookEvent, asCodeMsg []sdk.Message) (*ProcessorReport, error) {
	var end func()
	ctx, end = telemetry.Span(ctx, "workflow.RunFromHook")
	defer end()

	report := new(ProcessorReport)

	var h *sdk.NodeHook
	if sdk.IsValidUUID(e.WorkflowNodeHookUUID) {
		hooks := wr.Workflow.WorkflowData.GetHooks()
		h = hooks[e.WorkflowNodeHookUUID]
	} else {
		hooks := wr.Workflow.WorkflowData.GetHooksMapRef()
		if ho, ok := hooks[e.WorkflowNodeHookUUID]; ok {
			h = &ho
		}
	}

	if h == nil {
		return report, sdk.WrapError(sdk.ErrNoHook, "unable to find hook %s in %+v", e.WorkflowNodeHookUUID, wr.Workflow.WorkflowData.Node.Hooks)
	}

	//If the hook is on the root, it will trigger a new workflow run
	//Else if will trigger a new subnumber of the last workflow run
	if h.NodeID == wr.Workflow.WorkflowData.Node.ID {
		if err := CompleteWorkflow(ctx, db, &wr.Workflow, proj, LoadOptions{DeepPipeline: true}); err != nil {
			return nil, sdk.WrapError(err, "Unable to complete workflow")
		}

		// Add add code spawn info
		for _, msg := range asCodeMsg {
			AddWorkflowRunInfo(wr, msg.ToSpawnMsg())
		}

		//Process it
		r1, hasRun, errWR := processWorkflowDataRun(ctx, db, store, proj, wr, e, nil, nil)
		if errWR != nil {
			return nil, sdk.WrapError(errWR, "RunFromHook> Unable to process workflow run")
		}
		if !hasRun {
			wr.Status = sdk.StatusNeverBuilt
			wr.LastExecution = time.Now()
			report.Add(ctx, wr)
			return report, sdk.WithStack(sdk.ErrConditionsNotOk)
		}
		report.Merge(ctx, r1)
	}
	return report, nil
}

//ManualRunFromNode is the entry point to trigger manually a piece of an existing run workflow
func manualRunFromNode(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunManual, nodeID int64) (*ProcessorReport, error) {
	report := new(ProcessorReport)

	r1, condOk, err := processWorkflowDataRun(ctx, db, store, proj, wr, nil, e, &nodeID)
	if err != nil {
		return report, sdk.WrapError(err, "unable to process workflow run")
	}
	report.Merge(ctx, r1)
	if !condOk {
		return report, sdk.WithStack(sdk.ErrConditionsNotOk)
	}
	return report, nil
}

func StartWorkflowRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun,
	opts *sdk.WorkflowRunPostHandlerOption, u sdk.AuthConsumer, asCodeInfos []sdk.Message) (*ProcessorReport, error) {
	ctx, end := telemetry.Span(ctx, "api.startWorkflowRun")
	defer end()

	report := new(ProcessorReport)

	for _, msg := range asCodeInfos {
		AddWorkflowRunInfo(wr, msg.ToSpawnMsg())
	}

	wr.Status = sdk.StatusWaiting
	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return report, err
	}

	if opts.Hook != nil {
		// Run from HOOK
		r1, err := runFromHook(ctx, db, store, proj, wr, opts.Hook, asCodeInfos)
		if err != nil {
			return nil, err
		}
		report.Merge(ctx, r1)
	} else {
		// Manual RUN
		if opts.Manual == nil {
			opts.Manual = &sdk.WorkflowNodeRunManual{}
		}
		opts.Manual.Username = u.GetUsername()
		opts.Manual.Email = u.GetEmail()
		opts.Manual.Fullname = u.GetFullname()

		if len(opts.FromNodeIDs) > 0 && len(wr.WorkflowNodeRuns) > 0 {
			// MANUAL RUN FROM NODE

			fromNode := wr.Workflow.WorkflowData.NodeByID(opts.FromNodeIDs[0])
			if fromNode == nil {
				return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "unable to find node %d", opts.FromNodeIDs[0])
			}

			// check permission fo workflow node on handler layer
			if !permission.AccessToWorkflowNode(ctx, db, &wr.Workflow, fromNode, u, sdk.PermissionReadExecute) {
				return nil, sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on root node %d", wr.Workflow.WorkflowData.Node.ID)
			}

			// Continue  the current workflow run
			r1, errmr := manualRunFromNode(ctx, db, store, proj, wr, opts.Manual, fromNode.ID)
			if errmr != nil {
				return report, errmr
			}
			report.Merge(ctx, r1)
		} else {
			// heck permission fo workflow node on handler layer
			// MANUAL RUN FROM ROOT NODE
			if !permission.AccessToWorkflowNode(ctx, db, &wr.Workflow, &wr.Workflow.WorkflowData.Node, u, sdk.PermissionReadExecute) {
				return nil, sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %d", wr.Workflow.WorkflowData.Node.ID)
			}
			// Start new workflow
			r1, errmr := manualRun(ctx, db, store, proj, wr, opts.Manual)
			if errmr != nil {
				return nil, errmr
			}
			report.Merge(ctx, r1)
		}
	}

	return report, nil
}

//ManualRun is the entry point to trigger a workflow manually
func manualRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunManual) (*ProcessorReport, error) {
	report := new(ProcessorReport)
	ctx, end := telemetry.Span(ctx, "workflow.ManualRun", telemetry.Tag(telemetry.TagWorkflowRun, wr.Number))
	defer end()

	if err := CompleteWorkflow(ctx, db, &wr.Workflow, proj, LoadOptions{DeepPipeline: true}); err != nil {
		return nil, sdk.WrapError(err, "unable to valid workflow")
	}

	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return nil, err
	}

	r1, hasRun, errWR := processWorkflowDataRun(ctx, db, store, proj, wr, nil, e, nil)
	if errWR != nil {
		return report, errWR
	}
	report.Merge(ctx, r1)
	if !hasRun {
		return report, sdk.WithStack(sdk.ErrConditionsNotOk)
	}
	return report, nil
}
