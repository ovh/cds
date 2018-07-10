package hatchery

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	checkProvisioningMutex = sync.Mutex{}
	nbWorkerToStart        int64
)

func checkProvisioning(h Interface) bool {
	t := time.Now()
	defer log.Debug("checkProvisioning> %d ns elapsed", time.Since(t).Nanoseconds())

	checkProvisioningMutex.Lock()
	defer checkProvisioningMutex.Unlock()

	workerPool, err := WorkerPool(h, sdk.StatusChecking, sdk.StatusWaiting, sdk.StatusBuilding, sdk.StatusWorkerPending, sdk.StatusWorkerRegistering)
	if err != nil {
		log.Error("hatchery> Pool> Error: %v", err)
		return false
	}

	if len(workerPool) >= h.Configuration().Provision.MaxWorker {
		log.Debug("hatchery> %s has reached the max worker: %d (max: %d)", h.Hatchery().Name, len(workerPool), h.Configuration().Provision.MaxWorker)
		if len(workerPool) > h.Configuration().Provision.MaxWorker {
			for _, w := range workerPool {
				log.Debug("hatchery> %s > pool > %s (status=%v)", h.Hatchery().Name, w.Name, w.Status)
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
	if nbPending+int(atomic.LoadInt64(&nbWorkerToStart)) > maxProv {
		log.Info("hatchery> too many pending worker in pool: %d", nbPending)
		return false
	}

	return true
}

func provisioning(h Interface, provisionDisabled bool, models []sdk.Model) {
	if provisionDisabled {
		log.Debug("provisioning> disabled on this hatchery")
		return
	}

	for k := range models {
		// for a shared.infra hatchery, all models are here (group shared.infra or not)
		// but, a shared.infra hatchery can provision only a shared.infra model
		// others hatcheries (not shared.infra): only worker models with same group are here
		// DO NOT provision if hatchery group is not the same as model
		if models[k].GroupID != h.Hatchery().GroupID {
			continue
		}
		if models[k].Type == h.ModelType() {
			existing := h.WorkersStartedByModel(&models[k])
			for i := existing; i < int(models[k].Provision); i++ {
				go func(m sdk.Model) {
					if name, errSpawn := h.SpawnWorker(SpawnArguments{Model: m, IsWorkflowJob: false, JobID: 0, Requirements: nil, LogInfo: "spawn for provision"}); errSpawn != nil {
						log.Warning("provisioning> cannot spawn worker %s with model %s for provisioning: %s", name, m.Name, errSpawn)
						if err := h.CDSClient().WorkerModelSpawnError(m.ID, fmt.Sprintf("hatchery %s cannot spawn worker %s for provisioning: %v", h.Hatchery().Name, m.Name, errSpawn)); err != nil {
							log.Error("provisioning> cannot client.WorkerModelSpawnError for worker %s with model %s for provisioning: %s", name, m.Name, errSpawn)
						}
					}
				}(models[k])
			}
		}
	}
}
