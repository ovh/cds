package worker

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ErrNoWorker means the given worker ID is not found
var ErrNoWorker = fmt.Errorf("cds: no worker found")

// RefreshWorker Update worker last_beat
func RefreshWorker(db gorp.SqlExecutor, w *sdk.Worker) error {
	if w == nil {
		return sdk.WrapError(sdk.ErrUnknownError, "RefreshWorker> Invalid worker")
	}
	query := `UPDATE worker SET last_beat = now() WHERE id = $1`
	res, err := db.Exec(query, w.ID)
	if err != nil {
		return sdk.WrapError(err, "Unable to update worker: %s", w.ID)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "Unable to refresh worker: %s", w.ID)
	}
	if n == 0 {
		return sdk.WithStack(errors.New("unable to refresh worker"))
	}
	return nil
}

// RegistrationForm represents the arguments needed to register a worker
type RegistrationForm struct {
	Name               string
	Token              string
	ModelID            int64
	Hatchery           int64
	HatcheryName       string
	BinaryCapabilities []string
	Version            string
	OS                 string
	Arch               string
}

// TakeForm contains booked JobID if exists
type TakeForm struct {
	BookedJobID int64
	Time        time.Time
}

// RegisterWorker  Register new worker
func RegisterWorker(db *gorp.DbMap, store cache.Store, name string, modelID int64, hatchery *sdk.Service, binaryCapabilities []string, OS, arch string) (*sdk.Worker, error) {
	if name == "" {
		return nil, fmt.Errorf("cannot register worker with empty name")
	}

	//Load Model
	var m *sdk.Model
	if modelID != 0 {
		var errM error
		m, errM = LoadWorkerModelByID(db, modelID)
		if errM != nil {
			log.Warning("RegisterWorker> Cannot load model: %s", errM)
			return nil, errM
		}
	}

	//If worker model is public (sharedInfraGroup) it can be ran by every one
	//If worker is public it can run every model
	//Private worker for a group cannot run a private model for another group
	if m != nil {
		if /*t.GroupID != group.SharedInfraGroup.ID && t.GroupID != m.GroupID &&*/ m.GroupID != group.SharedInfraGroup.ID {
			log.Warning("RegisterWorker> worker %s cannot be spawned as %s (%d)", name, m.Name, m.GroupID)
			return nil, sdk.ErrForbidden
		}
	}

	//Instanciate a new worker
	w := &sdk.Worker{
		ID:      sdk.UUID(),
		Name:    name,
		ModelID: modelID,
		Status:  sdk.StatusWaiting,
	}

	tx, errTx := db.Begin()
	if errTx != nil {
		return nil, errTx
	}
	defer tx.Rollback()

	if err := Insert(tx, w); err != nil {
		log.Warning("registerWorker: Cannot insert worker in database: %s", err)
		return nil, err
	}

	//If the worker is registered for a model and it gave us BinaryCapabilities...
	if len(binaryCapabilities) > 0 && modelID != 0 {
		go func() {
			//Start a new tx for this goroutine
			ntx, err := db.Begin()
			if err != nil {
				log.Warning("RegisterWorker> Unable to start a transaction: %s", err)
				return
			}
			defer ntx.Rollback()

			existingCapas, err := LoadWorkerModelCapabilities(ntx, modelID)
			if err != nil {
				log.Warning("RegisterWorker> Unable to load worker model capabilities: %s", err)
				return
			}

			var newCapas []string
			for _, b := range binaryCapabilities {
				var found bool
				for _, c := range existingCapas {
					if b == c.Value {
						found = true
						break
					}
				}
				if !found {
					newCapas = append(newCapas, b)
				}
			}
			if len(newCapas) > 0 {
				log.Debug("Updating model %d binary capabilities with %d capabilities", modelID, len(newCapas))
				for _, b := range newCapas {
					query := `insert into worker_capability (worker_model_id, name, argument, type) values ($1, $2, $3, $4)`
					if _, err := ntx.Exec(query, modelID, b, b, string(sdk.BinaryRequirement)); err != nil {
						//Ignore errors because we let the database to check constraints...
						log.Debug("registerWorker> Cannot insert into worker_capability: %v", err)
						return
					}
				}
			}

			var capaToDelete []string
			for _, existingCapa := range existingCapas {
				var found bool
				for _, currentCapa := range binaryCapabilities {
					if existingCapa.Value == currentCapa {
						found = true
						break
					}
				}
				if !found {
					capaToDelete = append(capaToDelete, existingCapa.Value)
				}
			}

			if len(capaToDelete) > 0 {
				log.Debug("Updating model %d binary capabilities with %d capabilities to delete", modelID, len(capaToDelete))
				query := `DELETE FROM worker_capability WHERE worker_model_id=$1 AND name=ANY(string_to_array($2, ',')::text[]) AND type=$3`
				if _, err := db.Exec(query, modelID, strings.Join(capaToDelete, ","), string(sdk.BinaryRequirement)); err != nil {
					//Ignore errors because we let the database to check constraints...
					log.Warning("registerWorker> Cannot delete from worker_capability: %v", err)
					return
				}
			}

			if OS != "" && arch != "" {
				if err := updateOSAndArch(db, modelID, OS, arch); err != nil {
					log.Warning("registerWorker> Cannot update os and arch for worker model %d : %s", modelID, err)
					return
				}
			}

			if err := ntx.Commit(); err != nil {
				log.Warning("RegisterWorker> Unable to commit transaction: %s", err)
			}
		}()
		if err := updateRegistration(tx, modelID); err != nil {
			log.Warning("registerWorker> Unable updateRegistration: %s", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return w, err
	}

	keyWorkerModel := keyBookWorkerModel(modelID)
	store.UpdateTTL(keyWorkerModel, modelsCacheTTLInSeconds+10)

	return w, nil
}
