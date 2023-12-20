package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	JobRunHatcheryTakeKey = "workflow:jobrun:hatchery:take"
)

func (api *API) postJobRunStepHandler() ([]service.RbacChecker, service.Handler) {
	return []service.RbacChecker{api.jobRunUpdate}, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		jobRunID := vars["runJobID"]

		var stepsStatus sdk.JobStepsStatus
		if err := service.UnmarshalBody(r, &stepsStatus); err != nil {
			return err
		}

		runjob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		runjob.StepsStatus = stepsStatus
		if err := workflow_v2.UpdateJobRun(ctx, tx, runjob); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (api *API) postRunInfoHandler() ([]service.RbacChecker, service.Handler) {
	return []service.RbacChecker{api.jobRunUpdate}, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		jobRunID := vars["runJobID"]

		var runInfo sdk.V2WorkflowRunInfo
		if err := service.UnmarshalBody(r, &runInfo); err != nil {
			return err
		}

		runjob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
		if err != nil {
			return err
		}

		runInfo.WorkflowRunID = runjob.WorkflowRunID

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}
}

func (api *API) postJobRunInfoHandler() ([]service.RbacChecker, service.Handler) {
	return []service.RbacChecker{api.jobRunUpdate}, func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		jobRunID := vars["runJobID"]

		var jobInfo sdk.V2SendJobRunInfo
		if err := service.UnmarshalBody(r, &jobInfo); err != nil {
			return err
		}

		runjob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
		if err != nil {
			return err
		}

		runJobInfo := &sdk.V2WorkflowRunJobInfo{
			Level:            jobInfo.Level,
			Message:          jobInfo.Message,
			WorkflowRunID:    runjob.WorkflowRunID,
			WorkflowRunJobID: runjob.ID,
			IssuedAt:         jobInfo.Time,
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		if err := workflow_v2.InsertRunJobInfo(ctx, tx, runJobInfo); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}
}

func (api *API) getJobsQueuedHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunList),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			regionName := vars["regionName"]
			hatchConsumer := getHatcheryConsumer(ctx)
			u := getUserConsumer(ctx)
			switch {
			case hatchConsumer != nil:
				hatch, err := hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatchConsumer.AuthConsumerHatchery.HatcheryID)
				if err != nil {
					return err
				}
				jobs, err := workflow_v2.LoadQueuedRunJobByModelTypeAndRegion(ctx, api.mustDB(), regionName, hatch.ModelType)
				if err != nil {
					return err
				}
				return service.WriteJSON(w, jobs, http.StatusOK)
			case u != nil:
				// TODO
				// check permission region / project / admin
				return sdk.WithStack(sdk.ErrNotImplemented)
			}

			return nil
		}
}

func (api *API) postJobResultHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunUpdate),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]
			regionName := vars["regionName"]

			var result sdk.V2WorkflowRunJobResult
			if err := service.UnmarshalBody(req, &result); err != nil {
				return err
			}

			jobRun, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}
			if jobRun.Region != regionName {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unknown job %s on region %s", jobRun.ID, regionName)
			}

			if sdk.StatusIsTerminated(jobRun.Status) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "job %s is already in a final state %s", jobRun.JobID, jobRun.Status)
			}

			telemetry.MainSpan(ctx).AddAttributes(trace.StringAttribute(telemetry.TagJob, jobRun.JobID),
				trace.StringAttribute(telemetry.TagWorkflow, jobRun.WorkflowName),
				trace.StringAttribute(telemetry.TagProjectKey, jobRun.ProjectKey),
				trace.StringAttribute(telemetry.TagWorkflowRunNumber, strconv.FormatInt(jobRun.RunNumber, 10)))

			hatchConsumer := getHatcheryConsumer(ctx)
			hatch, err := hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatchConsumer.AuthConsumerHatchery.HatcheryID)
			if err != nil {
				return err
			}
			if jobRun.HatcheryName != hatch.Name {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			jobRun.Status = result.Status

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if result.Error != "" {
				jobInfo := sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    jobRun.WorkflowRunID,
					WorkflowRunJobID: jobRun.ID,
					Level:            sdk.WorkflowRunInfoLevelError,
					Message:          result.Error,
				}
				if err := workflow_v2.InsertRunJobInfo(ctx, tx, &jobInfo); err != nil {
					return err
				}
			}

			if err := workflow_v2.UpdateJobRun(ctx, tx, jobRun); err != nil {
				return err
			}
			if err := sdk.WithStack(tx.Commit()); err != nil {
				return err
			}

			api.EnqueueWorkflowRun(ctx, jobRun.WorkflowRunID, jobRun.UserID, jobRun.WorkflowName, jobRun.RunNumber)

			api.GoRoutines.Exec(ctx, "postJobResultHandler.event", func(ctx context.Context) {
				run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), jobRun.WorkflowRunID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
				event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, run.Contexts.Git.Server, run.Contexts.Git.Repository, *jobRun)
			})

			return nil
		}
}

func (api *API) postJobRunResultHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunUpdate),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]

			runJob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}

			var runResult sdk.V2WorkflowRunResult
			if err := service.UnmarshalBody(req, &runResult); err != nil {
				return err
			}

			runResult.ID = sdk.UUID()
			runResult.WorkflowRunJobID = runJob.ID
			runResult.WorkflowRunID = runJob.WorkflowRunID
			runResult.RunAttempt = runJob.RunAttempt
			if runJob.Integrations != nil {
				runResult.ArtifactManagerIntegration = runJob.Integrations.ArtifactManager
			}

			if runResult.Status == "" {
				return sdk.WithStack(sdk.ErrWrongRequest)
			}

			if err := workflow_v2.InsertRunResult(ctx, api.mustDB(), &runResult); err != nil {
				return err
			}

			api.GoRoutines.Exec(ctx, "postJobRunResultHandler-"+runResult.ID, func(ctx context.Context) {
				run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), runJob.WorkflowRunID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
				event_v2.PublishRunJobRunResult(ctx, api.Cache, sdk.EventRunJobRunResultAdded, run.Contexts.Git.Server, run.Contexts.Git.Repository, *runJob, runResult)
			})

			return service.WriteJSON(w, runResult, http.StatusCreated)
		}
}

func (api *API) putJobRunResultHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunUpdate),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]

			runJob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}

			var runResult sdk.V2WorkflowRunResult
			if err := service.UnmarshalBody(req, &runResult); err != nil {
				return err
			}

			oldRunResult, err := workflow_v2.LoadRunResult(ctx, api.mustDB(), runJob.ID, runResult.ID)
			if err != nil {
				return err
			}

			// Check consistency
			if oldRunResult.WorkflowRunID != runResult.WorkflowRunID ||
				oldRunResult.WorkflowRunJobID != runResult.WorkflowRunJobID ||
				runResult.WorkflowRunJobID != runJob.ID {
				return sdk.WithStack(sdk.ErrWrongRequest)
			}

			if runResult.Status == "" {
				return sdk.WithStack(sdk.ErrWrongRequest)
			}

			if err := workflow_v2.UpdateRunResult(ctx, api.mustDB(), &runResult); err != nil {
				return err
			}

			api.GoRoutines.Exec(ctx, "putJobRunResultHandler-"+runResult.ID, func(ctx context.Context) {
				run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), runJob.WorkflowRunID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
				event_v2.PublishRunJobRunResult(ctx, api.Cache, sdk.EventRunJobRunResultUpdated, run.Contexts.Git.Server, run.Contexts.Git.Repository, *runJob, runResult)
			})

			return service.WriteJSON(w, runResult, http.StatusCreated)
		}
}

func (api *API) getJobRunResultHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			runJobID := vars["runJobID"]

			runJob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), runJobID)
			if err != nil {
				return err
			}

			runResultID := vars["runResultID"]

			runResult, err := workflow_v2.LoadRunResult(ctx, api.mustDB(), runJob.ID, runResultID)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, runResult, http.StatusCreated)
		}
}

func (api *API) getJobRunResultsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			runJobID := vars["runJobID"]

			runJob, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), runJobID)
			if err != nil {
				return err
			}

			runResults, err := workflow_v2.LoadRunResults(ctx, api.mustDB(), runJob.WorkflowRunID)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, runResults, http.StatusCreated)
		}
}

func (api *API) deleteHatcheryReleaseJobRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunUpdate, api.isHatchery),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]
			regionName := vars["regionName"]

			hatch := getHatcheryConsumer(ctx)

			jobRun, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}
			if jobRun.Region != regionName || jobRun.HatcheryName != hatch.Name {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unknown job %s on region %s taken by hatchery %s", jobRun.ID, regionName, hatch.Name)
			}

			jobRun.Status = sdk.StatusWaiting
			jobRun.HatcheryName = ""

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := workflow_v2.UpdateJobRun(ctx, tx, jobRun); err != nil {
				return err
			}

			info := sdk.V2WorkflowRunJobInfo{
				WorkflowRunID:    jobRun.WorkflowRunID,
				IssuedAt:         time.Now(),
				Level:            sdk.WorkflowRunInfoLevelWarning,
				Message:          hatch.Name + " stops working on the job " + jobRun.JobID,
				WorkflowRunJobID: jobRun.ID,
			}
			if err := workflow_v2.InsertRunJobInfo(ctx, tx, &info); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			// Enqueue the job
			api.GoRoutines.Exec(ctx, "deleteHatcheryReleaseJobRunHandler.event", func(ctx context.Context) {
				run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), jobRun.WorkflowRunID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
				event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnqueued, run.Contexts.Git.Server, run.Contexts.Git.Repository, *jobRun)
			})
			return nil
		}
}

func (api *API) postHatcheryTakeJobRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunUpdate, api.isHatchery),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]
			regionName := vars["regionName"]

			_, next := telemetry.Span(ctx, "api.postHatcheryTakeJobRunHandler.lock")
			lockKey := cache.Key(JobRunHatcheryTakeKey, jobRunID)
			b, err := api.Cache.Lock(lockKey, 30*time.Second, 0, 1)
			if err != nil {
				next()
				return err
			}
			if !b {
				log.Debug(ctx, "api.postHatcheryTakeJobRunHandler> jobRun %s is locked in cache", jobRunID)
				next()
				return sdk.ErrNotFound
			}
			next()
			defer func() {
				_ = api.Cache.Unlock(lockKey)
			}()

			hatchConsumer := getHatcheryConsumer(ctx)
			hatch, err := hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatchConsumer.AuthConsumerHatchery.HatcheryID)
			if err != nil {
				return err
			}

			jobRun, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}
			if jobRun.Region != regionName {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unknown job %s on region %s", jobRun.ID, regionName)
			}
			if sdk.StatusIsTerminated(jobRun.Status) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "job %s is already in a final state %s", jobRun.JobID, jobRun.Status)
			}

			telemetry.MainSpan(ctx).AddAttributes(trace.StringAttribute(telemetry.TagJob, jobRun.JobID),
				trace.StringAttribute(telemetry.TagWorkflow, jobRun.WorkflowName),
				trace.StringAttribute(telemetry.TagProjectKey, jobRun.ProjectKey),
				trace.StringAttribute(telemetry.TagWorkflowRunNumber, strconv.FormatInt(jobRun.RunNumber, 10)))

			if jobRun.Status != sdk.StatusWaiting {
				return sdk.WrapError(sdk.ErrNotFound, "job has already been taken by %s", jobRun.HatcheryName)
			}

			canTake, err := hatcheryCanGetJob(ctx, api.mustDB(), jobRun.Region, hatch.ID)
			if err != nil {
				return err
			}

			if !canTake {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			jobRun.HatcheryName = hatch.Name
			jobRun.Status = sdk.StatusScheduling
			jobRun.Scheduled = time.Now()

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			if err := workflow_v2.UpdateJobRun(ctx, tx, jobRun); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			api.GoRoutines.Exec(ctx, "postHatcheryTakeJobRunHandler", func(ctx context.Context) {
				run, err := workflow_v2.LoadRunByID(ctx, api.mustDB(), jobRun.WorkflowRunID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
				event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobScheduled, run.Contexts.Git.Server, run.Contexts.Git.Repository, *jobRun)
			})
			return service.WriteJSON(w, jobRun, http.StatusOK)

		}
}

func (api *API) getJobRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]
			regionName := vars["regionName"]

			jobRun, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}
			if jobRun.Region != regionName {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			hatch := getHatcheryConsumer(ctx)
			switch {
			case hatch != nil:
				if jobRun.HatcheryName == "" {
					canGet, err := hatcheryCanGetJob(ctx, api.mustDB(), jobRun.Region, hatch.AuthConsumerHatchery.HatcheryID)
					if err != nil {
						return err
					}
					if !canGet {
						return sdk.WithStack(sdk.ErrForbidden)
					}
				}
			default:
				// Manage worker / user rights
				return sdk.WithStack(sdk.ErrNotImplemented)
			}

			return service.WriteJSON(w, jobRun, http.StatusOK)
		}
}

func hatcheryCanGetJob(ctx context.Context, db gorp.SqlExecutor, regionName string, hatcheryID string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "hatcheryCanGetJob")
	defer next()

	reg, err := region.LoadRegionByName(ctx, db, regionName)
	if err != nil {
		return false, err
	}

	perm, err := rbac.LoadRBACByHatcheryID(ctx, db, hatcheryID)
	if err != nil {
		return false, err
	}

	canAccessJob := false
	for _, rbacHatch := range perm.Hatcheries {
		if rbacHatch.HatcheryID != hatcheryID {
			continue
		}
		if reg.ID == rbacHatch.RegionID {
			canAccessJob = true
		}
	}
	return canAccessJob, nil
}
