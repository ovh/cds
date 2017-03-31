package main

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

var logsecrets []sdk.Variable

func sendLog(pipJobID int64, value string, pipelineBuildID int64, stepOrder int, final bool) error {
	for i := range logsecrets {
		if len(logsecrets[i].Value) >= 6 {
			value = strings.Replace(value, logsecrets[i].Value, "**"+logsecrets[i].Name+"**", -1)
		}
	}

	l := sdk.NewLog(pipJobID, value, pipelineBuildID, stepOrder)
	if final {
		l.Done, _ = ptypes.TimestampProto(time.Now())
	} else {
		l.Done = &timestamp.Timestamp{}
	}
	logChan <- *l
	return nil
}

func logger(inputChan chan sdk.Log) {
	if grpcConn != nil {
		if err := grpcLogger(inputChan); err != nil {
			log.Critical("Unable to start grpc logger : %s", err)
		} else {
			return
		}
	}

	llist := list.New()
	for {
		select {
		case l, ok := <-inputChan:
			if ok {
				llist.PushBack(l)
			}
			break
		case <-time.After(1 * time.Second):

			var logs []*sdk.Log

			var currentStepLog *sdk.Log
			// While list is not empty
			for llist.Len() > 0 {
				// get older log line
				l := llist.Front().Value.(sdk.Log)
				llist.Remove(llist.Front())

				// then count how many lines are exactly the same
				count := 1
				for llist.Len() > 0 {
					n := llist.Front().Value.(sdk.Log)
					if string(n.Val) != string(l.Val) {
						break
					}
					count++
					llist.Remove(llist.Front())
				}

				// and if count > 1, then add it at the beginning of the log
				if count > 1 {
					l.Val = fmt.Sprintf("[x%d] %s", count, l.Val)
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
				// Buffer log list is empty, sending batch to API
				data, err := json.Marshal(l)
				if err != nil {
					fmt.Printf("Error: cannot marshal logs: %s\n", err)
					continue
				}

				path := fmt.Sprintf("/build/%d/log", l.PipelineBuildJobID)
				if _, _, err := sdk.Request("POST", path, data); err != nil {
					fmt.Printf("error: cannot send logs: %s\n", err)
					continue
				}
			}
		}
	}
}

func grpcLogger(inputChan chan sdk.Log) error {
	log.Notice("Logging through grpc")
	client := grpc.NewBuildLogClient(grpcConn)
	stream, err := client.AddBuildLog(context.Background())
	if err != nil {
		return err
	}

	for {
		l, ok := <-inputChan
		if ok {
			if err := stream.Send(&l); err != nil {
				log.Critical("grpcLogger> Error sending message : %s", err)
				//Close all
				stream.CloseSend()
				grpcConn.Close()
				//Try to reopen connection
				initGRPCConn()
				//restart the logger
				go logger(inputChan)
				//Reinject log
				inputChan <- l
				return nil
			}
		} else {
			break
		}
	}

	return stream.CloseSend()
}
