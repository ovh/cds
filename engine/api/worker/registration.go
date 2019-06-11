package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/accesstoken"
	"github.com/ovh/cds/engine/api/cache"

	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/sdk"
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
func RegisterWorker(db gorp.SqlExecutor, store cache.Store, spawnArgs hatchery.SpawnArguments, hatchery *sdk.Service, registrationForm sdk.WorkerRegistrationForm, execsgroups []sdk.Group) (*sdk.Worker, string, error) {
	if spawnArgs.WorkerName == "" {
		return nil, "", sdk.WithStack(sdk.ErrWrongRequest)
	}
	// Load Model
	model, err := workermodel.LoadByID(db, spawnArgs.Model.ID)
	if err != nil {
		return nil, "", err
	}

	// Load the access token of the hatchery
	atoken, err := accesstoken.LoadSessionByID(context.Background(), db, hatchery.TokenID,
		accesstoken.LoadSessionOptions.WithGroups,
	)
	if err != nil {
		return nil, "", err
	}

	// If worker model is public (sharedInfraGroup) it can be ran by every one
	// If worker is public it can run every model
	// Private worker for a group cannot run a private model for another group
	if !sdk.IsInInt64Array(group.SharedInfraGroup.ID, sdk.GroupsToIDs(atoken.Groups)) &&
		!sdk.IsInInt64Array(model.GroupID, sdk.GroupsToIDs(atoken.Groups)) &&
		model.GroupID != group.SharedInfraGroup.ID {
		return nil, "", sdk.WithStack(sdk.ErrForbidden)
	}

	//Instanciate a new worker
	w := &sdk.Worker{
		ID:         sdk.UUID(),
		Name:       spawnArgs.WorkerName,
		ModelID:    spawnArgs.Model.ID,
		Status:     sdk.StatusWaiting,
		HatcheryID: hatchery.ID,
		LastBeat:   time.Now(),
	}
	w.Uptodate = registrationForm.Version == sdk.VERSION

	// TODO:
	// we probably should remove shareinfra from the groups
	// and make the intersection with hatchery accesstoken group

	//	jwtoken, jwt, err := accesstoken.New( /*hatchery.Maintainer*/ sdk.AuthConsumer{} /*execsgroups*/, nil, []string{sdk.AccessTokenScopeRunExecution}, spawnArgs.HatcheryName, spawnArgs.WorkerName, time.Now().Add(24*time.Hour))
	/*if err != nil {
		return nil, "", err
	}

	if err := accesstoken.InsertSession(db, &jwtoken); err != nil {
		return nil, "", err
	}

	w.AccessTokenID = &jwtoken.ID*/

	if err := Insert(db, w); err != nil {
		return nil, "", err
	}

	//If the worker is registered for a model and it gave us BinaryCapabilities...
	if spawnArgs.RegisterOnly && len(registrationForm.BinaryCapabilities) > 0 && spawnArgs.Model.ID != 0 {
		if err := workermodel.UpdateCapabilities(db, spawnArgs, registrationForm); err != nil {
			log.Error("updateWorkerModelCapabilities> %v", err)
		}
		if err := workermodel.UpdateRegistration(db, store, spawnArgs.Model.ID); err != nil {
			log.Warning("registerWorker> Unable to update registration: %s", err)
		}
	}

	return w, "", nil
}
