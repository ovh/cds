package hatchery

import (
	"fmt"
	"math"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Register calls CDS API to register current hatchery
func Register(h Interface) error {
	newHatchery, uptodate, err := h.CDSClient().HatcheryRegister(*h.Hatchery())
	if err != nil {
		return sdk.WrapError(err, "register> Got HTTP exiting")
	}
	h.Hatchery().ID = newHatchery.ID
	h.Hatchery().GroupID = newHatchery.GroupID
	h.Hatchery().Model = newHatchery.Model
	h.Hatchery().Name = newHatchery.Name
	h.Hatchery().IsSharedInfra = newHatchery.IsSharedInfra

	log.Info("Register> Hatchery %s registered with id:%d", h.Hatchery().Name, h.Hatchery().ID)

	if !uptodate {
		log.Warning("-=-=-=-=- Please update your hatchery binary - Hatchery Version:%s %s %s -=-=-=-=-", sdk.VERSION, runtime.GOOS, runtime.GOARCH)
	}
	return nil
}

// workerRegister is called by a ticker.
// the hatchery checks each worker model, and if a worker model needs to
// be registered, the hatchery calls SpawnWorker().
// each ticker can trigger 5 worker models (maximum)
// and 5 worker models can be spawned in same time, in the case of a spawn takes longer
// than a tick.
var nbRegisteringWorkerModels int64

func workerRegister(h Interface, startWorkerChan chan<- workerStarterRequest) error {
	if len(models) == 0 {
		return fmt.Errorf("hatchery> workerRegister> No model returned by GetWorkerModels")
	}
	// currentRegister contains the register spawned in this ticker
	currentRegistering, err := WorkerPool(h, sdk.StatusWorkerRegistering, sdk.StatusWorkerPending)
	if err != nil {
		log.Error("hatchery> workerRegister> %v", err)
	}

	atomic.StoreInt64(&nbRegisteringWorkerModels, int64(len(currentRegistering)))
	for k := range models {
		if models[k].Type != h.ModelType() {
			continue
		}
		maxRegistration := int64(math.Floor(float64(h.Configuration().Provision.MaxWorker) / 4))
		if atomic.LoadInt64(&nbRegisteringWorkerModels) > maxRegistration {
			log.Debug("hatchery> workerRegister> max registering worker reached")
			return nil
		}

		if !checkCapacities(h) {
			log.Debug("hatchery> workerRegister> unable to register now")
			return nil
		}

		// Check if there is a pending registering worker
		for _, w := range currentRegistering {
			if strings.HasPrefix(w.Name, "register-") && strings.Contains(w.Name, models[k].Name) {
				log.Info("hatchery> workerRegister> %s is already registering (%s)", models[k].Name, w.Name)
			}
		}

		// if current hatchery is in same group than worker model -> do not avoid spawn, even if worker model is in error
		if models[k].NbSpawnErr > 5 && h.Hatchery().GroupID != models[k].ID {
			log.Warning("hatchery> workerRegister> Too many errors on spawn with model %s, please check this worker model", models[k].Name)
			continue
		}

		if h.NeedRegistration(&models[k]) || models[k].CheckRegistration {
			if err := h.CDSClient().WorkerModelBook(models[k].ID); err != nil {
				log.Debug("hatchery> workerRegister> WorkerModelBook on model %s err: %v", models[k].Name, err)
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
func CheckWorkerModelRegister(h Interface, modelID int64) {
	var sendError bool
	var m *sdk.Model
	for i := range models {
		m = &models[i]
		year, month, day := m.LastRegistration.Date()
		if m.ID == modelID {
			sendError = year == 1 && month == 1 && day == 1
			log.Debug("checking last registration date of %s: %v (%v)", m.Name, m.LastRegistration, sendError)
			break
		}
	}
	if m != nil && sendError {
		if err := h.CDSClient().WorkerModelSpawnError(m.ID, fmt.Sprintf("worker model deployment failed")); err != nil {
			log.Error("CheckWorkerModelRegister> error on call client.WorkerModelSpawnError on worker model %s for register: %s", m.Name, err)
		}
	}
}
