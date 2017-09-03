package workflow

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

//RunFromHook is the entry point to trigger a workflow from a hook
func RunFromHook(db gorp.SqlExecutor, w *sdk.Workflow, e *sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error) {
	return nil, nil
}

//ManualRunFromNode is the entry point to trigger manually a piece of an existing run workflow
func ManualRunFromNode(db gorp.SqlExecutor, w *sdk.Workflow, number int64, e *sdk.WorkflowNodeRunManual, nodeID int64) (*sdk.WorkflowRun, error) {
	lastWorkflowRun, err := LoadRun(db, w.ProjectKey, w.Name, number)
	if err != nil {
		return nil, sdk.WrapError(err, "ManualRunFromNode> Unable to load last run")
	}

	if err := processWorkflowRun(db, lastWorkflowRun, nil, e, &nodeID); err != nil {
		return nil, sdk.WrapError(err, "ManualRunFromNode> Unable to process workflow run")
	}

	lastWorkflowRun, err = LoadRunByIDAndProjectKey(db, w.ProjectKey, lastWorkflowRun.ID)
	if err != nil {
		return nil, err
	}

	return lastWorkflowRun, nil
}

//ManualRun is the entry point to trigger a workflow manually
func ManualRun(db gorp.SqlExecutor, w *sdk.Workflow, e *sdk.WorkflowNodeRunManual) (*sdk.WorkflowRun, error) {
	lastWorkflowRun, err := LoadLastRun(db, w.ProjectKey, w.Name)
	if err != nil {
		if err != sdk.ErrWorkflowNotFound {
			return nil, sdk.WrapError(err, "ManualRun> Unable to load last run")
		}
	}

	var number = int64(1)
	if lastWorkflowRun != nil {
		number = lastWorkflowRun.Number + 1
	}

	wr := &sdk.WorkflowRun{
		Number:       number,
		Workflow:     *w,
		WorkflowID:   w.ID,
		Start:        time.Now(),
		LastModified: time.Now(),
		ProjectID:    w.ProjectID,
	}

	if err := insertWorkflowRun(db, wr); err != nil {
		return nil, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
	}

	return wr, processWorkflowRun(db, wr, nil, e, nil)
}
