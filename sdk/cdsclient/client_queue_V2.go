package cdsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

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
func (c *client) V2QueueGetJobRun(ctx context.Context, regionName, id string) (*sdk.V2WorkflowRunJob, error) {
	path := fmt.Sprintf("/v2/queue/%s/job/%s", regionName, id)
	var job sdk.V2WorkflowRunJob
	if _, err := c.GetJSON(ctx, path, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (c *client) V2QueuePolling(ctx context.Context, regionName string, goRoutines *sdk.GoRoutines, jobs chan<- sdk.V2WorkflowRunJob, errs chan<- error, delay time.Duration, ms ...RequestModifier) error {
	jobsTicker := time.NewTicker(delay)

	// This goroutine call the Websocket
	chanMessageReceived := make(chan sdk.WebsocketHatcheryEvent, 10)
	goRoutines.Exec(ctx, "RequestWebsocketHatchery", func(ctx context.Context) {
		c.WebsocketHatcheryJobQueuedListen(ctx, goRoutines, chanMessageReceived, errs)
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
			j, err := c.V2QueueGetJobRun(ctx, wsEvent.Event.Region, wsEvent.Event.JobRunID)
			// Do not log the error if the job does not exist
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				continue
			}
			if err != nil {
				errs <- newError(fmt.Errorf("unable to get job %s info: %v", wsEvent.Event.JobRunID, err))
				continue
			}
			// push the job in the channel
			if j.Status == sdk.StatusWaiting {
				jobs <- *j
			}
		case <-jobsTicker.C:
			if c.config.Verbose {
				fmt.Println("jobsTicker")
			}

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
			if c.config.Verbose {
				fmt.Println("Jobs Queue size: ", len(queue))
			}

			max := cap(jobs) * 2
			if len(queue) < max {
				max = len(queue)
			}
			for i := 0; i < max; i++ {
				jobs <- queue[i]
			}

		}
	}
}

func (c *client) WebsocketHatcheryJobQueuedListen(ctx context.Context, goRoutines *sdk.GoRoutines, chanEventReceived chan<- sdk.WebsocketHatcheryEvent, chanErrorReceived chan<- error) {
	chanMsgReceived := make(chan json.RawMessage)
	goRoutines.Exec(ctx, "WebsocketHatcheryJobQueuedListen", func(ctx context.Context) {
		for {
			select {
			case m := <-chanMsgReceived:
				var wsEvent sdk.WebsocketHatcheryEvent
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
