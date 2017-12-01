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

func (w *currentWorker) sendLog(buildID int64, value string, stepOrder int, final bool) error {
	for i := range logsecrets {
		if len(logsecrets[i].Value) >= 6 {
			value = strings.Replace(value, logsecrets[i].Value, "**"+logsecrets[i].Name+"**", -1)
		}
	}

	var id = w.currentJob.pbJob.PipelineBuildID
	if w.currentJob.wJob != nil {
		id = w.currentJob.wJob.WorkflowNodeRunID
	}

	l := sdk.NewLog(buildID, value, id, stepOrder)
	if final {
		l.Done, _ = ptypes.TimestampProto(time.Now())
	} else {
		l.Done, _ = ptypes.TimestampProto(time.Time{})
	}
	w.logger.logChan <- *l
	return nil
}

func (w *currentWorker) logProcessor(ctx context.Context) error {
	if w.grpc.conn != nil {
		if err := w.grpcLogger(ctx, w.logger.logChan); err != nil {
			log.Error("GPPC logger : %s", err)
		} else {
			return nil
		}
	} else {
		w.logger.llist = list.New()
		for {
			select {
			case l, ok := <-w.logger.logChan:
				if ok {
					w.logger.llist.PushBack(l)
				}
				break
			case <-time.After(250 * time.Millisecond):
				var logs []*sdk.Log
				var currentStepLog *sdk.Log
				// While list is not empty
				for w.logger.llist.Len() > 0 {
					// get older log line
					l := w.logger.llist.Front().Value.(sdk.Log)
					w.logger.llist.Remove(w.logger.llist.Front())

					// then count how many lines are exactly the same
					count := 1
					for w.logger.llist.Len() > 0 {
						n := w.logger.llist.Front().Value.(sdk.Log)
						if string(n.Val) != string(l.Val) {
							break
						}
						count++
						w.logger.llist.Remove(w.logger.llist.Front())
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
					continue
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
					if w.currentJob.wJob != nil {
						path = fmt.Sprintf("/queue/workflows/%d/log", w.currentJob.wJob.ID)
					} else {
						path = fmt.Sprintf("/build/%d/log", l.PipelineBuildJobID)
					}

					if _, _, err := sdk.Request("POST", path, data); err != nil {
						log.Error("error: cannot send logs: %s", err)
						continue
					}
				}
			}
		}
	}
	return nil
}

func (w *currentWorker) grpcLogger(ctx context.Context, inputChan chan sdk.Log) error {
	log.Info("Logging through grpc")

	stream, err := grpc.NewBuildLogClient(w.grpc.conn).AddBuildLog(ctx)
	if err != nil {
		return err
	}

	streamWorkflow, err := grpc.NewWorkflowQueueClient(w.grpc.conn).SendLog(ctx)
	if err != nil {
		return err
	}

	for {
		l, ok := <-inputChan
		if ok {

			log.Debug("LOG: %v", l.Val)
			var errSend error
			if w.currentJob.wJob == nil {
				errSend = stream.Send(&l)
			} else {
				errSend = streamWorkflow.Send(&l)
			}

			if errSend != nil {
				log.Error("grpcLogger> Error sending message : %s", errSend)
				//Close all
				stream.CloseSend()
				streamWorkflow.CloseSend()
				w.grpc.conn.Close()
				w.grpc.conn = nil
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

func (w *currentWorker) drainLogsAndCloseLogger(c context.Context) error {
	var i int
	for (len(w.logger.logChan) > 0 || (w.logger.llist != nil && w.logger.llist.Len() > 0)) && i < 60 {
		log.Debug("Draining logs...")
		i++
		time.Sleep(1 * time.Second)
	}
	return c.Err()
}
