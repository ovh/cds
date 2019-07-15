package hatchery

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/namesgenerator"
)

var (
	nbWorkerToStart int64
)

func checkCapacities(ctx context.Context, h Interface) bool {
	t := time.Now()
	defer log.Debug("hatchery> checkCapacities> %.3f seconds elapsed", time.Since(t).Seconds())

	workerPool, err := WorkerPool(ctx, h, sdk.StatusChecking, sdk.StatusWaiting, sdk.StatusBuilding, sdk.StatusWorkerPending, sdk.StatusWorkerRegistering)
	if err != nil {
		log.Error("hatchery> checkCapacities> Pool> Error: %v", err)
		return false
	}

	if len(workerPool) >= h.Configuration().Provision.MaxWorker {
		log.Debug("hatchery> checkCapacities> %s has reached the max worker: %d (max: %d)", h.Service().Name, len(workerPool), h.Configuration().Provision.MaxWorker)
		if len(workerPool) > h.Configuration().Provision.MaxWorker {
			for _, w := range workerPool {
				log.Debug("hatchery> checkCapacities> %s > pool > %s (status=%v)", h.Service().Name, w.Name, w.Status)
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
		log.Info("hatchery> checkCapacities> too many pending worker in pool: %d", nbPending)
		return false
	}

	if int(atomic.LoadInt64(&nbWorkerToStart)) >= maxProv {
		log.Info("hatchery> checkCapacities> too many starting worker in pool: %d", atomic.LoadInt64(&nbWorkerToStart))
		return false
	}

	return true
}

func provisioning(h InterfaceWithModels, models []sdk.Model) {
	if h.Configuration().Provision.Disabled {
		log.Debug("provisioning> disabled on this hatchery")
		return
	}

	for k := range models {
		if models[k].Type == h.ModelType() {
			existing := h.WorkersStartedByModel(&models[k])
			for i := existing; i < int(models[k].Provision); i++ {
				go func(m sdk.Model) {
					arg := SpawnArguments{
						WorkerName:   fmt.Sprintf("%s-%s", strings.ToLower(m.Name), strings.Replace(namesgenerator.GetRandomNameCDS(0), "_", "-", -1)),
						Model:        &m,
						HatcheryName: h.ServiceName(),
					}
					// Get a JWT to authentified the worker
					_, jwt, err := NewWorkerToken(h.ServiceName(), h.PrivateKey(), time.Now().Add(1*time.Hour), arg)
					if err != nil {
						var spawnError = sdk.SpawnErrorForm{
							Error: fmt.Sprintf("hatchery %s cannot spawn worker %s for provisioning", h.Service().Name, m.Name),
							Logs:  []byte(err.Error()),
						}
						if err := h.CDSClient().WorkerModelSpawnError(m.Group.Name, m.Name, spawnError); err != nil {
							log.Error("provisioning> cannot client.WorkerModelSpawnError for worker %s with model %s for provisioning: %v", arg.WorkerName, m.Name, err)
						}
						return
					}
					arg.WorkerToken = jwt

					if errSpawn := h.SpawnWorker(context.Background(), arg); errSpawn != nil {
						log.Warning("provisioning> cannot spawn worker %s with model %s for provisioning: %v", arg.WorkerName, m.Name, errSpawn)
						var spawnError = sdk.SpawnErrorForm{
							Error: fmt.Sprintf("hatchery %s cannot spawn worker %s for provisioning", h.Service().Name, m.Name),
							Logs:  []byte(errSpawn.Error()),
						}
						if err := h.CDSClient().WorkerModelSpawnError(m.Group.Name, m.Name, spawnError); err != nil {
							log.Error("provisioning> cannot client.WorkerModelSpawnError for worker %s with model %s for provisioning: %v", arg.WorkerName, m.Name, errSpawn)
						}
					}
				}(models[k])
			}
		}
	}
}
