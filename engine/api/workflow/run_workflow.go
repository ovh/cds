package workflow

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

const (
	tagTriggeredBy   = "triggered_by"
	tagEnvironment   = "environment"
	tagGitHash       = "git.hash"
	tagGitRepository = "git.repository"
	tagGitBranch     = "git.branch"
	tagGitTag        = "git.tag"
	tagGitAuthor     = "git.author"
	tagGitMessage    = "git.message"
	tagGitURL        = "git.url"
	tagGitHTTPURL    = "git.http_url"
)

//RunFromHook is the entry point to trigger a workflow from a hook
func runFromHook(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunHookEvent, asCodeMsg []sdk.Message) (*sdk.WorkflowRun, *ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.RunFromHook")
	defer end()

	report := new(ProcessorReport)

	hooks := wr.Workflow.GetHooks()
	h, ok := hooks[e.WorkflowNodeHookUUID]
	if !ok {
		return nil, report, sdk.ErrNoHook
	}

	//If the hook is on the root, it will trigger a new workflow run
	//Else if will trigger a new subnumber of the last workflow run
	if h.WorkflowNodeID == wr.Workflow.Root.ID {
		if err := IsValid(ctx, store, db, &wr.Workflow, p, nil); err != nil {
			return nil, nil, sdk.WrapError(err, "Unable to valid workflow")
		}

		// Add add code spawn info
		for _, msg := range asCodeMsg {
			AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{ID: msg.ID, Args: msg.Args})
		}

		//Process it
		r1, hasRun, errWR := processWorkflowDataRun(ctx, db, store, p, wr, e, nil, nil)
		if errWR != nil {
			return nil, nil, sdk.WrapError(errWR, "RunFromHook> Unable to process workflow run")
		}
		if !hasRun {
			wr.Status = sdk.StatusNeverBuilt.String()
			wr.LastExecution = time.Now()
			report.Add(wr)
			return wr, report, UpdateWorkflowRun(ctx, db, wr)
		}
		_, _ = report.Merge(r1, nil)
	}
	return wr, report, nil
}

//ManualRunFromNode is the entry point to trigger manually a piece of an existing run workflow
func manualRunFromNode(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunManual, nodeID int64) (*sdk.WorkflowRun, *ProcessorReport, error) {
	report := new(ProcessorReport)

	r1, condOk, err := processWorkflowDataRun(ctx, db, store, p, wr, nil, e, &nodeID)
	if err != nil {
		return nil, report, sdk.WrapError(err, "Unable to process workflow run")
	}
	_, _ = report.Merge(r1, nil)

	if !condOk {
		return nil, report, sdk.WrapError(sdk.ErrConditionsNotOk, "ManualRunFromNode> Conditions aren't ok")
	}
	return wr, report, nil
}

func StartWorkflowRun(ctx context.Context, db *gorp.DbMap, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun, opts *sdk.WorkflowRunPostHandlerOption, u *sdk.User, asCodeInfos []sdk.Message) (*ProcessorReport, error) {
	ctx, end := observability.Span(ctx, "api.startWorkflowRun")
	defer end()

	report := new(ProcessorReport)

	tx, errb := db.Begin()
	if errb != nil {
		return nil, sdk.WrapError(errb, "Cannot start transaction")
	}
	defer tx.Rollback() // nolint

	for _, msg := range asCodeInfos {
		AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{ID: msg.ID, Args: msg.Args})
	}

	wr.Status = sdk.StatusWaiting.String()
	if err := UpdateWorkflowRun(ctx, tx, wr); err != nil {
		return report, err
	}

	if opts.Hook != nil {
		// Run from HOOK
		_, r1, err := runFromHook(ctx, tx, store, p, wr, opts.Hook, asCodeInfos)
		if err != nil {
			return nil, sdk.WrapError(err, "Unable to run workflow from hook")
		}

		_, _ = report.Merge(r1, nil)

	} else {
		// Manual RUN
		if opts.Manual == nil {
			opts.Manual = &sdk.WorkflowNodeRunManual{}
		}
		opts.Manual.User = *u

		if len(opts.FromNodeIDs) > 0 && len(wr.WorkflowNodeRuns) > 0 {
			// MANUAL RUN FROM NODE

			fromNode := wr.Workflow.WorkflowData.NodeByID(opts.FromNodeIDs[0])
			if fromNode == nil {
				return nil, sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "unable to find node %d", opts.FromNodeIDs[0])
			}

			if !permission.AccessToWorkflowNode(&wr.Workflow, fromNode, u, permission.PermissionReadExecute) {
				return nil, sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on root node %d", wr.Workflow.Root.ID)
			}

			// Continue  the current workflow run
			_, r1, errmr := manualRunFromNode(ctx, tx, store, p, wr, opts.Manual, fromNode.ID)
			if errmr != nil {
				return nil, sdk.WrapError(errmr, "Unable to run workflow")
			}
			_, _ = report.Merge(r1, nil)

		} else {
			// MANUAL RUN FROM ROOT NODE
			if !permission.AccessToWorkflowNode(&wr.Workflow, &wr.Workflow.WorkflowData.Node, u, permission.PermissionReadExecute) {
				return nil, sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %d", wr.Workflow.WorkflowData.Node.ID)
			}
			// Start new workflow
			_, r1, errmr := manualRun(ctx, tx, store, p, wr, opts.Manual)
			if errmr != nil {
				return nil, sdk.WrapError(errmr, "Unable to run workflow")
			}
			_, _ = report.Merge(r1, nil)
		}
	}

	//Commit and return success
	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "Unable to commit transaction")
	}
	return report, nil
}

//ManualRun is the entry point to trigger a workflow manually
func manualRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunManual) (*sdk.WorkflowRun, *ProcessorReport, error) {
	report := new(ProcessorReport)
	ctx, end := observability.Span(ctx, "workflow.ManualRun", observability.Tag(observability.TagWorkflowRun, wr.Number))
	defer end()

	if err := IsValid(ctx, store, db, &wr.Workflow, p, &e.User); err != nil {
		return nil, nil, sdk.WrapError(err, "Unable to valid workflow")
	}

	if err := UpdateWorkflowRun(ctx, db, wr); err != nil {
		return nil, nil, err
	}

	r1, hasRun, errWR := processWorkflowDataRun(ctx, db, store, p, wr, nil, e, nil)
	if errWR != nil {
		return wr, report, sdk.WrapError(errWR, "ManualRun")
	}
	_, _ = report.Merge(r1, nil)
	if !hasRun {
		wr.Status = sdk.StatusNeverBuilt.String()
		report.Add(wr)
		return wr, report, UpdateWorkflowRun(ctx, db, wr)
	}

	wrUpdated, errReload := LoadRunByID(db, wr.ID, LoadRunOptions{})
	if errReload == nil {
		return wrUpdated, report, nil
	}
	return wr, report, nil
}
