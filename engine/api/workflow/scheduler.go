package workflow

import (
	"context"

	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const queueWorkflowNodeRun = "queue:workflow:node:run"

var (
	chanWorkflowNodeRun = make(chan *sdk.WorkflowNodeRun)
)

func dequeueWorkflows(c context.Context) {
	for {
		run := &sdk.WorkflowNodeRun{}
		cache.DequeueWithContext(queueWorkflowNodeRun, run, c)
		if run != nil {
			chanWorkflowNodeRun <- run
		}
	}
}

func Scheduler(c context.Context) error {
	t0 := time.Now()

	log.Debug("workflow.Scheduler> Begin")
	defer func() {
		log.Debug("workflow.Scheduler> End (%.3fs)", time.Since(t0).Seconds())
	}()

	go dequeueWorkflows(c)
	for {
		select {
		case <-c.Done():
			err := c.Err()
			if err != nil {
				log.Error("Exiting workflow.Scheduler: %s", err)
			}
			return err
		case n := <-chanWorkflowNodeRun:
			log.Debug("%#v", n)
			db := database.GetDBMap()
			if err := execute(db, n); err != nil {
				log.Error("Error workflow.Scheduler executing node: %s", err)
			}
		}
	}
}
