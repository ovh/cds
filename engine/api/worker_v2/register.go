package worker_v2

import (
	"context"
	"time"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/jws"
)

// RegisterWorker  Register new worker
func RegisterWorker(ctx context.Context, db gorpmapper.SqlExecutorWithTx, spawnArgs hatchery.SpawnArgumentsJWTV2, hatch sdk.Hatchery, workerConsumer *sdk.AuthHatcheryConsumer,
	registrationForm sdk.WorkerRegistrationForm) (*sdk.V2Worker, error) {

	if err := spawnArgs.Validate(); err != nil {
		return nil, err
	}

	workerKey, err := jws.NewRandomSymmetricKey(32)
	if err != nil {
		return nil, err
	}

	// Instanciate a new worker
	w := &sdk.V2Worker{
		Name:         spawnArgs.WorkerName,
		Status:       sdk.StatusWaiting,
		HatcheryID:   hatch.ID,
		HatcheryName: hatch.Name,
		LastBeat:     time.Now(),
		ConsumerID:   workerConsumer.ID,
		Version:      registrationForm.Version,
		OS:           registrationForm.OS,
		Arch:         registrationForm.Arch,
		ModelName:    spawnArgs.ModelName,
		JobRunID:     spawnArgs.RunJobID,
		PrivateKey:   workerKey,
	}
	if err := insert(ctx, db, w); err != nil {
		return nil, err
	}
	return w, nil
}
