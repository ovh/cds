package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mitchellh/hashstructure"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) getWorkflowRunJobsV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}

			attemptS := FormString(req, "attempt")

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			attempt := wr.RunAttempt
			if attemptS != "" {
				attempt, err = strconv.ParseInt(attemptS, 10, 64)
				if err != nil {
					return err
				}
			}

			runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), wr.ID, attempt)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, runJobs, http.StatusOK)

		}
}

func (api *API) getWorkflowRunJobInfosHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}
			jobIdentifier := vars["jobIdentifier"]
			attemptS := FormString(req, "attempt")

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			attempt := wr.RunAttempt
			if attemptS != "" {
				attempt, err = strconv.ParseInt(attemptS, 10, 64)
				if err != nil {
					return err
				}
			}

			var runJob *sdk.V2WorkflowRunJob
			if sdk.IsValidUUID(jobIdentifier) {
				runJob, err = workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobIdentifier)
			} else {
				runJob, err = workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobIdentifier, attempt)
			}
			if err != nil {
				return err
			}

			infos, err := workflow_v2.LoadRunJobInfosByRunJobID(ctx, api.mustDB(), runJob.ID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, infos, http.StatusOK)
		}
}

func (api *API) getWorkflowRunJobHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}
			jobIdentifier := vars["jobIdentifier"]

			attemptS := FormString(req, "attempt")

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			attempt := wr.RunAttempt
			if attemptS != "" {
				attempt, err = strconv.ParseInt(attemptS, 10, 64)
				if err != nil {
					return err
				}
			}

			var runJob *sdk.V2WorkflowRunJob
			if sdk.IsValidUUID(jobIdentifier) {
				runJob, err = workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobIdentifier)
			} else {
				runJob, err = workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobIdentifier, attempt)
			}
			if err != nil {
				return err
			}
			return service.WriteJSON(w, runJob, http.StatusOK)
		}
}

func (api *API) postStopJobHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}
			jobIdentifier := vars["jobIdentifier"]
			attemptS := FormString(req, "attempt")

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			attempt := wr.RunAttempt
			if attemptS != "" {
				attempt, err = strconv.ParseInt(attemptS, 10, 64)
				if err != nil {
					return err
				}
			}

			var runJob *sdk.V2WorkflowRunJob
			if sdk.IsValidUUID(jobIdentifier) {
				runJob, err = workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobIdentifier)
			} else {
				runJob, err = workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobIdentifier, attempt)
			}
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			runJob.Status = sdk.StatusStopped
			if err := workflow_v2.UpdateJobRun(ctx, tx, runJob); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, wr.Contexts.Git.Server, wr.Contexts.Git.Repository, *runJob)
			api.EnqueueWorkflowRun(ctx, runJob.WorkflowRunID, runJob.UserID, runJob.WorkflowName, runJob.RunNumber)

			return nil
		}
}

func (api *API) getWorkflowRunJobLogsLinksV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}
			jobIdentifier := vars["jobIdentifier"]
			attemptS := FormString(req, "attempt")

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			attempt := wr.RunAttempt
			if attemptS != "" {
				attempt, err = strconv.ParseInt(attemptS, 10, 64)
				if err != nil {
					return err
				}
			}

			var runJob *sdk.V2WorkflowRunJob
			if sdk.IsValidUUID(jobIdentifier) {
				runJob, err = workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobIdentifier)
			} else {
				runJob, err = workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobIdentifier, attempt)
			}
			if err != nil {
				return err
			}

			refs := make([]sdk.CDNLogAPIRefV2, 0)
			apiRef := sdk.CDNLogAPIRefV2{
				ProjectKey:   proj.Key,
				WorkflowName: wr.WorkflowName,
				RunID:        wr.ID,
				RunJobName:   runJob.JobID,
				RunJobID:     runJob.ID,
				RunNumber:    runJob.RunNumber,
				RunAttempt:   runJob.RunAttempt,
			}

			for serviceName := range runJob.Job.Services {
				ref := apiRef
				ref.ServiceName = serviceName
				ref.ItemType = sdk.CDNTypeItemServiceLogV2
				refs = append(refs, ref)
			}

			for k := range runJob.StepsStatus {
				stepOrder := -1
				for i := range runJob.Job.Steps {
					stepName := sdk.GetJobStepName(runJob.Job.Steps[i].ID, i)
					if stepName == k {
						stepOrder = i
						break
					}
				}

				if stepOrder == -1 {
					continue
				}
				ref := apiRef
				ref.StepName = sdk.GetJobStepName(k, stepOrder)
				ref.StepOrder = int64(stepOrder)
				ref.ItemType = sdk.CDNTypeItemJobStepLog
				refs = append(refs, ref)
			}
			datas := make([]sdk.CDNLogLinkData, 0, len(refs))
			for _, r := range refs {
				apiRefHashU, err := hashstructure.Hash(r, nil)
				if err != nil {
					return sdk.WithStack(err)
				}
				apiRefHash := strconv.FormatUint(apiRefHashU, 10)
				datas = append(datas, sdk.CDNLogLinkData{
					APIRef:      apiRefHash,
					StepOrder:   r.StepOrder,
					StepName:    r.StepName,
					ServiceName: r.ServiceName,
					ItemType:    r.ItemType,
				})
			}

			httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
			if err != nil {
				return err
			}

			return service.WriteJSON(w, sdk.CDNLogLinks{
				CDNURL: httpURL,
				Data:   datas,
			}, http.StatusOK)
		}
}

func (api *API) getWorkflowRunInfoV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			infos, err := workflow_v2.LoadRunInfosByRunID(ctx, api.mustDB(), wr.ID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, infos, http.StatusOK)
		}
}

func (api *API) getWorkflowRunV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber, workflow_v2.WithRunResults)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func (api *API) getWorkflowRunsFiltersV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			consumer := getUserConsumer(ctx)

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			actors, err := workflow_v2.LoadRunsActors(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			workflowNames, err := workflow_v2.LoadRunsWorkflowNames(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			refs, err := workflow_v2.LoadRunsGitRefs(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			filters := []sdk.V2WorkflowRunSearchFilter{
				{
					Key:     "actor",
					Options: actors,
					Example: consumer.GetUsername(),
				},
				{
					Key:     "workflow",
					Options: workflowNames,
					Example: "workflow-name",
				},
				{
					Key:     "branch",
					Options: refs,
					Example: "branch-name",
				},
				{
					Key:     "status",
					Options: []string{sdk.StatusFail, sdk.StatusSuccess, sdk.StatusBuilding, sdk.StatusStopped},
					Example: "Success, Failure, etc.",
				},
			}

			return service.WriteJSON(w, filters, http.StatusOK)
		}
}

func (api *API) getWorkflowRunsSearchV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			offset := service.FormUInt(req, "offset")
			limit := service.FormUInt(req, "limit")

			filters := workflow_v2.SearchsRunsFilters{
				Workflows: req.URL.Query()["workflow"],
				Actors:    req.URL.Query()["actor"],
				Status:    req.URL.Query()["status"],
				Branches:  req.URL.Query()["branch"],
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			count, err := workflow_v2.CountRuns(ctx, api.mustDB(), proj.Key, filters)
			if err != nil {
				return err
			}

			runs, err := workflow_v2.SearchRuns(ctx, api.mustDB(), proj.Key, filters, offset, limit)
			if err != nil {
				return err
			}

			w.Header().Add("X-Total-Count", fmt.Sprintf("%d", count))

			return service.WriteJSON(w, runs, http.StatusOK)
		}
}

func (api *API) getWorkflowRunsV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			runs, err := workflow_v2.LoadRuns(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, workflow_v2.WithRunResults)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, runs, http.StatusOK)
		}
}

func (api *API) postStopWorkflowRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			runJobs, err := workflow_v2.LoadRunJobsByRunIDAndStatus(ctx, api.mustDB(), wr.ID, []string{sdk.StatusWaiting, sdk.StatusBuilding, sdk.StatusScheduling})
			if err != nil {
				return err
			}

			for _, rj := range runJobs {
				rj.Status = sdk.StatusStopped

				tx, err := api.mustDB().Begin()
				if err != nil {
					return err
				}

				if err := workflow_v2.UpdateJobRun(ctx, tx, &rj); err != nil {
					_ = tx.Rollback()
					return err
				}

				runJobInfo := sdk.V2WorkflowRunJobInfo{
					WorkflowRunID:    rj.WorkflowRunID,
					WorkflowRunJobID: rj.ID,
					IssuedAt:         time.Now(),
					Level:            sdk.WorkflowRunInfoLevelInfo,
					Message:          fmt.Sprintf("User %s stopped the job", u.GetUsername()),
				}
				if err := workflow_v2.InsertRunJobInfo(ctx, tx, &runJobInfo); err != nil {
					_ = tx.Rollback()
					return err
				}

				if err := tx.Commit(); err != nil {
					return sdk.WithStack(err)
				}
			}
			wr.Status = sdk.StatusStopped
			tx, err := api.mustDB().Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback() // nolint

			if err := workflow_v2.UpdateRun(ctx, tx, wr); err != nil {
				return err
			}

			runInfo := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wr.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelInfo,
				Message:       fmt.Sprintf("User %s stopped the workflow", u.GetUsername()),
			}
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			for _, rj := range runJobs {
				event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, wr.Contexts.Git.Server, wr.Contexts.Git.Repository, rj)
			}
			event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunEnded, *wr, *u.AuthConsumerUser.AuthentifiedUser)

			return nil
		}
}

func (api *API) postWorkflowRunFromHookV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]

			var runRequest sdk.V2WorkflowRunHookRequest
			if err := service.UnmarshalRequest(ctx, req, &runRequest); err != nil {
				return err
			}

			ctx = context.WithValue(ctx, cdslog.HookEventID, runRequest.HookEventID)

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			ref, err := api.getEntityRefFromQueryParams(ctx, req, pKey, vcsProject.Name, repo.Name)
			if err != nil {
				return err
			}

			workflowEntity, err := entity.LoadByRefTypeName(ctx, api.mustDB(), repo.ID, ref, sdk.EntityTypeWorkflow, workflowName)
			if err != nil {
				return err
			}

			var wk sdk.V2Workflow
			if err := yaml.Unmarshal([]byte(workflowEntity.Data), &wk); err != nil {
				return err
			}

			u, err := user.LoadByID(ctx, api.mustDB(), runRequest.UserID)
			if err != nil {
				return err
			}

			hasRole, err := rbac.HasRoleOnWorkflowAndUserID(ctx, api.mustDB(), sdk.WorkflowRoleTrigger, u.ID, proj.Key, wk.Name)
			if err != nil {
				return err
			}
			if !hasRole {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			runEvent := sdk.V2WorkflowRunEvent{}
			switch runRequest.HookType {
			case sdk.WorkflowHookTypeWorkerModel:
				runEvent.ModelUpdateTrigger = &sdk.ModelUpdateTrigger{
					Ref:          runRequest.Ref,
					ModelUpdated: runRequest.EntityUpdated,
				}
			case sdk.WorkflowHookTypeWorkflow:
				runEvent.WorkflowUpdateTrigger = &sdk.WorkflowUpdateTrigger{
					Ref:             runRequest.Ref,
					WorkflowUpdated: runRequest.EntityUpdated,
				}
			case sdk.WorkflowHookTypeRepository:
				runEvent.GitTrigger = &sdk.GitTrigger{
					Payload:       runRequest.Payload,
					EventName:     runRequest.EventName,
					Ref:           runRequest.Ref,
					Sha:           runRequest.Sha,
					SemverCurrent: runRequest.SemverCurrent,
					SemverNext:    runRequest.SemverNext,
				}
			default:
				return sdk.WrapError(sdk.ErrWrongRequest, "unknown event: %v", runRequest)

			}
			wr, err := api.startWorkflowV2(ctx, *proj, *vcsProject, *repo, *workflowEntity, wk, runEvent, u)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func (api *API) putWorkflowRunV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}

			if !sdk.StatusIsTerminated(wr.Status) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to rerun a running workflow")
			}

			runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), wr.ID, wr.RunAttempt)
			if err != nil {
				return err
			}

			runJobsMap := make(map[string]sdk.V2WorkflowRunJob)
			runJobToRestart := make(map[string]sdk.V2WorkflowRunJob)
			for _, rj := range runJobs {
				runJobsMap[rj.ID] = rj
				if rj.Status == sdk.StatusFail {
					runJobToRestart[rj.ID] = rj
				}
			}
			if len(runJobToRestart) == 0 {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "workflow doesn't contains failed jobs")
			}

			runJobsToKeep := workflow_v2.RetrieveJobToKeep(ctx, wr.WorkflowData.Workflow, runJobsMap, runJobToRestart)

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := restartWorkflowRun(ctx, tx, wr, runJobsToKeep); err != nil {
				return err
			}

			runInfo := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wr.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelInfo,
				Message:       u.GetFullname() + " restarted all failed jobs",
			}
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunRestartFailedJob, *wr, *u.AuthConsumerUser.AuthentifiedUser)

			// Then continue the workflow
			api.EnqueueWorkflowRun(ctx, wr.ID, u.AuthConsumerUser.AuthentifiedUserID, wr.WorkflowName, wr.RunNumber)
			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func restartWorkflowRun(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, wr *sdk.V2WorkflowRun, runJobsToKeep map[string]sdk.V2WorkflowRunJob) error {
	wr.RunAttempt++
	wr.Status = sdk.StatusBuilding
	wr.Contexts.CDS.RunAttempt = wr.RunAttempt

	srvs, err := services.LoadAllByType(ctx, tx, sdk.TypeCDN)
	if err != nil {
		return err
	}

	// Duplicate runJob to keep
	for _, rj := range runJobsToKeep {
		duplicatedRJ := rj
		duplicatedRJ.ID = ""
		duplicatedRJ.RunAttempt = wr.RunAttempt
		if err := workflow_v2.InsertRunJob(ctx, tx, &duplicatedRJ); err != nil {
			return err
		}
		runResults, err := workflow_v2.LoadRunResultsByRunJobID(ctx, tx, rj.ID)
		if err != nil {
			return err
		}
		for _, r := range runResults {
			duplicatedRunResult := r
			duplicatedRunResult.ID = ""
			duplicatedRunResult.WorkflowRunJobID = duplicatedRJ.ID
			duplicatedRunResult.RunAttempt = duplicatedRJ.RunAttempt
			if err := workflow_v2.InsertRunResult(ctx, tx, &duplicatedRunResult); err != nil {
				return err
			}
		}
		req := sdk.CDNDuplicateItemRequest{FromJob: rj.ID, ToJob: duplicatedRJ.ID}
		_, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/item/duplicate", req, nil)
		if err != nil || code >= 400 {
			return fmt.Errorf("unable to duplicate cdn item for runjob %s. Code result %d: %v", rj.ID, code, err)
		}
	}

	if err := workflow_v2.UpdateRun(ctx, tx, wr); err != nil {
		return err
	}
	return nil
}

func (api *API) putWorkflowRunJobV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}
			jobIdentifier := vars["jobIdentifier"]

			var gateInputs map[string]interface{}
			if err := service.UnmarshalBody(req, &gateInputs); err != nil {
				return err
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, proj.Key, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber, workflow_v2.WithRunResults)
			if err != nil {
				return err
			}
			runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), wr.ID, wr.RunAttempt)
			if err != nil {
				return err
			}

			var jobToRun *sdk.V2WorkflowRunJob
			if sdk.IsValidUUID(jobIdentifier) {
				jobToRun, err = workflow_v2.LoadRunJobByID(ctx, api.mustDB(), jobIdentifier)
			} else {
				jobToRun, err = workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobIdentifier, wr.RunAttempt)
			}
			if err != nil {
				return err
			}

			// Check job status
			if jobToRun.Status != sdk.StatusSkipped {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to run manually a non skipped job")
			}

			// Gate check
			if jobToRun.Job.Gate == "" {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "there is no gate on job %s", jobToRun.JobID)
			}
			// Check gate reviewers
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			gate := wr.WorkflowData.Workflow.Gates[jobToRun.Job.Gate]
			reviewersChecked := len(gate.Reviewers.Users) == 0 && len(gate.Reviewers.Groups) == 0
			if len(gate.Reviewers.Users) > 0 {
				if sdk.IsInArray(u.GetUsername(), gate.Reviewers.Users) {
					reviewersChecked = true
				}
			}
			if !reviewersChecked && len(gate.Reviewers.Groups) > 0 {
			groupLoop:
				for _, g := range gate.Reviewers.Groups {
					grp, err := group.LoadByName(ctx, api.mustDBWithCtx(ctx), g, group.LoadOptions.WithMembers)
					if err != nil {
						return err
					}
					for _, m := range grp.Members {
						if m.Username == u.GetUsername() {
							reviewersChecked = true
							break groupLoop
						}
					}
				}
			}
			if !reviewersChecked {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "you are not part of the reviewers")
			}

			inputs := make(map[string]interface{})
			for k, v := range gate.Inputs {
				inputs[k] = v.Default
			}
			// Check Gate inputs
			for k, v := range gateInputs {
				if _, has := inputs[k]; has {
					inputs[k] = v
				}
			}

			// Check gate condition
			// retrieve previous jobs context
			runJobsContexts := computeExistingRunJobContexts(*wr, runJobs)
			jobContext := buildContextForJob(ctx, wr.WorkflowData.Workflow.Jobs, runJobsContexts, wr.Contexts, jobToRun.JobID)
			jobContext.Gate = inputs
			bts, err := json.Marshal(jobContext)
			if err != nil {
				return sdk.WithStack(err)
			}

			var mapContexts map[string]interface{}
			if err := json.Unmarshal(bts, &mapContexts); err != nil {
				return sdk.WithStack(err)
			}
			ap := sdk.NewActionParser(mapContexts, sdk.DefaultFuncs)
			booleanResult, err := ap.InterpolateToBool(ctx, gate.If)
			if err != nil {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "job %s: gate statement %s doesn't return a boolean: %v", jobToRun.JobID, gate.If, err)
			}

			if !booleanResult {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "gate conditions are not satisfied")
			}
			//////////

			runJobsMap := make(map[string]sdk.V2WorkflowRunJob)
			for _, rj := range runJobs {
				runJobsMap[rj.ID] = rj
			}

			runJobsToKeep := workflow_v2.RetrieveJobToKeep(ctx, wr.WorkflowData.Workflow, runJobsMap, map[string]sdk.V2WorkflowRunJob{jobToRun.ID: *jobToRun})

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := restartWorkflowRun(ctx, tx, wr, runJobsToKeep); err != nil {
				return err
			}
			wr.RunJobEvent = append(wr.RunJobEvent, sdk.V2WorkflowRunJobEvent{
				Inputs:     gateInputs,
				UserID:     u.AuthConsumerUser.AuthentifiedUserID,
				Username:   u.GetUsername(),
				JobID:      jobToRun.JobID,
				RunAttempt: wr.RunAttempt,
			})
			if err := workflow_v2.UpdateRun(ctx, tx, wr); err != nil {
				return err
			}

			runMsg := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wr.ID,
				Level:         sdk.WorkflowRunInfoLevelInfo,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("%s manually trigger the job %s", u.GetFullname(), jobToRun.JobID),
			}
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runMsg); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishRunJobManualEvent(ctx, api.Cache, sdk.EventRunJobManualTriggered, *wr, jobToRun.JobID, gateInputs, *u.AuthConsumerUser.AuthentifiedUser)

			// Then continue the workflow
			api.EnqueueWorkflowRun(ctx, wr.ID, u.AuthConsumerUser.AuthentifiedUserID, wr.WorkflowName, wr.RunNumber)
			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func (api *API) postWorkflowRunV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]

			var runRequest map[string]interface{}
			if err := service.UnmarshalBody(req, &runRequest); err != nil {
				return err
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			ref, err := api.getEntityRefFromQueryParams(ctx, req, pKey, vcsProject.Name, repo.Name)
			if err != nil {
				return err
			}

			workflowEntity, err := entity.LoadByRefTypeName(ctx, api.mustDB(), repo.ID, ref, sdk.EntityTypeWorkflow, workflowName)
			if err != nil {
				return err
			}

			var wk sdk.V2Workflow
			if err := yaml.Unmarshal([]byte(workflowEntity.Data), &wk); err != nil {
				return err
			}

			runEvent := sdk.V2WorkflowRunEvent{ // TODO handler semver ?
				Manual: &sdk.ManualTrigger{
					Payload: runRequest,
				},
			}

			u := getUserConsumer(ctx)
			wr, err := api.startWorkflowV2(ctx, *proj, *vcsProject, *repo, *workflowEntity, wk, runEvent, u.AuthConsumerUser.AuthentifiedUser)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, wr, http.StatusCreated)
		}
}

func (api *API) startWorkflowV2(ctx context.Context, proj sdk.Project, vcsProject sdk.VCSProject, repo sdk.ProjectRepository, wkEntity sdk.Entity, wk sdk.V2Workflow, runEvent sdk.V2WorkflowRunEvent, u *sdk.AuthentifiedUser) (*sdk.V2WorkflowRun, error) {
	log.Debug(ctx, "Start Workflow %s", wkEntity.Name)
	var msg string
	switch {
	case runEvent.Manual != nil:
		msg = fmt.Sprintf("Workflow was manually triggered by user %s", u.Username)
	case runEvent.GitTrigger != nil:
		msg = fmt.Sprintf("The workflow was triggered by the repository webhook event %s by user %s", runEvent.GitTrigger.EventName, u.Username)
	case runEvent.WorkflowUpdateTrigger != nil:
		msg = fmt.Sprintf("Workflow was triggered by the workflow_update hook by user %s", u.Username)
	case runEvent.ModelUpdateTrigger != nil:
		msg = fmt.Sprintf("Workflow was triggered by the model_update hook by user %s", u.Username)
	default:
		return nil, sdk.WrapError(sdk.ErrNotImplemented, "event not implemented")
	}

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsProject.ID,
		VCSServer:    vcsProject.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: wk.Name,
		WorkflowRef:  wkEntity.Ref,
		WorkflowSha:  wkEntity.Commit,
		Status:       sdk.StatusCrafting,
		RunAttempt:   0,
		Started:      time.Now(),
		LastModified: time.Now(),
		ToDelete:     false,
		WorkflowData: sdk.V2WorkflowRunData{Workflow: wk},
		UserID:       u.ID,
		Username:     u.Username,
		RunEvent:     runEvent,
		Contexts:     sdk.WorkflowRunContext{},
	}

	wrNumber, err := workflow_v2.WorkflowRunNextNumber(api.mustDB(), repo.ID, wk.Name)
	if err != nil {
		return nil, err
	}
	wr.RunNumber = wrNumber

	telemetry.MainSpan(ctx).AddAttributes(trace.StringAttribute(telemetry.TagWorkflowRunNumber, strconv.FormatInt(wrNumber, 10)))

	tx, err := api.mustDB().Begin()
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	defer tx.Rollback()

	wr.RunNumber = wrNumber
	if err := workflow_v2.InsertRun(ctx, tx, &wr); err != nil {
		return nil, err
	}

	runInfo := sdk.V2WorkflowRunInfo{
		WorkflowRunID: wr.ID,
		Level:         sdk.WorkflowRunInfoLevelInfo,
		IssuedAt:      time.Now(),
	}

	runInfo.Message = msg
	if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WithStack(err)
	}

	event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunCrafted, wr, *u)

	select {
	case api.workflowRunCraftChan <- wr.ID:
		log.Debug(ctx, "postWorkflowRunV2Handler: workflow run %s %d sent into chan", wr.WorkflowName, wr.RunNumber)
	default:
		// Default behaviour is made by a goroutine that call directly the database
	}

	return &wr, nil
}
