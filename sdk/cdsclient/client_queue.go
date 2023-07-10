package cdsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"
)

// shrinkQueue is used to shrink the polled queue 200% of the channel capacity (l)
// it returns as reference date the date of the last element in the shrinkked queue
func shrinkQueue(queue *sdk.WorkflowQueue, nbJobsToKeep int) time.Time {
	if len(*queue) == 0 {
		return time.Time{}
	}

	if nbJobsToKeep < 1 {
		nbJobsToKeep = 1
	}

	// nbJobsToKeep is by default the concurrent max worker provisionning.
	// we keep 2x this number
	nbJobsToKeep = nbJobsToKeep * 2

	queue.Sort()

	if len(*queue) > nbJobsToKeep {
		newQueue := (*queue)[:nbJobsToKeep]
		*queue = newQueue
	}
	t0 := time.Now()
	for _, q := range *queue {
		if q.Queued.Before(t0) {
			t0 = q.Queued
		}
	}
	return t0
}

func (c *client) QueuePolling(ctx context.Context, goRoutines *sdk.GoRoutines, jobs chan<- sdk.WorkflowNodeJobRun, errs chan<- error, delay time.Duration, ms ...RequestModifier) error {
	jobsTicker := time.NewTicker(delay)

	// This goroutine call the SSE route
	chanMessageReceived := make(chan sdk.WebsocketEvent, 10)
	chanMessageToSend := make(chan []sdk.WebsocketFilter, 10)
	goRoutines.Exec(ctx, "RequestWebsocket", func(ctx context.Context) {
		c.WebsocketEventsListen(ctx, goRoutines, chanMessageToSend, chanMessageReceived, errs)
	})
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type: sdk.WebsocketFilterTypeQueue,
	}}

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
			if wsEvent.Event.EventType == "sdk.EventRunWorkflowJob" && wsEvent.Event.Status == sdk.StatusWaiting {
				var jobEvent sdk.EventRunWorkflowJob
				if err := sdk.JSONUnmarshal(wsEvent.Event.Payload, &jobEvent); err != nil {
					errs <- newError(fmt.Errorf("unable to unmarshal job %v: %v", wsEvent.Event.Payload, err))
					continue
				}
				job, err := c.QueueJobInfo(ctx, strconv.FormatInt(jobEvent.ID, 10))
				// Do not log the error if the job does not exist
				if sdk.ErrorIs(err, sdk.ErrWorkflowNodeRunJobNotFound) {
					continue
				}

				if err != nil {
					errs <- newError(fmt.Errorf("unable to get job %v info: %v", jobEvent.ID, err))
					continue
				}
				// push the job in the channel
				if job.Status == sdk.StatusWaiting && job.BookedBy.Name == "" {
					job.Header["WS"] = "true"
					jobs <- *job
				}
			}
		case <-jobsTicker.C:
			if c.config.Verbose {
				fmt.Println("jobsTicker")
			}

			if jobs == nil {
				continue
			}

			ctxt, cancel := context.WithTimeout(ctx, 10*time.Second)
			queue := sdk.WorkflowQueue{}
			if _, err := c.GetJSON(ctxt, "/queue/workflows", &queue, ms...); err != nil && !sdk.ErrorIs(err, sdk.ErrUnauthorized) {
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

			shrinkQueue(&queue, cap(jobs))
			for _, j := range queue {
				jobs <- j
			}
		}
	}
}

func (c *client) QueueWorkflowNodeJobRun(ms ...RequestModifier) ([]sdk.WorkflowNodeJobRun, error) {
	wJobs := []sdk.WorkflowNodeJobRun{}
	url, _ := url.Parse("/queue/workflows")
	if _, err := c.GetJSON(context.Background(), url.String(), &wJobs, ms...); err != nil {
		return nil, err
	}
	return wJobs, nil
}

func (c *client) QueueCountWorkflowNodeJobRun(since *time.Time, until *time.Time, modelType string) (sdk.WorkflowNodeJobRunCount, error) {
	if since == nil {
		since = new(time.Time)
	}
	if until == nil {
		now := time.Now()
		until = &now
	}
	url, _ := url.Parse("/queue/workflows/count")
	q := url.Query()
	if modelType != "" {
		q.Add("modelType", modelType)
	}
	url.RawQuery = q.Encode()

	countWJobs := sdk.WorkflowNodeJobRunCount{}
	_, _, err := c.GetJSONWithHeaders(url.String(),
		&countWJobs,
		SetHeader(RequestedIfModifiedSinceHeader, since.Format(time.RFC1123)),
		SetHeader("X-CDS-Until", until.Format(time.RFC1123)))
	return countWJobs, err
}

func (c *client) QueueTakeJob(ctx context.Context, job sdk.WorkflowNodeJobRun) (*sdk.WorkflowNodeJobRunData, error) {
	path := fmt.Sprintf("/queue/workflows/%d/take", job.ID)
	var info sdk.WorkflowNodeJobRunData
	if _, err := c.PostJSON(ctx, path, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// QueueJobInfo returns information about a job
func (c *client) QueueJobInfo(ctx context.Context, id string) (*sdk.WorkflowNodeJobRun, error) {
	path := fmt.Sprintf("/queue/workflows/%s/infos", id)
	var job sdk.WorkflowNodeJobRun

	if _, err := c.GetJSON(context.Background(), path, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

// QueueJobSendSpawnInfo sends a spawn info on a job
func (c *client) QueueJobSendSpawnInfo(ctx context.Context, id string, in []sdk.SpawnInfo) error {
	path := fmt.Sprintf("/queue/workflows/%s/spawn/infos", id)
	_, err := c.PostJSON(ctx, path, &in, nil)
	return err
}

// QueueJobBook books a job for a Hatchery
func (c *client) QueueJobBook(ctx context.Context, id string) (sdk.WorkflowNodeJobRunBooked, error) {
	var resp sdk.WorkflowNodeJobRunBooked
	path := fmt.Sprintf("/queue/workflows/%s/book", id)
	_, err := c.PostJSON(ctx, path, nil, &resp)
	return resp, err
}

func (c *client) QueueWorkflowRunResultsAdd(ctx context.Context, jobID int64, addRequest sdk.WorkflowRunResult) error {
	uri := fmt.Sprintf("/queue/workflows/%d/run/results", jobID)
	if _, err := c.PostJSON(ctx, uri, addRequest, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) QueueWorkflowRunResultCheck(ctx context.Context, jobID int64, runResultCheck sdk.WorkflowRunResultCheck) (int, error) {
	uri := fmt.Sprintf("/queue/workflows/%d/run/results/check", jobID)
	code, err := c.PostJSON(ctx, uri, runResultCheck, nil)
	return code, err
}

// QueueJobRelease release a job for a worker
func (c *client) QueueJobRelease(ctx context.Context, id string) error {
	path := fmt.Sprintf("/queue/workflows/%s/book", id)
	_, err := c.DeleteJSON(context.Background(), path, nil)
	return err
}

func (c *client) QueueSendResult(ctx context.Context, id int64, res sdk.Result) error {
	path := fmt.Sprintf("/queue/workflows/%d/result", id)
	b, err := json.Marshal(res)
	if err != nil {
		return newError(err)
	}
	result, _, code, err := c.Stream(ctx, c.HTTPNoTimeoutClient(), "POST", path, bytes.NewBuffer(b), nil)
	if err != nil {
		return err
	}
	defer result.Close()
	if code >= 300 {
		return newError(fmt.Errorf("unable to send job result. HTTP code error : %d", code))
	}
	return nil
}

func (c *client) QueueSendUnitTests(ctx context.Context, id int64, report sdk.JUnitTestsSuites) error {
	path := fmt.Sprintf("/queue/workflows/%d/test", id)
	_, err := c.PostJSON(ctx, path, report, nil)
	return err
}

func (c *client) QueueSendVulnerability(ctx context.Context, id int64, report sdk.VulnerabilityWorkerReport) error {
	path := fmt.Sprintf("/queue/workflows/%d/vulnerability", id)
	_, err := c.PostJSON(ctx, path, report, nil)
	return err
}

func (c *client) QueueSendStepResult(ctx context.Context, id int64, res sdk.StepStatus) error {
	path := fmt.Sprintf("/queue/workflows/%d/step", id)
	_, err := c.PostJSON(ctx, path, res, nil)
	return err
}

func (c *client) QueueWorkerCacheLink(ctx context.Context, jobID int64, tag string) (sdk.CDNItemLinks, error) {
	var result sdk.CDNItemLinks
	path := fmt.Sprintf("/queue/workflows/%d/cache/%s/links", jobID, tag)
	_, err := c.GetJSON(ctx, path, &result, nil)
	return result, err
}

func (c *client) QueueJobTag(ctx context.Context, jobID int64, tags []sdk.WorkflowRunTag) error {
	path := fmt.Sprintf("/queue/workflows/%d/tag", jobID)
	_, err := c.PostJSON(ctx, path, tags, nil)
	return err
}

func (c *client) QueueJobSetVersion(ctx context.Context, jobID int64, version sdk.WorkflowRunVersion) error {
	path := fmt.Sprintf("/queue/workflows/%d/version", jobID)
	_, err := c.PostJSON(ctx, path, version, nil)
	return err
}

func (c *client) QueueWorkflowRunResultsRelease(ctx context.Context, permJobID int64, runResultIDs []string, to string) error {
	req := sdk.WorkflowRunResultPromotionRequest{
		IDs:        runResultIDs,
		ToMaturity: to,
	}
	uri := fmt.Sprintf("/queue/workflows/%d/run/results/release", permJobID)
	_, err := c.PostJSON(ctx, uri, req, nil)
	return err
}

func (c *client) QueueWorkflowRunResultsPromote(ctx context.Context, permJobID int64, runResultIDs []string, to string) error {
	req := sdk.WorkflowRunResultPromotionRequest{
		IDs:        runResultIDs,
		ToMaturity: to,
	}
	uri := fmt.Sprintf("/queue/workflows/%d/run/results/promote", permJobID)
	_, err := c.PostJSON(ctx, uri, req, nil)
	return err
}
