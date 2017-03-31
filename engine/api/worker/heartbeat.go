package worker

import (
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
)

// WorkerHeartbeatTimeout defines the number of seconds allowed for workers to refresh their beat
var WorkerHeartbeatTimeout = 1200.0

// Heartbeat runs in a goroutine and check last beat from all workers
func Heartbeat() {
	// If this goroutine exit, then it's a crash
	defer log.Fatalf("Goroutine of worker.Heartbeat exited - Exit CDS Engine")

	for {
		time.Sleep(10 * time.Second)
		if db := database.DB(); db != nil {
			w, err := LoadDeadWorkers(database.DBMap(db), WorkerHeartbeatTimeout)
			if err != nil {
				log.Warning("WorkerHeartbeat> Cannot load dead workers: %s", err)
				time.Sleep(10 * time.Second)
				continue
			}

			for i := range w {
				log.Info("WorkerHeartbeat> Delete worker %s[%s] LastBeat:%d hatchery:%d status:%s", w[i].Name, w[i].ID, w[i].LastBeat, w[i].HatcheryID, w[i].Status)
				if err = DeleteWorker(database.DBMap(db), w[i].ID); err != nil {
					log.Warning("WorkerHeartbeat> Cannot delete worker %s: %s", w[i].ID, err)
					continue
				}
			}
		}
	}
}
