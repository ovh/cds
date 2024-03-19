package cdsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func (c *client) V2QueueJobStepUpdate(ctx context.Context, regionName string, jobRunID string, stepsStatus sdk.JobStepsStatus) error {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/step", regionName, jobRunID)
	if _, err := c.PostJSON(ctx, path, stepsStatus, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) V2QueuePushRunInfo(ctx context.Context, regionName string, jobRunID string, msg sdk.V2WorkflowRunInfo) error {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/runinfo", regionName, jobRunID)
	if _, err := c.PostJSON(ctx, path, msg, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) V2QueuePushJobInfo(ctx context.Context, regionName string, jobRunID string, msg sdk.V2SendJobRunInfo) error {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/info", regionName, jobRunID)
	if _, err := c.PostJSON(ctx, path, msg, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) V2QueueJobResult(ctx context.Context, regionName string, jobRunID string, result sdk.V2WorkflowRunJobResult) error {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/result", regionName, jobRunID)
	if _, err := c.PostJSON(ctx, path, result, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) V2QueueJobRunResultGet(ctx context.Context, regionName string, jobRunID string, runResultID string) (*sdk.V2WorkflowRunResult, error) {
	var result sdk.V2WorkflowRunResult
	path := fmt.Sprintf("/v2/queue/%s/job/%s/runresult/%s", regionName, jobRunID, runResultID)
	if _, err := c.GetJSON(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *client) V2QueueJobRunResultsGet(ctx context.Context, regionName string, jobRunID string) ([]sdk.V2WorkflowRunResult, error) {
	var result []sdk.V2WorkflowRunResult
	path := fmt.Sprintf("/v2/queue/%s/job/%s/runresult", regionName, jobRunID)
	if _, err := c.GetJSON(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *client) V2QueueJobRunResultCreate(ctx context.Context, regionName string, jobRunID string, result *sdk.V2WorkflowRunResult) error {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/runresult", regionName, jobRunID)
	if _, err := c.PostJSON(ctx, path, result, result); err != nil {
		return err
	}
	return nil
}

func (c *client) V2QueueJobRunResultUpdate(ctx context.Context, regionName string, jobRunID string, result *sdk.V2WorkflowRunResult) error {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/runresult", regionName, jobRunID)
	if _, err := c.PutJSON(ctx, path, result, result); err != nil {
		return err
	}
	return nil
}

// V2HatcheryTakeJob job status pssed to crafting and other hatcheries cannot work on it
func (c *client) V2HatcheryTakeJob(ctx context.Context, regionName string, jobRunID string) (*sdk.V2WorkflowRunJob, error) {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/hatchery/take", regionName, jobRunID)
	var jobRun sdk.V2WorkflowRunJob
	if _, err := c.PostJSON(ctx, path, nil, &jobRun); err != nil {
		return nil, err
	}
	return &jobRun, nil
}

func (c *client) V2HatcheryReleaseJob(ctx context.Context, regionName string, jobRunID string) error {
	path := fmt.Sprintf("/v2/queue/%s/job/%s/hatchery/take", regionName, jobRunID)
	if _, err := c.DeleteJSON(ctx, path, nil); err != nil {
		return err
	}
	return nil
}

// V2QueueGetJobRun returns information about a job
func (c *client) V2QueueGetJobRun(ctx context.Context, regionName, id string) (*sdk.V2QueueJobInfo, error) {
	path := fmt.Sprintf("/v2/queue/%s/job/%s", regionName, id)
	var job sdk.V2QueueJobInfo
	if _, err := c.GetJSON(ctx, path, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *client) V2QueuePolling(ctx context.Context, regionName string, goRoutines *sdk.GoRoutines, hatcheryMetrics *sdk.HatcheryMetrics, pendingWorkerCreation *sdk.HatcheryPendingWorkerCreation, jobs chan<- sdk.V2QueueJobInfo, errs chan<- error, delay time.Duration, ms ...RequestModifier) error {
	jobsTicker := time.NewTicker(delay)
	ctx, cancel := context.WithCancel(ctx)

	// This goroutine call the Websocket
	chanMessageReceived := make(chan sdk.WebsocketJobQueueEvent, 10)
	goRoutines.Exec(ctx, "RequestWebsocketHatchery", func(ctx context.Context) {
		c.WebsocketHatcheryJobQueuedListen(ctx, goRoutines, chanMessageReceived, errs)
		cancel()
	})

	for {
		select {
		case <-ctx.Done():
			jobsTicker.Stop()
			if jobs != nil {
				close(jobs)
			}
			return ctx.Err()
		case wsEvent := <-chanMessageReceived:
			if jobs == nil {
				continue
			}
			telemetry.Record(ctx, hatcheryMetrics.JobReceivedInQueuePollingWSv2, 1)
			j, err := c.V2QueueGetJobRun(ctx, wsEvent.Region, wsEvent.JobRunID)
			// Do not log the error if the job does not exist
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				continue
			}
			if err != nil {
				errs <- newError(fmt.Errorf("unable to get job %s info: %v", wsEvent.JobRunID, err))
				continue
			}
			// push the job in the channel
			if j.RunJob.Status == sdk.StatusWaiting {
				if pendingWorkerCreation.IsJobAlreadyPendingWorkerCreation(wsEvent.JobRunID) {
					log.Debug(ctx, "skipping job %s", wsEvent.JobRunID)
					continue
				}
				lenqueue := pendingWorkerCreation.SetJobInPendingWorkerCreation(wsEvent.JobRunID)
				log.Debug(ctx, "v2_len_queue: %v", lenqueue)
				telemetry.Record(ctx, hatcheryMetrics.ChanV2JobAdd, 1)
				jobs <- *j
			}
		case <-jobsTicker.C:
			if jobs == nil {
				continue
			}

			ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
			var queue []sdk.V2WorkflowRunJob
			if _, err := c.GetJSON(ctxt, "/v2/queue/"+regionName, &queue); err != nil && !sdk.ErrorIs(err, sdk.ErrUnauthorized) {
				errs <- newError(fmt.Errorf("unable to load jobs: %v", err))
				cancel()
				continue
			} else if sdk.ErrorIs(err, sdk.ErrUnauthorized) {
				cancel()
				continue
			}
			cancel()

			queueFiltered := []sdk.V2WorkflowRunJob{}
			for _, job := range queue {
				if pendingWorkerCreation.IsJobAlreadyPendingWorkerCreation(job.ID) {
					log.Debug(ctx, "skipping job %s", job.ID)
					continue
				}
				queueFiltered = append(queueFiltered, job)
			}

			log.Debug(ctx, "v2_job_queue_from_api: %v job_queue_filtered: %v len_queue: %v", len(queue), len(queueFiltered), pendingWorkerCreation.NbJobInPendingWorkerCreation())

			max := cap(jobs) * 2
			if len(queueFiltered) < max {
				max = len(queueFiltered)
			}
			for i := 0; i < max; i++ {
				pendingWorkerCreation.SetJobInPendingWorkerCreation(queueFiltered[i].ID)
				telemetry.Record(ctx, hatcheryMetrics.ChanV2JobAdd, 1)

				jobInfo, err := c.V2QueueGetJobRun(ctx, regionName, queueFiltered[i].ID)
				if err != nil {
					return err
				}
				jobs <- *jobInfo
			}
		}
	}
}

func (c *client) WebsocketHatcheryJobQueuedListen(ctx context.Context, goRoutines *sdk.GoRoutines, chanEventReceived chan<- sdk.WebsocketJobQueueEvent, chanErrorReceived chan<- error) {
	chanMsgReceived := make(chan json.RawMessage)
	goRoutines.Exec(ctx, "WebsocketHatcheryJobQueuedListen", func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.ErrorWithStackTrace(ctx, ctx.Err())
				}
				return
			case m := <-chanMsgReceived:
				var wsEvent sdk.WebsocketJobQueueEvent
				if err := sdk.JSONUnmarshal(m, &wsEvent); err != nil {
					chanErrorReceived <- newError(fmt.Errorf("unable to unmarshal message: %s: %v", string(m), err))
					continue
				}
				chanEventReceived <- wsEvent
			}
		}
	})

	for ctx.Err() == nil {
		if err := c.RequestWebsocket(ctx, goRoutines, "/v2/hatchery/ws", nil, chanMsgReceived, chanErrorReceived); err != nil {
			chanErrorReceived <- newError(fmt.Errorf("websocket error: %v", err))
		}
		time.Sleep(1 * time.Second)
	}
}
