package workflow

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
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
func RunFromHook(dbCopy *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, e *sdk.WorkflowNodeRunHookEvent, chanEvent chan<- interface{}, asCodeMsg []sdk.Message) (*sdk.WorkflowRun, error) {
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
			return nil, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
		}

		//Process it
		hasRun, errWR := processWorkflowRun(dbCopy, db, store, p, wr, e, nil, nil, chanEvent)
		if errWR != nil {
			return nil, sdk.WrapError(errWR, "RunFromHook> Unable to process workflow run")
		}
		if !hasRun {
			wr.Status = sdk.StatusNeverBuilt.String()
			wr.LastExecution = time.Now()
			return wr, UpdateWorkflowRun(db, wr)
		}
	} else {

		//Load the last workflow run
		lastWorkflowRun, err := LoadLastRun(db, w.ProjectKey, w.Name, LoadRunOptions{})
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
		if _, err := processWorkflowRun(dbCopy, db, store, p, lastWorkflowRun, e, nil, &oldH.WorkflowNodeID, chanEvent); err != nil {
			return nil, sdk.WrapError(err, "RunFromHook> Unable to process workflow run")
		}
	}

	run, err := LoadRun(db, w.ProjectKey, w.Name, number, LoadRunOptions{})
	if err != nil {
		return nil, sdk.WrapError(err, "RunFromHook> Unable to reload workflow run")
	}

	return run, nil
}

//ManualRunFromNode is the entry point to trigger manually a piece of an existing run workflow
func ManualRunFromNode(dbCopy *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, number int64, e *sdk.WorkflowNodeRunManual, nodeID int64, chanEvent chan<- interface{}) (*sdk.WorkflowRun, error) {
	lastWorkflowRun, errLoadRun := LoadRun(db, w.ProjectKey, w.Name, number, LoadRunOptions{})
	if errLoadRun != nil {
		return nil, sdk.WrapError(errLoadRun, "ManualRunFromNode> Unable to load last run")
	}
	lastWorkflowRun.Tag(tagTriggeredBy, e.User.Username)

	condOk, err := processWorkflowRun(dbCopy, db, store, p, lastWorkflowRun, nil, e, &nodeID, chanEvent)
	if err != nil {
		return nil, sdk.WrapError(err, "ManualRunFromNode> Unable to process workflow run")
	}

	if !condOk {
		return nil, sdk.WrapError(sdk.ErrConditionsNotOk, "ManualRunFromNode> Conditions aren't ok")
	}

	var errLoadRunByID error
	lastWorkflowRun, errLoadRunByID = LoadRunByIDAndProjectKey(db, w.ProjectKey, lastWorkflowRun.ID, LoadRunOptions{})
	if errLoadRunByID != nil {
		return nil, errLoadRunByID
	}

	return lastWorkflowRun, nil
}

//ManualRun is the entry point to trigger a workflow manually
func ManualRun(dbCopy *gorp.DbMap, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, w *sdk.Workflow, e *sdk.WorkflowNodeRunManual, chanEvent chan<- interface{}, asCodeInfos []sdk.Message) (*sdk.WorkflowRun, error) {
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
		Status:       sdk.StatusWaiting.String(),
	}
	wr.Tag(tagTriggeredBy, e.User.Username)

	for _, msg := range asCodeInfos {
		AddWorkflowRunInfo(wr, false, sdk.SpawnMsg{ID: msg.ID, Args: msg.Args})
	}

	if err := insertWorkflowRun(db, wr); err != nil {
		return nil, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
	}

	hasRun, errWR := processWorkflowRun(dbCopy, db, store, p, wr, nil, e, nil, chanEvent)
	if errWR != nil {
		return wr, sdk.WrapError(errWR, "ManualRun")
	}
	if !hasRun {
		wr.Status = sdk.StatusNeverBuilt.String()
		return wr, UpdateWorkflowRun(db, wr)
	}
	return wr, nil
}

// GetTag return a specific tag from a list of tags
func GetTag(tags []sdk.WorkflowRunTag, tag string) sdk.WorkflowRunTag {
	for _, currentTag := range tags {
		if currentTag.Tag == tag {
			return currentTag
		}
	}

	return sdk.WorkflowRunTag{}
}
