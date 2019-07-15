package hatchery

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
		log.Error("hatchery> workerRegister> worker pool error: %v", err)
	}

	atomic.StoreInt64(&nbRegisteringWorkerModels, int64(len(currentRegistering)))
loopModels:
	for k := range models {
		if models[k].Type != h.ModelType() {
			continue
		}
		if h.NeedRegistration(&models[k]) || models[k].CheckRegistration {
			log.Debug("hatchery> workerRegister> need register")
		} else {
			continue
		}

		maxRegistration := int64(h.Configuration().Provision.MaxConcurrentRegistering)
		if maxRegistration == 0 {
			maxRegistration = 2
		}
		if atomic.LoadInt64(&nbRegisteringWorkerModels) > maxRegistration {
			log.Debug("hatchery> workerRegister> max registering worker reached")
			return nil
		}

		if !checkCapacities(ctx, h) {
			log.Debug("hatchery> workerRegister> unable to register now")
			return nil
		}

		// Check if there is a pending registering worker
		for _, w := range currentRegistering {
			if strings.Contains(w.Name, models[k].Name) {
				log.Info("hatchery> workerRegister> %s is already registering (%s)", models[k].Name, w.Name)
				continue loopModels
			}
		}

		// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
		if models[k].NbSpawnErr > 5 {
			log.Warning("hatchery> workerRegister> Too many errors on spawn with model %s, please check this worker model", models[k].Name)
			continue
		}

		if h.NeedRegistration(&models[k]) || models[k].CheckRegistration {
			if err := h.CDSClient().WorkerModelBook(models[k].Group.Name, models[k].Name); err != nil {
				log.Debug("%v", sdk.WrapError(err, "cannot book model %s with id %d", models[k].Name, models[k].ID))
			} else {
				log.Info("hatchery> workerRegister> spawning model %s (%d)", models[k].Name, models[k].ID)
				//Ask for the creation
				startWorkerChan <- workerStarterRequest{
					registerWorkerModel: &models[k],
				}
			}
		}
	}
	return nil
}

// CheckWorkerModelRegister checks if a model has been registered, if not it raises an error on the API
func CheckWorkerModelRegister(h Interface, modelPath string) error {
	var sendError bool
	var m *sdk.Model
	for i := range models {
		m = &models[i]
		year, month, day := m.LastRegistration.Date()
		if m.Group.Name+"/"+m.Name == modelPath {
			sendError = year == 1 && month == 1 && day == 1
			log.Debug("checking last registration date of %s: %v (%v)", m.Name, m.LastRegistration, sendError)
			break
		}
	}
	if m != nil && sendError {
		return sdk.ErrWorkerModelDeploymentFailed
	}
	return nil
}
