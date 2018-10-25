package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// WorkerHeartbeatTimeout defines the number of seconds allowed for workers to refresh their beat
var WorkerHeartbeatTimeout = 300.0

// checkHeartbeat check last beat from all workers
func checkHeartbeat(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	w, err := worker.LoadDeadWorkers(db, WorkerHeartbeatTimeout)
	if err != nil {
		return sdk.WrapError(err, "Cannot load dead workers")
	}
	for i := range w {
		log.Debug("WorkerHeartbeat> Delete worker %s[%s] LastBeat:%v hatchery:%s status:%s", w[i].Name, w[i].ID, w[i].LastBeat, w[i].HatcheryName, w[i].Status)
		if w[i].Status.String() == sdk.StatusBuilding.String() && w[i].JobType == sdk.JobTypeWorkflowNode {
			wNodeJob, errL := LoadNodeJobRun(db, store, w[i].ActionBuildID)
			if errL != nil || wNodeJob.Retry > 3 {
				if errL != nil && !sdk.ErrorIs(errL, sdk.ErrWorkflowNodeRunJobNotFound) {
					log.Warning("WorkerHeartBeat> Cannot load node job run %d : %v", w[i].ActionBuildID, errL)
				}
				if errD := worker.SetStatus(db, w[i].ID, sdk.StatusDisabled); errD != nil {
					log.Warning("WorkerHeartbeat> Cannot disable worker %v: %v", w[i].ID, errD)
				}
				continue
			}

			tx, errTx := db.Begin()
			if errTx != nil {
				log.Warning("WorkerHeartBeat> Cannot create transaction %d : %v", w[i].ActionBuildID, errTx)
				continue
			}

			if err := RestartWorkflowNodeJob(ctx, db, *wNodeJob); err != nil {
				log.Warning("WorkerHeartBeat> Cannot restart node job run %d: %v", w[i].ActionBuildID, err)
				tx.Rollback()
				continue
			}

			if err := tx.Commit(); err != nil {
				log.Warning("WorkerHeartBeat> Cannot commit transaction %d: %v", w[i].ID, err)
			}
		} else {
			if err := worker.DeleteWorker(db, w[i].ID); err != nil {
				log.Warning("WorkerHeartbeat> Cannot delete worker %v: %v", w[i].ID, err)
			}
		}
	}

	return nil
}
