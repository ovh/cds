package worker

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk/log"
)

// WorkerHeartbeatTimeout defines the number of seconds allowed for workers to refresh their beat
var WorkerHeartbeatTimeout = 300.0

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
					log.Debug("WorkerHeartbeat> Delete worker %s[%s] LastBeat:%v hatchery:%d status:%s", w[i].Name, w[i].ID, w[i].LastBeat, w[i].HatcheryID, w[i].Status)
					if errD := DeleteWorker(db, w[i].ID); errD != nil {
						log.Warning("WorkerHeartbeat> Cannot delete worker %s: %v", w[i].ID, errD)
						continue
					}
				}
			}
		}
	}
}
