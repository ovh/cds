package workflow

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
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
func RunFromHook(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunHookEvent, asCodeMsg []sdk.Message) (*sdk.WorkflowRun, *ProcessorReport, error) {
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

		if trigg, ok := e.Payload["cds.triggered_by.username"]; ok {
			wr.Tag(tagTriggeredBy, trigg)
		} else {
			wr.Tag(tagTriggeredBy, "cds.hook")
		}

		// Add ass code spawn info
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
func ManualRunFromNode(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunManual, nodeID int64) (*sdk.WorkflowRun, *ProcessorReport, error) {
	report := new(ProcessorReport)
	wr.Tag(tagTriggeredBy, e.User.Username)

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

//ManualRun is the entry point to trigger a workflow manually
func ManualRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wr *sdk.WorkflowRun, e *sdk.WorkflowNodeRunManual, asCodeInfos []sdk.Message) (*sdk.WorkflowRun, *ProcessorReport, error) {
	report := new(ProcessorReport)

	ctx, end := observability.Span(ctx, "workflow.ManualRun", observability.Tag(observability.TagWorkflowRun, wr.Number))
	defer end()

	if err := IsValid(ctx, store, db, &wr.Workflow, p, &e.User); err != nil {
		return nil, nil, sdk.WrapError(err, "Unable to valid workflow")
	}

	for _, msg := range asCodeInfos {
		AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{ID: msg.ID, Args: msg.Args})
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
