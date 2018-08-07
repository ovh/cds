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
func RunFromHook(ctx context.Context, dbCopy *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, e *sdk.WorkflowNodeRunHookEvent, asCodeMsg []sdk.Message) (*sdk.WorkflowRun, *ProcessorReport, error) {
	var end func()
	ctx, end = observability.Span(ctx, "workflow.RunFromHook")
	defer end()

	report := new(ProcessorReport)

	hooks := w.GetHooks()
	h, ok := hooks[e.WorkflowNodeHookUUID]
	if !ok {
		return nil, report, sdk.ErrNoHook
	}

	//If the hook is on the root, it will trigger a new workflow run
	//Else if will trigger a new subnumber of the last workflow run
	var number int64
	if h.WorkflowNodeID == w.RootID {

		//Get the next number from our sequence
		var errnum error
		number, errnum = nextRunNumber(db, w)
		if errnum != nil {
			return nil, report, sdk.WrapError(errnum, "RunFromHook> Unable to get next number")
		}

		//Compute a new workflow run
		wr := &sdk.WorkflowRun{
			Number:        number,
			Workflow:      *w,
			WorkflowID:    w.ID,
			Start:         time.Now(),
			LastModified:  time.Now(),
			ProjectID:     w.ProjectID,
			Status:        string(sdk.StatusWaiting),
			LastExecution: time.Now(),
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

		//Insert it
		if err := insertWorkflowRun(db, wr); err != nil {
			return nil, nil, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
		}

		//Process it
		r1, hasRun, errWR := processWorkflowRun(ctx, db, store, p, wr, e, nil, nil)
		if errWR != nil {
			return nil, nil, sdk.WrapError(errWR, "RunFromHook> Unable to process workflow run")
		}
		_, _ = report.Merge(r1, nil)
		if !hasRun {
			wr.Status = sdk.StatusNeverBuilt.String()
			wr.LastExecution = time.Now()
			return wr, report, UpdateWorkflowRun(ctx, db, wr)
		}
	} else {

		//Load the last workflow run
		lastWorkflowRun, err := LoadLastRun(db, w.ProjectKey, w.Name, LoadRunOptions{})
		if err != nil {
			return nil, nil, sdk.WrapError(err, "RunFromHook> Unable to load last run")
		}

		number = lastWorkflowRun.Number

		//Load the last definition of the hooks
		oldHooks := lastWorkflowRun.Workflow.GetHooks()
		oldH, ok := oldHooks[h.UUID]
		if !ok {
			return nil, nil, sdk.WrapError(sdk.ErrNoHook, "RunFromHook> Hook not found")
		}

		//Process the workflow run from the node ID
		r1, _, err := processWorkflowRun(ctx, db, store, p, lastWorkflowRun, e, nil, &oldH.WorkflowNodeID)
		if err != nil {
			return nil, nil, sdk.WrapError(err, "RunFromHook> Unable to process workflow run")
		}
		_, _ = report.Merge(r1, nil)
	}

	run, err := LoadRun(db, w.ProjectKey, w.Name, number, LoadRunOptions{})
	if err != nil {
		return nil, nil, sdk.WrapError(err, "RunFromHook> Unable to reload workflow run")
	}

	return run, report, nil
}

//ManualRunFromNode is the entry point to trigger manually a piece of an existing run workflow
func ManualRunFromNode(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, number int64, e *sdk.WorkflowNodeRunManual, nodeID int64) (*sdk.WorkflowRun, *ProcessorReport, error) {
	report := new(ProcessorReport)

	lastWorkflowRun, errLoadRun := LoadRun(db, w.ProjectKey, w.Name, number, LoadRunOptions{})
	if errLoadRun != nil {
		return nil, report, sdk.WrapError(errLoadRun, "ManualRunFromNode> Unable to load last run")
	}
	lastWorkflowRun.Tag(tagTriggeredBy, e.User.Username)

	r1, condOk, err := processWorkflowRun(ctx, db, store, p, lastWorkflowRun, nil, e, &nodeID)
	if err != nil {
		return nil, report, sdk.WrapError(err, "ManualRunFromNode> Unable to process workflow run")
	}
	_, _ = report.Merge(r1, nil)

	if !condOk {
		return nil, report, sdk.WrapError(sdk.ErrConditionsNotOk, "ManualRunFromNode> Conditions aren't ok")
	}

	var errLoadRunByID error
	lastWorkflowRun, errLoadRunByID = LoadRunByIDAndProjectKey(db, w.ProjectKey, lastWorkflowRun.ID, LoadRunOptions{})
	if errLoadRunByID != nil {
		return nil, report, errLoadRunByID
	}

	return lastWorkflowRun, report, nil
}

//ManualRun is the entry point to trigger a workflow manually
func ManualRun(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, e *sdk.WorkflowNodeRunManual, asCodeInfos []sdk.Message) (*sdk.WorkflowRun, *ProcessorReport, error) {
	report := new(ProcessorReport)
	number, err := nextRunNumber(db, w)
	if err != nil {
		return nil, report, sdk.WrapError(err, "ManualRun> Unable to get next number")
	}

	ctx, end := observability.Span(ctx, "workflow.ManualRun", observability.Tag(observability.TagWorkflowRun, number))
	defer end()

	wr := &sdk.WorkflowRun{
		Number:        number,
		Workflow:      *w,
		WorkflowID:    w.ID,
		Start:         time.Now(),
		LastModified:  time.Now(),
		ProjectID:     w.ProjectID,
		Status:        sdk.StatusWaiting.String(),
		LastExecution: time.Now(),
	}
	wr.Tag(tagTriggeredBy, e.User.Username)

	for _, msg := range asCodeInfos {
		AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{ID: msg.ID, Args: msg.Args})
	}

	if err := insertWorkflowRun(db, wr); err != nil {
		return nil, report, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
	}

	r1, hasRun, errWR := processWorkflowRun(ctx, db, store, p, wr, nil, e, nil)
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
