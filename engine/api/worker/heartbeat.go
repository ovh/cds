package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk/log"
)

// WorkerHeartbeatTimeout defines the number of seconds allowed for workers to refresh their beat
var WorkerHeartbeatTimeout = 600.0

// CheckHeartbeat runs in a goroutine and check last beat from all workers
func CheckHeartbeat(c context.Context, DBFunc func(context.Context) *gorp.DbMap) {
	tick := time.NewTicker(10 * time.Second).C

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting WorkerHeartbeat: %v", c.Err())
			}
			return
		case <-tick:
			if db := DBFunc(c); db != nil {
				w, err := LoadDeadWorkers(db, WorkerHeartbeatTimeout)
				if err != nil {
					log.Warning("WorkerHeartbeat> Cannot load dead workers: %s", err)
					continue
				}
				for i := range w {
					log.Debug("WorkerHeartbeat> Delete worker %s[%s] LastBeat:%d hatchery:%d status:%s", w[i].Name, w[i].ID, w[i].LastBeat, w[i].HatcheryID, w[i].Status)
					if errD := DeleteWorker(db, w[i].ID); errD != nil {
						log.Warning("WorkerHeartbeat> Cannot delete worker %s: %s", w[i].ID, errD)
						continue
					}
				}
			}
		}
	}
}
