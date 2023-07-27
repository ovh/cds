package api

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	workerauth "github.com/ovh/cds/engine/api/authentication/worker"
	"github.com/ovh/cds/engine/api/worker_v2"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postV2WorkerTakeJobHandler() ([]service.RbacChecker, service.Handler) {
	return []service.RbacChecker{api.jobRunUpdate, api.isWorker}, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		jobRunID := vars["runJobID"]

		wk := getWorker(ctx)
		wrkWithSecret, err := worker_v2.LoadByID(ctx, api.mustDB(), wk.ID, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			return err
		}
		workerKey := wrkWithSecret.PrivateKey

		if wrkWithSecret.Status != sdk.StatusWaiting {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		jobRun, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
		if err != nil {
			return err
		}

		if jobRun.Status != sdk.StatusScheduling {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "unable take the job %s, current status %s", jobRunID, jobRun.Status)
		}

		run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), jobRun.WorkflowRunID)
		if err != nil {
			return err
		}

		contexts, err := computeRunJobContext(ctx, api.mustDB(), *run, *jobRun, *wk)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// Change worker status
		wrkWithSecret.Status = sdk.StatusBuilding
		if err := worker_v2.Update(ctx, tx, wrkWithSecret); err != nil {
			return err
		}

		jobRun.Status = sdk.StatusBuilding
		jobRun.Started = time.Now()
		jobRun.WorkerName = wrkWithSecret.Name
		if err := workflow_v2.UpdateJobRun(ctx, tx, jobRun); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		takeResponse := sdk.V2TakeJobResponse{
			RunJob:        *jobRun,
			AsCodeActions: run.WorkflowData.Actions,
			SigningKey:    base64.StdEncoding.EncodeToString(workerKey),
			Contexts:      *contexts,
		}
		return service.WriteJSON(w, takeResponse, http.StatusOK)
	}
}

func computeRunJobContext(ctx context.Context, db gorp.SqlExecutor, run sdk.V2WorkflowRun, jobRun sdk.V2WorkflowRunJob, wk sdk.V2Worker) (*sdk.WorkflowRunJobsContext, error) {
	contexts := &sdk.WorkflowRunJobsContext{}
	contexts.CDS = run.Contexts.CDS
	contexts.CDS.Job = jobRun.JobID
	contexts.CDS.Stage = jobRun.Job.Stage

	contexts.Vars = run.Contexts.Vars

	contexts.Git = run.Contexts.Git

	runJobs, err := workflow_v2.LoadRunJobsByRunIDAndStatus(ctx, db, run.ID, []string{sdk.StatusFail, sdk.StatusSkipped, sdk.StatusSuccess, sdk.StatusStopped})
	if err != nil {
		return nil, err
	}
	contexts.Jobs = sdk.JobsResultContext{}
	for _, rj := range runJobs {
		jobResult := sdk.JobResultContext{
			Result:  rj.Status,
			Outputs: rj.Outputs,
		}
		contexts.Jobs[rj.JobID] = jobResult
	}
	return contexts, nil
}

func (api *API) postV2RefreshWorkerHandler() ([]service.RbacChecker, service.Handler) {
	return []service.RbacChecker{api.jobRunUpdate, api.isWorker}, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wk := getWorker(ctx)
		wk.LastBeat = time.Now()
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := worker_v2.Update(ctx, tx, wk); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}
}

func (api *API) postV2RegisterWorkerHandler() ([]service.RbacChecker, service.Handler) {
	return nil, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		jobRunID := vars["runJobID"]
		regionName := vars["regionName"]

		// First get the jwt token to checks where this registration is coming from
		jwt := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if jwt == "" {
			return sdk.WithStack(sdk.ErrUnauthorized)
		}

		var registrationForm sdk.WorkerRegistrationForm
		if err := service.UnmarshalBody(r, &registrationForm); err != nil {
			return err
		}

		// Check that the worker can authentify on CDS API
		workerTokenFromHatchery, hatch, err := workerauth.VerifyTokenV2(ctx, api.mustDB(), jwt)
		if err != nil {
			return sdk.NewErrorWithStack(sdk.WrapError(err, "unauthorized worker jwt token %s", jwt), sdk.ErrUnauthorized)
		}

		if err := hatcheryHasRoleOnRegion(ctx, api.mustDB(), hatch.ID, regionName, sdk.HatcheryRoleSpawn); err != nil {
			return err
		}

		hatcheryConsumer, err := authentication.LoadHatcheryConsumerByName(ctx, api.mustDB(), hatch.Name)
		if err != nil {
			return sdk.WrapError(err, "unable to load hatchery %s consumer", hatch.ID)
		}

		// Check runjob status
		runJob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), workerTokenFromHatchery.Worker.RunJobID)
		if err != nil {
			return err
		}
		if runJob.Status != sdk.StatusScheduling || runJob.HatcheryName != hatch.Name || runJob.ID != jobRunID || runJob.Region != regionName {
			return sdk.WrapError(sdk.ErrForbidden, "unable to take job %s, current status: %s, hatchery: %s, region: %s", runJob.ID, runJob.Status, runJob.HatcheryName, runJob.Region)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// We have to issue a new consumer for the worker
		workerConsumer, err := authentication.NewConsumerWorkerV2(ctx, tx, workerTokenFromHatchery.Subject, hatcheryConsumer)
		if err != nil {
			return err
		}

		// Try to register worker
		wk, err := worker_v2.RegisterWorker(ctx, tx, workerTokenFromHatchery.Worker, *hatch, workerConsumer, registrationForm)
		if err != nil {
			return sdk.NewErrorWithStack(
				sdk.WrapError(err, "[%s] Registering failed", workerTokenFromHatchery.Worker.WorkerName),
				sdk.ErrUnauthorized,
			)
		}

		log.Debug(ctx, "New worker: [%s] - %s", wk.ID, wk.Name)

		workerSession, err := authentication.NewSession(ctx, tx, &workerConsumer.AuthConsumer, workerauth.SessionDuration)
		if err != nil {
			return sdk.NewErrorWithStack(
				sdk.WrapError(err, "[%s] Registering failed", workerTokenFromHatchery.Worker.WorkerName),
				sdk.ErrUnauthorized,
			)
		}

		// Store the last authentication date on the consumer
		now := time.Now()
		workerConsumer.LastAuthentication = &now
		if err := authentication.UpdateConsumerLastAuthentication(ctx, tx, &workerConsumer.AuthConsumer); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		jwt, err = authentication.NewSessionJWT(workerSession, "")
		if err != nil {
			return sdk.NewErrorWithStack(
				sdk.WrapError(err, "[%s] Registering failed", workerTokenFromHatchery.Worker.WorkerName),
				sdk.ErrUnauthorized,
			)
		}

		// Set the JWT token as a header
		log.Debug(ctx, "worker.registerWorkerHandler> X-CDS-JWT:%s", sdk.StringFirstN(jwt, 12))
		w.Header().Add("X-CDS-JWT", jwt)

		// Return worker info to worker itself
		return service.WriteJSON(w, wk, http.StatusOK)
	}
}

func (api *API) postV2UnregisterWorkerHandler() ([]service.RbacChecker, service.Handler) {
	return []service.RbacChecker{api.jobRunUpdate, api.isWorker}, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		wk := getWorker(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		wk.Status = sdk.StatusDisabled
		if err := worker_v2.Update(ctx, tx, wk); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}
}
