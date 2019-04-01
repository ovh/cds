package main

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var logsecrets []sdk.Variable

func (wk *currentWorker) sendLog(buildID int64, value string, stepOrder int, final bool) error {
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

func (wk *currentWorker) logProcessor(ctx context.Context) error {
	if wk.grpc.conn != nil {
		if err := wk.grpcLogger(ctx, wk.logger.logChan); err != nil {
			log.Error("GPPC logger : %s", err)
		} else {
			return nil
		}
	} else {
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
				wk.sendHTTPLog()
			}
		}
	}

	return nil
}

func (wk *currentWorker) sendHTTPLog() {
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
		// Buffer log list is empty, sending batch to API
		data, err := json.Marshal(l)
		if err != nil {
			log.Error("Error: cannot marshal logs: %s", err)
			continue
		}

		path := fmt.Sprintf("/queue/workflows/%d/log", wk.currentJob.wJob.ID)

		if _, _, err := sdk.Request("POST", path, data); err != nil {
			log.Error("error: cannot send logs: %s", err)
			continue
		}
	}
}

func (wk *currentWorker) grpcLogger(ctx context.Context, inputChan chan sdk.Log) error {
	log.Info("Logging through grpc")

	streamWorkflow, err := grpc.NewWorkflowQueueClient(wk.grpc.conn).SendLog(ctx)
	if err != nil {
		return err
	}

	for {
		l, ok := <-inputChan
		if ok {

			log.Debug("LOG: %v", l.Val)
			var errSend error

			errSend = streamWorkflow.Send(&l)

			if errSend != nil {
				log.Error("grpcLogger> Error sending message : %s", errSend)
				//Close all
				streamWorkflow.CloseSend()
				wk.grpc.conn.Close()
				wk.grpc.conn = nil
				//Reinject log
				inputChan <- l
				return nil
			}
		} else {
			return streamWorkflow.CloseSend()
		}
	}
}

func (wk *currentWorker) drainLogsAndCloseLogger(c context.Context) error {
	var i int
	for (len(wk.logger.logChan) > 0 || (wk.logger.llist != nil && wk.logger.llist.Len() > 0)) && i < 60 {
		log.Debug("Draining logs...")
		i++
		time.Sleep(1 * time.Second)
	}
	return c.Err()
}
