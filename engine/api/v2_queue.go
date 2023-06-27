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

func (api *API) getJobsQueuedHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunList),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {

			hatchConsumer := getHatcheryConsumer(ctx)
			u := getUserConsumer(ctx)
			switch {
			case hatchConsumer != nil:
				hatch, err := hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatchConsumer.AuthConsumerHatchery.HatcheryID)
				if err != nil {
					return err
				}
				hatchPerm, err := rbac.LoadRBACByHatcheryID(ctx, api.mustDB(), hatch.ID)
				if err != nil {
					return err
				}
				var regName string
				for _, p := range hatchPerm.Hatcheries {
					if p.HatcheryID == hatch.ID {
						reg, err := region.LoadRegionByID(ctx, api.mustDB(), p.RegionID)
						if err != nil {
							return err
						}
						regName = reg.Name
						break
					}
				}
				jobs, err := workflow_v2.LoadQueuedRunJobByModelTypeAndRegion(ctx, api.mustDB(), regName, hatch.ModelType)
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

			var result sdk.V2WorkflowRunJobResult
			if err := service.UnmarshalBody(req, &result); err != nil {
				return err
			}

			jobRun, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}

			telemetry.MainSpan(ctx).AddAttributes(trace.StringAttribute(telemetry.TagJob, jobRun.JobID),
				trace.StringAttribute(telemetry.TagWorkflow, jobRun.WorkflowName),
				trace.StringAttribute(telemetry.TagProjectKey, jobRun.ProjectKey),
				trace.StringAttribute(telemetry.TagWorkflowRunNumber, strconv.FormatInt(jobRun.RunNumber, 10)))

			hatchConsumer := getHatcheryConsumer(ctx)
			switch {
			case hatchConsumer != nil:
				hatch, err := hatchery.LoadHatcheryByID(ctx, api.mustDB(), hatchConsumer.AuthConsumerHatchery.HatcheryID)
				if err != nil {
					return err
				}
				if jobRun.HatcheryName != hatch.Name {
					return sdk.WithStack(sdk.ErrForbidden)
				}
			default:
				// TODO Manage worker job run update
				return sdk.WithStack(sdk.ErrNotImplemented)
			}

			jobRun.Status = result.Status

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

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

			// Continue workflow
			enqueueRequest := sdk.V2WorkflowRunEnqueue{
				RunID:  jobRun.WorkflowRunID,
				UserID: jobRun.UserID,
			}
			if err := api.Cache.Enqueue(workflow_v2.WorkflowEngineKey, enqueueRequest); err != nil {
				return err
			}
			return sdk.WithStack(tx.Commit())
		}
}

func (api *API) postHatcheryTakeJobRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunUpdate),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]

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
			jobRun.Status = sdk.StatusCrafting

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

			return service.WriteJSON(w, jobRun, http.StatusOK)

		}
}

func (api *API) getJobRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.jobRunRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			jobRunID := vars["runJobID"]

			jobRun, err := workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobRunID)
			if err != nil {
				return err
			}

			hatch := getHatcheryConsumer(ctx)
			switch {
			case hatch != nil:
				canGet, err := hatcheryCanGetJob(ctx, api.mustDB(), jobRun.Region, hatch.AuthConsumerHatchery.HatcheryID)
				if err != nil {
					return err
				}
				if !canGet {
					return sdk.WithStack(sdk.ErrForbidden)
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
