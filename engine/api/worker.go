package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	workerauth "github.com/ovh/cds/engine/api/authentication/worker"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postRegisterWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// First get the jwt token to checks where this registration is coming from
		jwt := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if jwt == "" {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		var registrationForm sdk.WorkerRegistrationForm
		if err := service.UnmarshalBody(r, &registrationForm); err != nil {
			return sdk.WrapError(err, "Unable to parse registration form")
		}

		// Check that the worker can authentify on CDS API
		workerTokenFromHatchery, err := workerauth.VerifyToken(api.mustDB(), jwt)
		if err != nil {
			log.Error("registerWorkerHandler> unauthorized worker jwt token %s: %v", jwt[:12], err)
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		// Check that hatchery exists
		hatchSrv, err := services.LoadByNameAndType(ctx, api.mustDB(), workerTokenFromHatchery.Worker.HatcheryName, services.TypeHatchery)
		if err != nil {
			return sdk.WrapError(err, "registerWorkerHandler> Unable to load hatchery %s", workerTokenFromHatchery.Worker.HatcheryName)
		}

		// Retrieve the authentifed Consumer from the hatchery
		hatcheryConsumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), *hatchSrv.ConsumerID)
		if err != nil {
			return sdk.WrapError(err, "registerWorkerHandler> Unable to load consumer %v", hatchSrv.ConsumerID)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		var groupIDs []int64
		if workerTokenFromHatchery.Worker.JobID != 0 {
			job, err := workflow.LoadNodeJobRun(tx, api.Cache, workerTokenFromHatchery.Worker.JobID)
			if err != nil {
				return sdk.NewError(sdk.ErrForbidden, err)
			}
			groupIDs = sdk.Groups(job.ExecGroups).ToIDs()
		} else {
			groupIDs = hatcheryConsumer.GetGroupIDs()
		}

		// We have to issue a new consumer for the worker
		workerConsumer, err := authentication.NewConsumerWorker(api.mustDB(), workerTokenFromHatchery.Subject, hatchSrv, hatcheryConsumer, groupIDs)
		if err != nil {
			return err
		}

		// Try to register worker
		wk, err := worker.RegisterWorker(tx, api.Cache, workerTokenFromHatchery.Worker, hatchSrv.ID, workerConsumer, registrationForm)
		if err != nil {
			err = sdk.NewError(sdk.ErrUnauthorized, err)
			return sdk.WrapError(err, "[%s] Registering failed", workerTokenFromHatchery.Worker.WorkerName)
		}

		log.Debug("New worker: [%s] - %s", wk.ID, wk.Name)

		workerSession, err := authentication.NewSession(tx, workerConsumer, workerauth.SessionDuration)
		if err != nil {
			err = sdk.NewError(sdk.ErrUnauthorized, err)
			return sdk.WrapError(err, "[%s] Registering failed", workerTokenFromHatchery.Worker.WorkerName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		jwt, err = authentication.NewSessionJWT(workerSession)
		if err != nil {
			err = sdk.NewError(sdk.ErrUnauthorized, err)
			return sdk.WrapError(err, "[%s] Registering failed", workerTokenFromHatchery.Worker.WorkerName)
		}

		// Set the JWT token as a header
		log.Debug("worker.registerWorkerHandler> X-CDS-JWT:%s", jwt[:12])
		w.Header().Add("X-CDS-JWT", jwt)

		// Return worker info to worker itself
		return service.WriteJSON(w, wk, http.StatusOK)
	}
}

func (api *API) getWorkersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var workers []sdk.Worker
		var err error
		if !isAdmin(ctx) {
			h, err := services.LoadByConsumerID(ctx, api.mustDB(), getAPIConsumer(ctx).ID)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			workers, err = worker.LoadByHatcheryID(ctx, api.mustDB(), h.ID)
			if err != nil {
				return err
			}
		} else {
			workers, err = worker.LoadAll(ctx, api.mustDB())
			if err != nil {
				return err
			}
		}
		return service.WriteJSON(w, workers, http.StatusOK)
	}
}

func (api *API) disableWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		id := vars["id"]

		wk, err := worker.LoadByID(ctx, api.mustDB(), id)
		if err != nil {
			return err
		}

		if !isAdmin(ctx) {
			if wk.Status == sdk.StatusBuilding {
				return sdk.WrapError(sdk.ErrForbidden, "Cannot disable a worker with status %s", wk.Status)
			}
			hatcherySrv, err := services.LoadByConsumerID(ctx, api.mustDB(), getAPIConsumer(ctx).ID)
			if err != nil {
				return sdk.WrapError(sdk.ErrForbidden, "Cannot disable a worker from this hatchery: %v", err)
			}
			if wk.HatcheryID != hatcherySrv.ID {
				return sdk.WrapError(sdk.ErrForbidden, "Cannot disable a worker from hatchery (expected: %d/actual: %d)", wk.HatcheryID, hatcherySrv.ID)
			}
		}

		if err := DisableWorker(api.mustDB(), id); err != nil {
			cause := sdk.Cause(err)
			if cause == worker.ErrNoWorker || cause == sql.ErrNoRows {
				return sdk.WrapError(sdk.ErrWrongRequest, "disableWorkerHandler> worker %s does not exists", id)
			}
			return sdk.WrapError(err, "cannot update worker status")
		}

		//Remove the worker from the cache
		key := cache.Key("worker", id)
		api.Cache.Delete(key)

		return nil
	}
}

func (api *API) postRefreshWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wk, err := worker.LoadByConsumerID(ctx, api.mustDB(), getAPIConsumer(ctx).ID)
		if err != nil {
			return err
		}

		if err := worker.RefreshWorker(api.mustDB(), wk.ID); err != nil && (sdk.Cause(err) != sql.ErrNoRows || sdk.Cause(err) != worker.ErrNoWorker) {
			return sdk.WrapError(err, "cannot refresh last beat of %s", wk.Name)
		}
		return nil
	}
}

func (api *API) postUnregisterWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wk, err := worker.LoadByConsumerID(ctx, api.mustDB(), getAPIConsumer(ctx).ID)
		if err != nil {
			return err
		}
		if err := DisableWorker(api.mustDB(), wk.ID); err != nil {
			return sdk.WrapError(err, "cannot delete worker %s", wk.Name)
		}
		return nil
	}
}

func (api *API) workerWaitingHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wk, err := worker.LoadByConsumerID(ctx, api.mustDB(), getAPIConsumer(ctx).ID)
		if err != nil {
			return err
		}

		if wk.Status == sdk.StatusWaiting {
			return nil
		}

		if wk.Status != sdk.StatusChecking && wk.Status != sdk.StatusBuilding {
			log.Debug("workerWaitingHandler> Worker %s cannot be Waiting. Current status: %s", wk.Name, wk.Status)
			return nil
		}

		if err := worker.SetStatus(api.mustDB(), wk.ID, sdk.StatusWaiting); err != nil {
			return sdk.WrapError(err, "cannot update worker %s", wk.ID)
		}
		return nil
	}
}

// After migration to new CDS Workflow, put DisableWorker into
// the package workflow

// DisableWorker disable a worker
func DisableWorker(db *gorp.DbMap, id string) error {
	tx, errb := db.Begin()
	if errb != nil {
		return fmt.Errorf("DisableWorker> Cannot start tx: %v", errb)
	}
	defer tx.Rollback() // nolint

	query := `SELECT name, status, job_run_id FROM worker WHERE id = $1 FOR UPDATE`
	var st, name string
	var jobID sql.NullInt64
	if err := tx.QueryRow(query, id).Scan(&name, &st, &jobID); err != nil {
		log.Debug("DisableWorker[%s]> Cannot lock worker: %v", id, err)
		return nil
	}

	if st == sdk.StatusBuilding && jobID.Valid {
		// Worker is awol while building !
		// We need to restart this action
		wNodeJob, errL := workflow.LoadNodeJobRun(tx, nil, jobID.Int64)
		if errL == nil && wNodeJob.Retry < 3 {
			if err := workflow.RestartWorkflowNodeJob(nil, db, *wNodeJob); err != nil {
				log.Warning("DisableWorker[%s]> Cannot restart workflow node run: %v", name, err)
			} else {
				log.Info("DisableWorker[%s]> WorkflowNodeRun %d restarted after crash", name, jobID.Int64)
			}
		}

		log.Info("DisableWorker> Worker %s crashed while building %d !", name, jobID.Int64)
	}

	if err := worker.SetStatus(tx, id, sdk.StatusDisabled); err != nil {
		cause := sdk.Cause(err)
		if cause == worker.ErrNoWorker || cause == sql.ErrNoRows {
			return sdk.WrapError(sdk.ErrWrongRequest, "DisableWorker> worker %s does not exists", id)
		}
		return sdk.WrapError(err, "cannot update worker status")
	}

	return tx.Commit()
}
