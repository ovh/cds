package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func (c *client) QueuePolling(ctx context.Context, jobs chan<- sdk.WorkflowNodeJobRun, pbjobs chan<- sdk.PipelineBuildJob, errs chan<- error, delay time.Duration) error {
	defer c.WorkerSetStatus(sdk.StatusWaiting)

	t0 := time.Unix(0, 0)
	jobsTicker := time.NewTicker(delay)
	pbjobsTicker := time.NewTicker(delay)
	oldJobsTicker := time.NewTicker(delay * 60)

	for {
		select {
		case <-ctx.Done():
			jobsTicker.Stop()
			pbjobsTicker.Stop()
			oldJobsTicker.Stop()
			if jobs != nil {
				close(jobs)
			}
			if pbjobs != nil {
				close(pbjobs)
			}
			return ctx.Err()
		case <-oldJobsTicker.C:
			if jobs != nil {
				queue := []sdk.WorkflowNodeJobRun{}
				if _, err := c.GetJSON("/queue/workflows", &queue); err != nil {
					errs <- sdk.WrapError(err, "Unable to load old jobs")
				}
				t0 = time.Now()
				for _, j := range queue {
					jobs <- j
				}
			}
		case <-jobsTicker.C:
			if jobs != nil {
				queue := []sdk.WorkflowNodeJobRun{}
				if _, err := c.GetJSON("/queue/workflows", &queue, SetHeader("If-Modified-Since", t0.Format(time.RFC1123))); err != nil {
					errs <- sdk.WrapError(err, "Unable to load jobs")
				}
				t0 = time.Now()
				for _, j := range queue {
					fmt.Printf("Job in queue :%+v\n", j)
					jobs <- j
				}
			}
		case <-pbjobsTicker.C:
			if pbjobs != nil {
				queue, err := sdk.GetBuildQueue()
				if err != nil {
					errs <- sdk.WrapError(err, "Unable to load pipeline build jobs")
				}
				for _, j := range queue {
					pbjobs <- j
				}
			}
		}
	}
}

func (c *client) QueueTakeJob(job sdk.WorkflowNodeJobRun, isBooked bool) (*worker.WorkflowNodeJobRunInfo, error) {
	in := worker.TakeForm{Time: time.Now()}
	if isBooked {
		in.BookedJobID = job.ID
	}

	var path = fmt.Sprintf("/queue/workflows/%d/take", job.ID)
	var info worker.WorkflowNodeJobRunInfo

	if code, err := c.PostJSON(path, &in, &info); err != nil {
		return nil, err
	} else if code != http.StatusOK {
		return nil, nil
	}

	return &info, nil
}

func (c *client) QueueJobInfo(id int64) (*sdk.WorkflowNodeJobRun, error) {
	var path = fmt.Sprintf("/queue/workflows/%d/infos", id)
	var job sdk.WorkflowNodeJobRun

	if code, err := c.PostJSON(path, nil, &job); err != nil {
		return nil, err
	} else if code != http.StatusOK {
		return nil, nil
	}
	return &job, nil
}
