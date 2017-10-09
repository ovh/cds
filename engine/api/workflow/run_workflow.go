package workflow

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

const (
	tagTriggeredBy = "triggered_by"
)

//RunFromHook is the entry point to trigger a workflow from a hook
func RunFromHook(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, e *sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error) {
	hooks := w.GetHooks()
	h, ok := hooks[e.WorkflowNodeHookUUID]
	if !ok {
		return nil, sdk.ErrNoHook
	}

	//If the hook is on the root, it will trigger a new workflow run
	//Else if will trigger a new subnumber of the last workflow run
	var number int64
	if h.WorkflowNodeID == w.RootID {

		//Get the next number from our sequence
		var errnum error
		number, errnum = nextRunNumber(db, w)
		if errnum != nil {
			return nil, sdk.WrapError(errnum, "RunFromHook> Unable to get next number")
		}

		//Compute a new workflow run
		wr := &sdk.WorkflowRun{
			Number:       number,
			Workflow:     *w,
			WorkflowID:   w.ID,
			Start:        time.Now(),
			LastModified: time.Now(),
			ProjectID:    w.ProjectID,
			Status:       string(sdk.StatusWaiting),
		}

		//Insert it
		if err := insertWorkflowRun(db, wr); err != nil {
			return nil, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
		}

		//Process it
		if err := processWorkflowRun(db, store, p, wr, e, nil, nil); err != nil {
			return nil, sdk.WrapError(err, "RunFromHook> Unable to process workflow run")
		}
	} else {

		//Load the last workflow run
		lastWorkflowRun, err := LoadLastRun(db, w.ProjectKey, w.Name)
		if err != nil {
			return nil, sdk.WrapError(err, "RunFromHook> Unable to load last run")
		}

		number = lastWorkflowRun.Number

		//Load the last definition of the hooks
		oldHooks := lastWorkflowRun.Workflow.GetHooks()
		oldH, ok := oldHooks[h.UUID]
		if !ok {
			return nil, sdk.WrapError(sdk.ErrNoHook, "RunFromHook> Hook not found")
		}

		//Process the workflow run from the node ID
		if err := processWorkflowRun(db, store, p, lastWorkflowRun, e, nil, &oldH.WorkflowNodeID); err != nil {
			return nil, sdk.WrapError(err, "RunFromHook> Unable to process workflow run")
		}
	}

	run, err := LoadRun(db, w.ProjectKey, w.Name, number)
	if err != nil {
		return nil, sdk.WrapError(err, "RunFromHook> Unable to reload workflow run")
	}

	return run, nil
}

//ManualRunFromNode is the entry point to trigger manually a piece of an existing run workflow
func ManualRunFromNode(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, number int64, e *sdk.WorkflowNodeRunManual, nodeID int64) (*sdk.WorkflowRun, error) {
	lastWorkflowRun, err := LoadRun(db, w.ProjectKey, w.Name, number)
	lastWorkflowRun.Tag(tagTriggeredBy, e.User.Username)

	if err != nil {
		return nil, sdk.WrapError(err, "ManualRunFromNode> Unable to load last run")
	}

	if err := processWorkflowRun(db, store, p, lastWorkflowRun, nil, e, &nodeID); err != nil {
		return nil, sdk.WrapError(err, "ManualRunFromNode> Unable to process workflow run")
	}

	lastWorkflowRun, err = LoadRunByIDAndProjectKey(db, w.ProjectKey, lastWorkflowRun.ID)
	if err != nil {
		return nil, err
	}

	return lastWorkflowRun, nil
}

//ManualRun is the entry point to trigger a workflow manually
func ManualRun(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, e *sdk.WorkflowNodeRunManual) (*sdk.WorkflowRun, error) {
	number, err := nextRunNumber(db, w)
	if err != nil {
		return nil, sdk.WrapError(err, "ManualRun> Unable to get next number")
	}

	wr := &sdk.WorkflowRun{
		Number:       number,
		Workflow:     *w,
		WorkflowID:   w.ID,
		Start:        time.Now(),
		LastModified: time.Now(),
		ProjectID:    w.ProjectID,
		Status:       string(sdk.StatusWaiting),
	}
	wr.Tag(tagTriggeredBy, e.User.Username)

	if err := insertWorkflowRun(db, wr); err != nil {
		return nil, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
	}

	return wr, processWorkflowRun(db, store, p, wr, nil, e, nil)
}
