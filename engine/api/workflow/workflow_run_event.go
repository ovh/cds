package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// GetWorkflowRunEventData read channel to get elements to push
func GetWorkflowRunEventData(cError <-chan error, cEvent <-chan interface{}) ([]sdk.WorkflowRun, []sdk.WorkflowNodeRun, []sdk.WorkflowNodeJobRun, error) {
	wrs := []sdk.WorkflowRun{}
	wnrs := []sdk.WorkflowNodeRun{}
	wnjrs := []sdk.WorkflowNodeJobRun{}
	var err error

	for {
		select {
		case e, has := <-cError:
			if e != nil {
				err = sdk.WrapError(e, "GetWorkflowRunEventData> Error received")
			}

			if !has {
				return wrs, wnrs, wnjrs, err
			}
		case w, has := <-cEvent:
			if !has {
				return wrs, wnrs, wnjrs, err
			}
			switch x := w.(type) {
			case sdk.WorkflowNodeJobRun:
				wnjrs = append(wnjrs, x)
			case sdk.WorkflowNodeRun:
				wnrs = append(wnrs, x)
			case sdk.WorkflowRun:
				wrs = append(wrs, x)
			default:
				log.Warning("GetWorkflowRunEventData> unknown type %T", w)
			}
		}
	}
}

// SendEvent Send event on workflow run
func SendEvent(db gorp.SqlExecutor, wrs []sdk.WorkflowRun, wnrs []sdk.WorkflowNodeRun, wnjrs []sdk.WorkflowNodeJobRun, key string) {
	for _, wr := range wrs {
		event.PublishWorkflowRun(wr, key)
	}
	for _, wnr := range wnrs {
		wr, errWR := LoadRunByID(db, wnr.WorkflowRunID, false)
		if errWR != nil {
			log.Warning("SendEvent.workflow> Cannot load workflow run %d: %s", wnr.WorkflowRunID, errWR)
			continue
		}

		var previousNodeRun sdk.WorkflowNodeRun
		if wnr.SubNumber > 0 {
			previousNodeRun = wnr
		} else {
			// Load previous run on current node
			node := wr.Workflow.GetNode(wnr.WorkflowNodeID)
			if node != nil {
				var errN error
				previousNodeRun, errN = PreviousNodeRun(db, wnr, *node, wr.WorkflowID)
				if errN != nil {
					log.Debug("SendEvent.workflow> Cannot load previous node run: %s", errN)
				}
			} else {
				log.Warning("SendEvent.workflow > Unable to find node %d in workflow", wnr.WorkflowNodeID)
			}
		}

		event.PublishWorkflowNodeRun(db, wnr, *wr, previousNodeRun, key)
	}
	for _, wnjr := range wnjrs {
		event.PublishWorkflowNodeJobRun(wnjr)
	}
}
