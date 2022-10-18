package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
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
func RegisterWorker(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store,
	spawnArgs hatchery.SpawnArgumentsJWT, hatcheryService sdk.Service, hatcheryConsumer *sdk.AuthUserConsumer, workerConsumer *sdk.AuthUserConsumer,
	registrationForm sdk.WorkerRegistrationForm, runNodeJob *sdk.WorkflowNodeJobRun) (*sdk.Worker, error) {

	if err := spawnArgs.Validate(); err != nil {
		return nil, err
	}

	var model *sdk.Model
	if spawnArgs.Model.ID > 0 {
		var err error
		model, err = workermodel.LoadByID(ctx, db, spawnArgs.Model.ID, workermodel.LoadOptions.Default)
		if err != nil {
			return nil, err
		}
	}
	if spawnArgs.RegisterOnly && model == nil {
		return nil, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unauthorized to register only worker without model information")
	}

	// To use a model, the hatchery's consumer should have read access to the model's group or be CDS maintainer. Shared models can be used by every Hatchery.
	// For restricted models, the hatchery consumer should have the model's group, it will not inherit permission from its parent.
	// To register a model, the hatchery's consumer should be admin of the model's group or CDS admin.
	if model != nil {
		g, err := group.LoadByID(ctx, db, model.GroupID, group.LoadOptions.WithMembers)
		if err != nil {
			return nil, err
		}
		canUseModel := sdk.IsInInt64Array(model.GroupID, hatcheryConsumer.GetGroupIDs()) || hatcheryConsumer.Maintainer() || model.GroupID == group.SharedInfraGroup.ID
		cantUseRestrictedModel := model.Restricted && !sdk.IsInInt64Array(model.GroupID, hatcheryConsumer.AuthConsumerUser.GroupIDs)
		if !canUseModel || cantUseRestrictedModel {
			return nil, sdk.WrapError(sdk.ErrForbidden, "hatchery can't use model %q", model.Path())
		}
		canRegisterModel := g.IsAdmin(*hatcheryConsumer.AuthConsumerUser.AuthentifiedUser) || hatcheryConsumer.Admin()
		if spawnArgs.RegisterOnly && !canRegisterModel {
			return nil, sdk.WrapError(sdk.ErrForbidden, "hatchery can't register model %q", model.Path())
		}
	}

	// To register a job the hatchery's consumer should have a group with read/exec permission on the job.
	// If the model is used to run a job, its group should be in the job's exec groups
	if spawnArgs.JobID > 0 {
		canTakeJob := runNodeJob.ExecGroups.HasOneOf(hatcheryConsumer.GetGroupIDs()...) || hatcheryConsumer.Admin()
		if !canTakeJob {
			return nil, sdk.WrapError(sdk.ErrForbidden, "hatchery can't register job with id %q", spawnArgs.JobID)
		}

		if model != nil {
			canUseModelForJob := runNodeJob.ExecGroups.HasOneOf(model.GroupID)
			if !canUseModelForJob {
				return nil, sdk.WrapError(sdk.ErrForbidden, "hatchery can't register job (id: %q) with model (path: %q) for a group that is not in job's exec groups", spawnArgs.JobID, model.Path())
			}
		}

		// Check additional information based on the consumer if a region is set.
		// Allows to register only job with same region or job without region if ServiceIgnoreJobWithNoRegion is not true.
		if hatcheryConsumer.AuthConsumerUser.ServiceRegion != nil && *hatcheryConsumer.AuthConsumerUser.ServiceRegion != "" {
			if runNodeJob.Region == nil {
				if hatcheryConsumer.AuthConsumerUser.ServiceIgnoreJobWithNoRegion != nil && *hatcheryConsumer.AuthConsumerUser.ServiceIgnoreJobWithNoRegion {
					return nil, sdk.WrapError(sdk.ErrForbidden, "hatchery can't register job with id %d without region requirement", spawnArgs.JobID)
				}
			} else if *runNodeJob.Region != *hatcheryConsumer.AuthConsumerUser.ServiceRegion {
				return nil, sdk.WrapError(sdk.ErrForbidden, "hatchery can't register job with id %d for region %s", spawnArgs.JobID, *runNodeJob.Region)
			}
		}
	}

	// Instanciate a new worker
	w := &sdk.Worker{
		ID:           sdk.UUID(),
		Name:         spawnArgs.WorkerName,
		Status:       sdk.StatusWaiting,
		HatcheryID:   &hatcheryService.ID,
		HatcheryName: hatcheryService.Name,
		LastBeat:     time.Now(),
		ConsumerID:   workerConsumer.ID,
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

	// If the worker is registered for a model and it gave us BinaryCapabilities...
	if model != nil && spawnArgs.RegisterOnly {
		if len(registrationForm.BinaryCapabilities) > 0 {
			if err := workermodel.UpdateCapabilities(ctx, db, model.ID, registrationForm); err != nil {
				log.Error(ctx, "updateWorkerModelCapabilities> %v", err)
			}
		}
		if err := workermodel.UpdateRegistration(ctx, db, store, model.ID); err != nil {
			log.Warn(ctx, "registerWorker> Unable to update registration: %s", err)
		}
	}

	return w, nil
}
