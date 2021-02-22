package hatchery

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// workerRegister is called by a ticker.
// the hatchery checks each worker model, and if a worker model needs to
// be registered, the hatchery calls SpawnWorker().
// each ticker can trigger 5 worker models (maximum)
// and 5 worker models can be spawned in same time, in the case of a spawn takes longer
// than a tick.
var nbRegisteringWorkerModels int64

func workerRegister(ctx context.Context, h InterfaceWithModels, startWorkerChan chan<- workerStarterRequest) error {
	if len(models) == 0 {
		return fmt.Errorf("hatchery> workerRegister> No model returned by GetWorkerModels")
	}
	// currentRegister contains the register spawned in this ticker
	currentRegistering, err := WorkerPool(ctx, h, sdk.StatusWorkerRegistering)
	if err != nil {
		log.Error(ctx, "hatchery> workerRegister> worker pool error: %v", err)
	}

	atomic.StoreInt64(&nbRegisteringWorkerModels, int64(len(currentRegistering)))
loopModels:
	for k := range models {
		log.Info(ctx, "checking for worker model %q registration")
		if models[k].Type != h.ModelType() {
			continue
		}
		log.Info(ctx, "checking for worker model %q registration (%+v)", models[k].Name, models[k])
		if h.CanSpawn(ctx, &models[k], 0, nil) && (h.NeedRegistration(ctx, &models[k]) || models[k].CheckRegistration) {
			log.Debug(ctx, "hatchery> workerRegister> need register")
		} else {
			continue
		}

		maxRegistration := int64(h.Configuration().Provision.MaxConcurrentRegistering)
		if maxRegistration == 0 {
			maxRegistration = 2
		}
		if atomic.LoadInt64(&nbRegisteringWorkerModels) > maxRegistration {
			log.Debug(ctx, "hatchery> workerRegister> max registering worker reached")
			return nil
		}

		if !checkCapacities(ctx, h) {
			log.Debug(ctx, "hatchery> workerRegister> unable to register now")
			return nil
		}

		// Check if there is a pending registering worker
		for _, w := range currentRegistering {
			if strings.Contains(w.Name, models[k].Name) {
				log.Info(ctx, "hatchery> workerRegister> %s is already registering (%s)", models[k].Name, w.Name)
				continue loopModels
			}
		}

		// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
		if models[k].NbSpawnErr > 5 {
			log.Warn(ctx, "hatchery> workerRegister> Too many errors on spawn with model %s, please check this worker model", models[k].Name)
			continue
		}

		if err := h.CDSClient().WorkerModelBook(models[k].Group.Name, models[k].Name); err != nil {
			log.Debug(ctx, "%v", sdk.WrapError(err, "cannot book model %s with id %d", models[k].Path(), models[k].ID))
			continue
		}

		log.Info(ctx, "hatchery> workerRegister> spawning model %s (%d)", models[k].Name, models[k].ID)

		// Interpolate model secrets
		if err := ModelInterpolateSecrets(h, &models[k]); err != nil {
			log.Error(ctx, "hatchery> workerRegister> cannot interpolate secrets for model %s: %v", models[k].Path(), err)
			continue
		}

		//Ask for the creation
		startWorkerChan <- workerStarterRequest{
			registerWorkerModel: &models[k],
		}
	}
	return nil
}

// CheckWorkerModelRegister checks if a model has been registered, if not it raises an error on the API
func CheckWorkerModelRegister(h Interface, modelPath string) error {
	var sendError bool
	for i := range models {
		if models[i].Group.Name+"/"+models[i].Name == modelPath {
			sendError = models[i].NeedRegistration
			break
		}
	}
	if sendError {
		return sdk.WithStack(sdk.ErrWorkerModelDeploymentFailed)
	}
	return nil
}
