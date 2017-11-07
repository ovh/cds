package workflow

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func GetWorkflowRunEventData(cError <-chan error, cEvent <-chan interface{}) ([]sdk.WorkflowRun, []sdk.WorkflowNodeRun, []sdk.WorkflowNodeJobRun, error) {
	wrs := []sdk.WorkflowRun{}
	wnrs := []sdk.WorkflowNodeRun{}
	wnjrs := []sdk.WorkflowNodeJobRun{}

	for {
		select {
		case e, has := <-cError:
			if !has {
				return wrs, wnrs, wnjrs, nil
			}
			if e != nil {
				return nil, nil, nil, e
			}
		case w, has := <-cEvent:
			if !has {
				return wrs, wnrs, wnjrs, nil
			}
			switch x := w.(type) {
			case sdk.WorkflowNodeJobRun:
				wnjrs = append(wnjrs, x)
			case sdk.WorkflowNodeRun:
				wnrs = append(wnrs, x)
			case sdk.WorkflowRun:
				wrs = append(wrs, x)
			}
		}
	}
	return wrs, wnrs, wnjrs, nil
}

// SendEvent Send event on workflow run
func SendEvent(db gorp.SqlExecutor, wrs []sdk.WorkflowRun, wnrs []sdk.WorkflowNodeRun, wnjrs []sdk.WorkflowNodeJobRun, key string) {
	for _, wr := range wrs {
		event.PublishWorkflowRun(wr, key)
	}
	for _, wnr := range wnrs {
		wr, errWR := LoadRunByID(db, wnr.WorkflowRunID)
		if errWR != nil {
			log.Warning("SendEvent> Cannot load workflow run %d: %s", wnr.WorkflowRunID, errWR)
			continue
		}
		event.PublishWorkflowNodeRun(wnr, *wr, key)
	}
	for _, wnjr := range wnjrs {
		event.PublishWorkflowNodeJobRun(wnjr)
	}
}
