package workflow

import (
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"

	"github.com/go-gorp/gorp"
)

func RunFromHook(db gorp.SqlExecutor, w *sdk.Workflow, e *sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error) {
	return nil, nil
}

func ManualRun(db gorp.SqlExecutor, w *sdk.Workflow, e *sdk.WorkflowNodeRunManual) (*sdk.WorkflowRun, error) {
	wr := &sdk.WorkflowRun{
		Number:       0, //Get last number
		WorkflowName: w.Name,
		Workflow:     *w,
		Start:        time.Now(),
		LastModified: time.Now(),
		ProjectKey:   w.ProjectKey,
		ProjectID:    w.ProjectID,
	}

	if err := insertWorkflowRun(db, wr); err != nil {
		return nil, sdk.WrapError(err, "ManualRun> Unable to manually run workflow %s/%s", w.ProjectKey, w.Name)
	}
	return wr, run(db, wr)
}

func run(db gorp.SqlExecutor, w *sdk.WorkflowRun) error {
	cache.Enqueue(processWorkflowQueue, w)
	return nil
}
