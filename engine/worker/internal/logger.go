package internal

import (
	"container/list"
	"context"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (wk *CurrentWorker) sendLog(buildID int64, value string, stepOrder int, final bool) error {
	now := time.Now()
	l := sdk.NewLog(buildID, wk.currentJob.wJob.WorkflowNodeRunID, value, stepOrder)
	if final {
		l.Done = &now
	}
	wk.logger.logChan <- *l
	return nil
}

func (wk *CurrentWorker) logProcessor(ctx context.Context, jobID int64) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer func() {
		ticker.Stop()
	}()

	wk.logger.llist = list.New()
	for {
		select {
		case l := <-wk.logger.logChan:
			wk.logger.llist.PushBack(l)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			wk.sendHTTPLog(ctx, jobID)
		}
	}
}

func (wk *CurrentWorker) sendHTTPLog(ctx context.Context, jobID int64) {
	var logs []*sdk.Log
	var currentStepLog *sdk.Log
	// While list is not empty
	for wk.logger.llist.Len() > 0 {
		// get older log line
		e := wk.logger.llist.Front()
		l := e.Value.(sdk.Log)
		wk.logger.llist.Remove(e)

		// First log
		if currentStepLog == nil {
			currentStepLog = &l
		} else if l.StepOrder == currentStepLog.StepOrder {
			currentStepLog.Val += l.Val
			currentStepLog.LastModified = l.LastModified
			currentStepLog.Done = l.Done
		} else {
			// new Step
			logs = append(logs, currentStepLog)
			currentStepLog = &l
		}
	}

	// insert last step
	if currentStepLog != nil {
		logs = append(logs, currentStepLog)
	}

	if len(logs) == 0 {
		return
	}

	for _, l := range logs {
		log.Debug("LOG: %v", l.Val)
		// TODO: stop the worker a nice way,
		// for the moment we are using context.Background and not the job context
		if err := wk.Client().QueueSendLogs(context.Background(), jobID, *l); err != nil {
			log.Error(ctx, "error: cannot send logs: %s", err)
			continue
		}
	}
}

func (wk *CurrentWorker) drainLogsAndCloseLogger(c context.Context) error {
	var i int
	for (len(wk.logger.logChan) > 0 || (wk.logger.llist != nil && wk.logger.llist.Len() > 0)) && i < 60 {
		log.Debug("Draining logs...")
		i++
		time.Sleep(1 * time.Second)
	}
	return c.Err()
}
