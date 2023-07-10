package hatchery

import (
	"context"
	"github.com/ovh/cds/sdk/telemetry"
	"sync/atomic"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

var (
	nbWorkerToStart int64
)

func checkCapacities(ctx context.Context, h Interface) bool {
	ctx, end := telemetry.Span(ctx, "hatchery.checkCapacities")
	defer end()
	t := time.Now()
	defer log.Debug(ctx, "hatchery> checkCapacities> %.3f seconds elapsed", time.Since(t).Seconds())

	workerPool, err := WorkerPool(ctx, h, sdk.StatusChecking, sdk.StatusWaiting, sdk.StatusBuilding, sdk.StatusWorkerPending, sdk.StatusWorkerRegistering)
	if err != nil {
		log.Error(ctx, "hatchery> checkCapacities> Pool> Error: %v", err)
		return false
	}

	if len(workerPool) >= h.Configuration().Provision.MaxWorker {
		log.Debug(ctx, "hatchery> checkCapacities> %s has reached the max worker: %d (max: %d)", h.Service().Name, len(workerPool), h.Configuration().Provision.MaxWorker)
		if len(workerPool) > h.Configuration().Provision.MaxWorker {
			for _, w := range workerPool {
				log.Debug(ctx, "hatchery> checkCapacities> %s > pool > %s (status=%v)", h.Service().Name, w.Name, w.Status)
			}
		}
		return false
	}

	var nbPending int
	for _, w := range workerPool {
		if w.Status == sdk.StatusWorkerPending {
			nbPending++
		}
	}

	maxProv := h.Configuration().Provision.MaxConcurrentProvisioning
	if maxProv < 1 {
		maxProv = defaultMaxProvisioning
	}

	if nbPending >= maxProv {
		log.Info(ctx, "hatchery> checkCapacities> too many pending worker in pool: %d", nbPending)
		return false
	}

	if int(atomic.LoadInt64(&nbWorkerToStart)) >= maxProv {
		log.Info(ctx, "hatchery> checkCapacities> too many starting worker in pool: %d", atomic.LoadInt64(&nbWorkerToStart))
		return false
	}

	return true
}
