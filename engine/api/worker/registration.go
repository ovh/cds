package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// ErrNoWorker means the given worker ID is not found
var ErrNoWorker = fmt.Errorf("cds: no worker found")

// RefreshWorker Update worker last_beat
func RefreshWorker(db gorp.SqlExecutor, id string) error {
	query := `UPDATE worker SET last_beat = now() WHERE id = $1`
	res, err := db.Exec(query, id)
	if err != nil {
		return sdk.WrapError(err, "Unable to update worker: %s", id)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return sdk.WrapError(err, "Unable to refresh worker: %s", id)
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
func RegisterWorker(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, spawnArgs hatchery.SpawnArguments, hatcheryService sdk.Service, consumer *sdk.AuthConsumer, registrationForm sdk.WorkerRegistrationForm) (*sdk.Worker, error) {
	if spawnArgs.WorkerName == "" {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register a worker without a name")
	}

	if !spawnArgs.RegisterOnly && spawnArgs.JobID == 0 {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register a worker for a job without a JobID")
	}

	var model *sdk.Model
	if spawnArgs.Model != nil {
		var err error
		model, err = workermodel.LoadByID(ctx, db, spawnArgs.Model.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return nil, err
		}
	}

	// If worker model is public (sharedInfraGroup) it can be ran by every one
	// If worker is public it can run every model
	// Private worker for a group cannot run a private model for another group
	if model != nil && !sdk.IsInInt64Array(group.SharedInfraGroup.ID, consumer.GetGroupIDs()) &&
		!sdk.IsInInt64Array(model.GroupID, consumer.GetGroupIDs()) &&
		model.GroupID != group.SharedInfraGroup.ID {
		return nil, sdk.WithStack(sdk.ErrForbidden)
	}

	//Instanciate a new worker
	w := &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         spawnArgs.WorkerName,
		Status:       sdk.StatusWaiting,
		HatcheryID:   &hatcheryService.ID,
		HatcheryName: hatcheryService.Name,
		LastBeat:     time.Now(),
		ConsumerID:   consumer.ID,
		Version:      registrationForm.Version,
		OS:           registrationForm.OS,
		Arch:         registrationForm.Arch,
	}
	if model != nil {
		w.ModelID = &spawnArgs.Model.ID
	}
	if spawnArgs.JobID > 0 {
		w.JobRunID = &spawnArgs.JobID
	}

	w.Uptodate = registrationForm.Version == sdk.VERSION

	if err := Insert(ctx, db, w); err != nil {
		return nil, err
	}

	//If the worker is registered for a model and it gave us BinaryCapabilities...
	if model != nil && spawnArgs.RegisterOnly {
		if len(registrationForm.BinaryCapabilities) > 0 {
			if err := workermodel.UpdateCapabilities(ctx, db, model.ID, registrationForm); err != nil {
				log.Error(ctx, "updateWorkerModelCapabilities> %v", err)
			}
		}
		if err := workermodel.UpdateRegistration(ctx, db, store, model.ID); err != nil {
			log.Warning(ctx, "registerWorker> Unable to update registration: %s", err)
		}
	}

	return w, nil
}
