package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) registerWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		params := &sdk.WorkerRegistrationForm{}
		if err := service.UnmarshalBody(r, params); err != nil {
			return sdk.WrapError(err, "Unable to parse registration form")
		}

		// Check that hatchery exists
		var hatch *sdk.Service
		if params.HatcheryName != "" {
			var errH error
			hatch, errH = services.FindByNameAndType(api.mustDB(), params.HatcheryName, services.TypeHatchery)
			if errH != nil {
				return sdk.WrapError(errH, "registerWorkerHandler> Unable to load hatchery %s", params.HatcheryName)
			}
		}

		// Try to register worker
		worker, err := worker.RegisterWorker(api.mustDB(), params.Name, params.Token, params.ModelID, hatch, params.BinaryCapabilities, params.OS, params.Arch)
		if err != nil {
			err = sdk.NewError(sdk.ErrUnauthorized, err)
			return sdk.WrapError(err, "[%s] Registering failed", params.Name)
		}

		worker.Uptodate = params.Version == sdk.VERSION

		log.Debug("New worker: [%s] - %s", worker.ID, worker.Name)

		// Return worker info to worker itself
		return service.WriteJSON(w, worker, http.StatusOK)
	}
}

func (api *API) getWorkersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(err, "cannot parse form")
		}

		var hatcheryName string
		h := getHatchery(ctx)
		if h != nil {
			hatcheryName = h.Name
		}
		workers, errl := worker.LoadWorkers(api.mustDB(), hatcheryName)
		if errl != nil {
			return sdk.WrapError(errl, "getWorkerModels> cannot load workers")
		}

		return service.WriteJSON(w, workers, http.StatusOK)
	}
}

func (api *API) disableWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		id := vars["id"]

		wor, err := worker.LoadWorker(api.mustDB(), id)
		if err != nil {
			if sdk.Cause(err) != sql.ErrNoRows {
				return sdk.WrapError(err, "Cannot load worker %s", id)
			}
			return sdk.WrapError(sdk.ErrNotFound, "disabledWorkerHandler> Cannot load worker %s", id)
		}

		if wor.Status == sdk.StatusBuilding {
			return sdk.WrapError(sdk.ErrForbidden, "Cannot disable a worker with status %s", wor.Status)
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

func (api *API) refreshWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := worker.RefreshWorker(api.mustDB(), getWorker(ctx)); err != nil && (sdk.Cause(err) != sql.ErrNoRows || sdk.Cause(err) != worker.ErrNoWorker) {
			return sdk.WrapError(err, "cannot refresh last beat of %s", getWorker(ctx).ID)
		}
		return nil
	}
}

func (api *API) unregisterWorkerHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := DisableWorker(api.mustDB(), getWorker(ctx).ID); err != nil {
			return sdk.WrapError(err, "cannot delete worker %s", getWorker(ctx).ID)
		}
		return nil
	}
}

func (api *API) workerCheckingHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerC := getWorker(ctx)
		wk, errW := worker.LoadWorker(api.mustDB(), workerC.ID)
		if errW != nil {
			return sdk.WrapError(errW, "workerCheckingHandler> Unable to load worker %s", workerC.ID)
		}

		if err := worker.SetStatus(api.mustDB(), wk.ID, sdk.StatusChecking); err != nil {
			return sdk.WrapError(err, "cannot update worker %s", workerC.ID)
		}
		key := cache.Key("worker", wk.ID)
		wk.Status = sdk.StatusChecking
		api.Cache.Set(key, wk)

		return nil
	}
}

func (api *API) workerWaitingHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerC := getWorker(ctx)
		wk, errW := worker.LoadWorker(api.mustDB(), workerC.ID)
		if errW != nil {
			return sdk.WrapError(errW, "workerWaitingHandler> Unable to load worker %s", workerC.ID)
		}

		if wk.Status == sdk.StatusWaiting {
			return nil
		}

		if wk.Status != sdk.StatusChecking && wk.Status != sdk.StatusBuilding {
			log.Debug("workerWaitingHandler> Worker %s cannot be Waiting. Current status: %s", wk.Name, wk.Status)
			return nil
		}

		if err := worker.SetStatus(api.mustDB(), wk.ID, sdk.StatusWaiting); err != nil {
			return sdk.WrapError(err, "cannot update worker %s", workerC.ID)
		}
		key := cache.Key("worker", wk.ID)
		wk.Status = sdk.StatusWaiting
		api.Cache.Set(key, wk)

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

	query := `SELECT name, status, action_build_id, job_type FROM worker WHERE id = $1 FOR UPDATE`
	var st, name string
	var jobID sql.NullInt64
	var jobType sql.NullString
	if err := tx.QueryRow(query, id).Scan(&name, &st, &jobID, &jobType); err != nil {
		log.Debug("DisableWorker[%s]> Cannot lock worker: %v", id, err)
		return nil
	}

	if st == sdk.StatusBuilding.String() && jobID.Valid && jobType.Valid {
		// Worker is awol while building !
		// We need to restart this action
		switch jobType.String {
		case sdk.JobTypePipeline:
			if err := pipeline.RestartPipelineBuildJob(tx, jobID.Int64); err != nil {
				log.Error("DisableWorker[%s]> Cannot restart pipeline build job: %v", name, err)
			} else {
				log.Info("DisableWorker[%s]> PipelineBuildJob %d restarted after crash", name, jobID.Int64)
			}
		case sdk.JobTypeWorkflowNode:
			wNodeJob, errL := workflow.LoadNodeJobRun(tx, nil, jobID.Int64)
			if errL == nil && wNodeJob.Retry < 3 {
				if err := workflow.RestartWorkflowNodeJob(nil, db, *wNodeJob); err != nil {
					log.Warning("DisableWorker[%s]> Cannot restart workflow node run: %v", name, err)
				} else {
					log.Info("DisableWorker[%s]> WorkflowNodeRun %d restarted after crash", name, jobID.Int64)
				}
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
