package internal

import (
	"container/list"
	"context"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var logsecrets []sdk.Variable

func (wk *CurrentWorker) sendLog(buildID int64, value string, stepOrder int, final bool) error {
	for i := range logsecrets {
		if len(logsecrets[i].Value) >= sdk.SecretMinLength {
			value = strings.Replace(value, logsecrets[i].Value, "**"+logsecrets[i].Name+"**", -1)
		}
	}

	l := sdk.NewLog(buildID, value, wk.currentJob.wJob.WorkflowNodeRunID, stepOrder)
	if final {
		l.Done, _ = ptypes.TimestampProto(time.Now())
	} else {
		l.Done, _ = ptypes.TimestampProto(time.Time{})
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
			wk.sendHTTPLog(jobID)
		}
	}
}

func (wk *CurrentWorker) sendHTTPLog(jobID int64) {
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
		if err := wk.Client().QueueSendLogs(wk.currentJob.context, jobID, *l); err != nil {
			log.Error("error: cannot send logs: %s", err)
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
