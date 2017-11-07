package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WorkerHeartbeatTimeout defines the number of seconds allowed for workers to refresh their beat
var WorkerHeartbeatTimeout = 600.0

// CheckHeartbeat runs in a goroutine and check last beat from all workers
func CheckHeartbeat(c context.Context, DBFunc func() *gorp.DbMap) {
	tick := time.NewTicker(10 * time.Second).C

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting WorkerHeartbeat: %v", c.Err())
			}
			return
		case <-tick:
			if db := DBFunc(); db != nil {
				w, err := LoadDeadWorkers(db, WorkerHeartbeatTimeout)
				if err != nil {
					log.Warning("WorkerHeartbeat> Cannot load dead workers: %s", err)
					continue
				}

				for i := range w {
					// Replace jobs in queue
					switch w[i].JobType {
					case sdk.JobTypePipeline:
					case sdk.JobTypeWorkflowNode:
						if errP := workflow.PutJobInQueue(db, w[i].ActionBuildID); errP != nil {
							log.Warning("WorkerHeartbeat> Cannot put job %d in queue: %s", w[i].ActionBuildID, errP)
						}
					}

					log.Debug("WorkerHeartbeat> Delete worker %s[%s] LastBeat:%d hatchery:%d status:%s", w[i].Name, w[i].ID, w[i].LastBeat, w[i].HatcheryID, w[i].Status)
					if err = DeleteWorker(db, w[i].ID); err != nil {
						log.Warning("WorkerHeartbeat> Cannot delete worker %s: %s", w[i].ID, err)
						continue
					}
				}
			}
		}
	}
}
