package main

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/plugin"
)

var logsecrets []sdk.Variable

func (wk *currentWorker) sendLog(buildID int64, value string, stepOrder int, final bool) error {
	for i := range logsecrets {
		if len(logsecrets[i].Value) >= 6 {
			value = strings.Replace(value, logsecrets[i].Value, "**"+logsecrets[i].Name+"**", -1)
		}
	}

	var id = wk.currentJob.pbJob.PipelineBuildID
	if wk.currentJob.wJob != nil {
		id = wk.currentJob.wJob.WorkflowNodeRunID
	}

	l := sdk.NewLog(buildID, value, id, stepOrder)
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
		l := wk.logger.llist.Front().Value.(sdk.Log)
		wk.logger.llist.Remove(wk.logger.llist.Front())

		// then count how many lines are exactly the same
		count := 1
		for wk.logger.llist.Len() > 0 {
			n := wk.logger.llist.Front().Value.(sdk.Log)
			if string(n.Val) != string(l.Val) {
				break
			}
			count++
			wk.logger.llist.Remove(wk.logger.llist.Front())
		}

		// and if count > 1, then add it at the beginning of the log
		if count > 1 {
			l.Val = fmt.Sprintf("[x%d]", count) + l.Val
		}
		// and append to the loerrorgs batch
		l.Val = strings.Trim(strings.Replace(l.Val, "\n", " ", -1), " \t\n") + "\n"

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

		var path string
		if wk.currentJob.wJob != nil {
			path = fmt.Sprintf("/queue/workflows/%d/log", wk.currentJob.wJob.ID)
		} else {
			path = fmt.Sprintf("/build/%d/log", l.PipelineBuildJobID)
		}

		if _, _, err := sdk.Request("POST", path, data); err != nil {
			log.Error("error: cannot send logs: %s", err)
			continue
		}
	}
}

func (wk *currentWorker) grpcLogger(ctx context.Context, inputChan chan sdk.Log) error {
	log.Info("Logging through grpc")

	stream, err := grpc.NewBuildLogClient(wk.grpc.conn).AddBuildLog(ctx)
	if err != nil {
		return err
	}

	streamWorkflow, err := grpc.NewWorkflowQueueClient(wk.grpc.conn).SendLog(ctx)
	if err != nil {
		return err
	}

	for {
		l, ok := <-inputChan
		if ok {

			log.Debug("LOG: %v", l.Val)
			var errSend error
			if wk.currentJob.wJob == nil {
				errSend = stream.Send(&l)
			} else {
				errSend = streamWorkflow.Send(&l)
			}

			if errSend != nil {
				log.Error("grpcLogger> Error sending message : %s", errSend)
				//Close all
				stream.CloseSend()
				streamWorkflow.CloseSend()
				wk.grpc.conn.Close()
				wk.grpc.conn = nil
				//Reinject log
				inputChan <- l
				return nil
			}
		} else {
			streamWorkflow.CloseSend()
			return stream.CloseSend()
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

func (wk *currentWorker) logHandler(w http.ResponseWriter, r *http.Request) {
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, errRead)
		writeError(w, r, newError)
		return
	}

	var pluginLog plugin.Log
	if err := json.Unmarshal(data, &pluginLog); err != nil {
		newError := sdk.NewError(sdk.ErrWrongRequest, err)
		writeError(w, r, newError)
		return
	}
	wk.sendLog(pluginLog.BuildID, pluginLog.Value, pluginLog.StepOrder, false)
}
