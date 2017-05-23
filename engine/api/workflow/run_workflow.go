package workflow

import (
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
)

func RunFromHook(db gorp.SqlExecutor, w *sdk.Workflow, e *sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error) {
	return nil, nil
}

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

	log.Debug("workflow.ManualRun> %#v", wr)

	return wr, processWorkflowRun(db, wr, nil, e, nil)
}
