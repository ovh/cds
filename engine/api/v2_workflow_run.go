package api

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/mitchellh/hashstructure"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/purge"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
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
			workflowRunID := vars["workflowRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			attemptS := FormString(req, "attempt")

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

func (api *API) getWorkflowRunResultsV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			workflowRunID := vars["workflowRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			attemptS := FormString(req, "attempt")

			attempt := wr.RunAttempt
			if attemptS != "" {
				attempt, err = strconv.ParseInt(attemptS, 10, 64)
				if err != nil {
					return err
				}
			}

			runResults, err := workflow_v2.LoadRunResultsByRunIDAttempt(ctx, api.mustDB(), wr.ID, attempt)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, runResults, http.StatusOK)
		}
}

func (api *API) getWorkflowRunJobInfosHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			workflowRunID := vars["workflowRunID"]
			jobRunID := vars["jobRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			runJob, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, api.mustDB(), wr.ID, jobRunID)
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
			workflowRunID := vars["workflowRunID"]
			jobRunID := vars["jobRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			runJob, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, api.mustDB(), wr.ID, jobRunID)
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
			workflowRunID := vars["workflowRunID"]
			jobIdentifier := vars["jobIdentifier"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			attemptS := FormString(req, "attempt")

			attempt := wr.RunAttempt
			if attemptS != "" {
				attempt, err = strconv.ParseInt(attemptS, 10, 64)
				if err != nil {
					return err
				}
			}

			var runJobs []sdk.V2WorkflowRunJob
			if sdk.IsValidUUID(jobIdentifier) {
				runJob, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, api.mustDB(), wr.ID, jobIdentifier)
				if err != nil {
					return err
				}
				runJobs = []sdk.V2WorkflowRunJob{*runJob}
			} else {
				runJobs, err = workflow_v2.LoadRunJobsByName(ctx, api.mustDB(), wr.ID, jobIdentifier, attempt)
				if err != nil {
					return err
				}
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			for i := range runJobs {
				runJobs[i].Status = sdk.V2WorkflowRunJobStatusStopped
				now := time.Now()
				runJobs[i].Ended = &now
				if err := workflow_v2.UpdateJobRun(ctx, tx, &runJobs[i]); err != nil {
					return err
				}
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			for i := range runJobs {
				event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, *wr, runJobs[i])
			}
			api.EnqueueWorkflowRun(ctx, wr.ID, u.AuthConsumerUser.AuthentifiedUserID, wr.WorkflowName, wr.RunNumber, isAdmin(ctx))

			return nil
		}
}

func (api *API) getWorkflowRunJobLogsLinksV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			workflowRunID := vars["workflowRunID"]
			jobRunID := vars["jobRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			runJob, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, api.mustDB(), wr.ID, jobRunID)
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
					} else if k == "Post-"+stepName {
						stepOrder = len(runJob.Job.Steps)*2 - 1 - i
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
			slices.SortFunc(refs, func(i, j sdk.CDNLogAPIRefV2) int {
				return cmp.Compare(i.StepOrder, j.StepOrder)
			})
			datas := make([]sdk.CDNLogLinkData, 0, len(refs))
			for i, r := range refs {
				r.StepOrder = int64(i)
				apiRefHashU, err := hashstructure.Hash(r, nil)
				if err != nil {
					return sdk.WithStack(err)
				}
				apiRefHash := strconv.FormatUint(apiRefHashU, 10)
				datas = append(datas, sdk.CDNLogLinkData{
					APIRef:      apiRefHash,
					StepName:    r.StepName,
					ServiceName: r.ServiceName,
					ItemType:    r.ItemType,
					StepOrder:   r.StepOrder,
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
			workflowRunID := vars["workflowRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
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
			workflowRunID := vars["workflowRunID"]

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func (api *API) deleteWorkflowRunV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			workflowRunID := vars["workflowRunID"]

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			if err := purge.WorkflowRunV2(ctx, api.mustDB(), wr.ID); err != nil {
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

			workflowRefs, err := workflow_v2.LoadRunsWorkflowRefs(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			refs, err := workflow_v2.LoadRunsGitRefs(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			repositories, err := workflow_v2.LoadRunsGitRepositories(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			workflowRepositories, err := workflow_v2.LoadRunsWorkflowRepositories(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			templates, err := workflow_v2.LoadRunsTemplates(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			annotations, err := workflow_v2.LoadRunsAnnotations(ctx, api.mustDB(), proj.Key)
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
					Example: "vcs_server/repository/workflow-name",
				},
				{
					Key:     "ref",
					Options: refs,
					Example: "ref/heads/main",
				},
				{
					Key:     "workflow_ref",
					Options: workflowRefs,
					Example: "ref/heads/main",
				},
				{
					Key:     "status",
					Options: []string{sdk.StatusFail, sdk.StatusSuccess, sdk.StatusBuilding, sdk.StatusStopped},
					Example: "Success, Failure, etc.",
				},
				{
					Key:     "repository",
					Options: repositories,
					Example: "vcs_server/repository",
				},
				{
					Key:     "workflow_repository",
					Options: workflowRepositories,
					Example: "vcs_server/repository",
				},
				{
					Key:     "template",
					Options: templates,
					Example: "vcs_server/repository/template-name",
				},
			}

			for _, x := range annotations {
				filters = append(filters, sdk.V2WorkflowRunSearchFilter{
					Key:     x.Key,
					Options: sdk.Unique(x.Values),
					Example: "Annotation value.",
				})
			}

			return service.WriteJSON(w, filters, http.StatusOK)
		}
}

func (api *API) getWorkflowRunsSearchAllProjectV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isAdmin),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			offset := service.FormUInt(req, "offset")
			limit := service.FormUInt(req, "limit")
			sort := req.FormValue("sort")

			filters := workflow_v2.SearchsRunsFilters{}

			for k, v := range req.URL.Query() {
				switch k {
				case "workflow":
					filters.Workflows = v
				case "actor":
					filters.Actors = v
				case "status":
					filters.Status = v
				case "ref":
					filters.Refs = v
				case "workflow_ref":
					filters.WorkflowRefs = v
				case "repository":
					filters.Repositories = v
				case "workflow_repository":
					filters.WorkflowRepositories = v
				case "commit":
					filters.Commits = v
				case "template":
					filters.Templates = v
				case "offset", "limit", "sort":
				default:
					filters.AnnotationKeys = append(filters.AnnotationKeys, k)
					filters.AnnotationValues = append(filters.AnnotationValues, v...)
				}
			}

			filters.Lower()

			count, err := workflow_v2.CountAllRuns(ctx, api.mustDB(), filters)
			if err != nil {
				return sdk.WrapError(err, "unable to count all runs")
			}
			if count == 0 {
				return service.WriteJSON(w, []sdk.V2WorkflowRun{}, http.StatusOK)
			}

			runs, err := workflow_v2.SearchAllRuns(ctx, api.mustDB(), filters, offset, limit, sort)
			if err != nil {
				return sdk.WrapError(err, "unable to search all runs")
			}

			w.Header().Add("X-Total-Count", fmt.Sprintf("%d", count))

			return service.WriteJSON(w, runs, http.StatusOK)
		}
}

func (api *API) getWorkflowRunsSearchV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			offset := service.FormUInt(req, "offset")
			limit := service.FormUInt(req, "limit")
			sort := req.FormValue("sort")

			filters := workflow_v2.SearchsRunsFilters{}

			for k, v := range req.URL.Query() {
				switch k {
				case "workflow":
					filters.Workflows = v
				case "actor":
					filters.Actors = v
				case "status":
					filters.Status = v
				case "ref":
					filters.Refs = v
				case "workflow_ref":
					filters.WorkflowRefs = v
				case "repository":
					filters.Repositories = v
				case "workflow_repository":
					filters.WorkflowRepositories = v
				case "commit":
					filters.Commits = v
				case "template":
					filters.Templates = v
				case "offset", "limit", "sort":
				default:
					filters.AnnotationKeys = append(filters.AnnotationKeys, k)
					filters.AnnotationValues = append(filters.AnnotationValues, v...)
				}
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			count, err := workflow_v2.CountRuns(ctx, api.mustDB(), proj.Key, filters)
			if err != nil {
				return sdk.WrapError(err, "unable to count runs")
			}

			runs, err := workflow_v2.SearchRuns(ctx, api.mustDB(), proj.Key, filters, offset, limit, sort)
			if err != nil {
				return sdk.WrapError(err, "unable to search runs")
			}

			w.Header().Add("X-Total-Count", fmt.Sprintf("%d", count))

			return service.WriteJSON(w, runs, http.StatusOK)
		}
}

func (api *API) postStopWorkflowRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			workflowRunID := vars["workflowRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			runJobs, err := workflow_v2.LoadRunJobsByRunIDAndStatus(ctx, api.mustDB(), wr.ID, []string{sdk.StatusWaiting, sdk.StatusBuilding, sdk.StatusScheduling})
			if err != nil {
				return err
			}

			for _, rj := range runJobs {
				rj.Status = sdk.V2WorkflowRunJobStatusStopped
				now := time.Now()
				rj.Ended = &now

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
			wr.Status = sdk.V2WorkflowRunStatusStopped
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
				event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, *wr, rj)
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

			ref, commit, err := api.getEntityRefFromQueryParams(ctx, req, pKey, vcsProject.Name, repo.Name)
			if err != nil {
				return err
			}
			if commit == "HEAD" {
				client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, proj.Key, vcsProject.Name)
				if err != nil {
					return err
				}
				switch {
				case strings.HasPrefix(ref, sdk.GitRefTagPrefix):
					tag, err := client.Tag(ctx, repo.Name, strings.TrimPrefix(ref, sdk.GitRefTagPrefix))
					if err != nil {
						return err
					}
					commit = tag.Hash
				default:
					branch, err := client.Branch(ctx, repo.Name, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(ref, sdk.GitRefBranchPrefix)})
					if err != nil {
						return err
					}
					commit = branch.LatestCommit
				}
			}

			workflowEntity, err := entity.LoadByRefTypeNameCommit(ctx, api.mustDB(), repo.ID, ref, sdk.EntityTypeWorkflow, workflowName, commit)
			if err != nil {
				return sdk.WrapError(err, "unable to get workflow %s for ref %s and commit %s", workflowName, ref, commit)
			}

			var wk sdk.V2Workflow
			if err := yaml.Unmarshal([]byte(workflowEntity.Data), &wk); err != nil {
				return err
			}

			if wk.Repository != nil && wk.Repository.InsecureSkipSignatureVerify {
				// Use entity owner as user fallback
				if workflowEntity.UserID == nil {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "unknown workflow owner. Please analyse your repository.")
				}
				runRequest.UserID = *workflowEntity.UserID
			}

			var u *sdk.AuthentifiedUser
			u, err = user.LoadByID(ctx, api.mustDB(), runRequest.UserID)
			if err != nil {
				return err
			}

			// Verify if user is admin
			if runRequest.AdminMFA && u.Ring != sdk.UserRingAdmin {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "user is not an administrator")
			}

			hasRole, err := rbac.HasRoleOnWorkflowAndUserID(ctx, api.mustDB(), sdk.WorkflowRoleTrigger, u.ID, proj.Key, vcsProject.Name, repo.Name, wk.Name)
			if err != nil {
				return err
			}
			if !hasRole {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			// Check runrequest git information regarding workflow
			if wk.Repository == nil || (wk.Repository.VCSServer == vcsProject.Name && wk.Repository.Name == repo.Name) {
				// git info must match between workflow def and target repository
				if (ref != runRequest.Ref && runRequest.Ref != "") || (commit != runRequest.Sha && runRequest.Sha != "") {
					return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to use a different commit")
				}
			}

			wr, err := api.startWorkflowV2(ctx, *proj, *vcsProject, *repo, *workflowEntity, wk, runRequest, u)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func (api *API) postRestartWorkflowRunHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			workflowRunID := vars["workflowRunID"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			if !wr.Status.IsTerminated() {
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
				if rj.Status == sdk.V2WorkflowRunJobStatusFail || rj.Status == sdk.V2WorkflowRunJobStatusStopped {
					runJobToRestart[rj.ID] = rj
				}
			}
			if len(runJobToRestart) == 0 {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "workflow doesn't contains failed or stopped jobs")
			}

			runJobsToKeep := workflow_v2.RetrieveJobToKeep(ctx, wr.WorkflowData.Workflow, runJobsMap, runJobToRestart)

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := api.restartWorkflowRun(ctx, tx, wr, runJobsToKeep); err != nil {
				return err
			}

			runInfo := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wr.ID,
				IssuedAt:      time.Now(),
				Level:         sdk.WorkflowRunInfoLevelInfo,
				Message:       u.GetFullname() + " restarted all failed and stopped jobs",
			}
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishRunEvent(ctx, api.Cache, sdk.EventRunRestartFailedJob, *wr, *u.AuthConsumerUser.AuthentifiedUser)

			// Then continue the workflow
			api.EnqueueWorkflowRun(ctx, wr.ID, u.AuthConsumerUser.AuthentifiedUserID, wr.WorkflowName, wr.RunNumber, isAdmin(ctx))
			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func (api *API) restartWorkflowRun(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, wr *sdk.V2WorkflowRun, runJobsToKeep map[string]sdk.V2WorkflowRunJob) error {
	wr.RunAttempt++
	wr.Status = sdk.V2WorkflowRunStatusBuilding
	wr.Contexts.CDS.RunAttempt = wr.RunAttempt

	srvs, err := services.LoadAllByType(ctx, tx, sdk.TypeCDN)
	if err != nil {
		return err
	}

	wg := new(sync.WaitGroup)
	chanErr := make(chan error, len(runJobsToKeep))

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
			duplicatedRunResult.ID = sdk.UUID()
			duplicatedRunResult.WorkflowRunJobID = duplicatedRJ.ID
			duplicatedRunResult.RunAttempt = duplicatedRJ.RunAttempt
			if err := workflow_v2.InsertRunResult(ctx, tx, &duplicatedRunResult); err != nil {
				return err
			}
		}

		wg.Add(1)
		api.GoRoutines.Exec(ctx, "CDNDuplicateItem-"+rj.ID, func(ctx context.Context) {
			defer wg.Done()
			req := sdk.CDNDuplicateItemRequest{FromJob: rj.ID, ToJob: duplicatedRJ.ID}
			_, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/item/duplicate", req, nil)
			if err != nil || code >= 400 {
				log.ErrorWithStackTrace(ctx, err)
				chanErr <- fmt.Errorf("unable to duplicate cdn item for runjob %s. Code result %d: %v", rj.ID, code, err)
			}
		})
	}
	wg.Wait()
	close(chanErr)

	for err := range chanErr {
		if err != nil {
			return err
		}
	}

	if err := workflow_v2.UpdateRun(ctx, tx, wr); err != nil {
		return err
	}
	return nil
}

func (api *API) postRunJobHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			workflowRunID := vars["workflowRunID"]
			jobIdentifier := vars["jobIdentifier"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			wr, err := workflow_v2.LoadRunByProjectKeyAndID(ctx, api.mustDB(), proj.Key, workflowRunID)
			if err != nil {
				return err
			}

			// Load current job and runresult
			runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), wr.ID, wr.RunAttempt)
			if err != nil {
				return err
			}
			runResults, err := workflow_v2.LoadRunResultsByRunIDAttempt(ctx, api.mustDB(), wr.ID, wr.RunAttempt)
			if err != nil {
				return err
			}

			if !wr.Status.IsTerminated() {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to start a job on a running workflow")
			}

			var jobToRuns []sdk.V2WorkflowRunJob
			if sdk.IsValidUUID(jobIdentifier) {
				jobToRun, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, api.mustDB(), wr.ID, jobIdentifier)
				if err != nil {
					return err
				}
				jobToRuns = []sdk.V2WorkflowRunJob{*jobToRun}
			} else {
				jobToRuns, err = workflow_v2.LoadRunJobsByName(ctx, api.mustDB(), wr.ID, jobIdentifier, wr.RunAttempt)
				if err != nil {
					return err
				}
			}
			if len(jobToRuns) == 0 {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "no job found for given identifier %q", jobIdentifier)
			}

			for _, jtr := range jobToRuns {
				if jtr.Status == sdk.V2WorkflowRunJobStatusSkipped && jtr.Job.Gate == "" {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "unable to start a skipped job without a gate")
				}
			}

			// If all run are skipped, check gate inputs
			if jobToRuns[0].Job.Gate != "" {
				// For job matrix, make sure that all job run contains the same gate definition
				var gateName string
				for _, j := range jobToRuns {
					if gateName == "" {
						gateName = j.Job.Gate
						continue
					}
					if j.Job.Gate != gateName {
						return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid gate condition detected on job matrix")
					}
				}
			}

			var inputs map[string]interface{}
			if err := service.UnmarshalBody(req, &inputs); err != nil {
				return err
			}
			if inputs == nil {
				// Retrieve inputs from last run if exists
				for _, je := range wr.RunJobEvent {
					if je.RunAttempt == wr.RunAttempt && je.JobID == jobToRuns[0].JobID {
						inputs = je.Inputs
						break
					}
				}
			}

			runJobsContexts, _ := computeExistingRunJobContexts(ctx, runJobs, runResults)
			jobContext := buildContextForJob(ctx, wr.WorkflowData.Workflow.Jobs, runJobsContexts, wr.Contexts, jobToRuns[0].JobID)
			booleanResult, err := checkJobCondition(ctx, api.mustDBWithCtx(ctx), *wr, inputs, jobToRuns[0].Job, jobContext, *u.AuthConsumerUser.AuthentifiedUser, isAdmin(ctx))
			if err != nil {
				return err
			}
			if !booleanResult {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "gate conditions are not satisfied")
			}

			runJobsMap := make(map[string]sdk.V2WorkflowRunJob)
			for _, rj := range runJobs {
				runJobsMap[rj.ID] = rj
			}
			runJobsToKeepMap := make(map[string]sdk.V2WorkflowRunJob)
			for _, rj := range jobToRuns {
				runJobsToKeepMap[rj.ID] = rj
			}
			runJobsToKeep := workflow_v2.RetrieveJobToKeep(ctx, wr.WorkflowData.Workflow, runJobsMap, runJobsToKeepMap)

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := api.restartWorkflowRun(ctx, tx, wr, runJobsToKeep); err != nil {
				return err
			}
			wr.RunJobEvent = append(wr.RunJobEvent, sdk.V2WorkflowRunJobEvent{
				Inputs:     inputs,
				UserID:     u.AuthConsumerUser.AuthentifiedUserID,
				Username:   u.GetUsername(),
				JobID:      jobToRuns[0].JobID,
				RunAttempt: wr.RunAttempt,
			})
			if err := workflow_v2.UpdateRun(ctx, tx, wr); err != nil {
				return err
			}

			runMsg := sdk.V2WorkflowRunInfo{
				WorkflowRunID: wr.ID,
				Level:         sdk.WorkflowRunInfoLevelInfo,
				IssuedAt:      time.Now(),
				Message:       fmt.Sprintf("%s has manually triggered the job %q", u.GetFullname(), jobToRuns[0].JobID),
			}
			if err := workflow_v2.InsertRunInfo(ctx, tx, &runMsg); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishRunJobManualEvent(ctx, api.Cache, sdk.EventRunJobManualTriggered, *wr, jobToRuns[0].JobID, inputs, *u.AuthConsumerUser.AuthentifiedUser)

			// Then continue the workflow
			api.EnqueueWorkflowRun(ctx, wr.ID, u.AuthConsumerUser.AuthentifiedUserID, wr.WorkflowName, wr.RunNumber, isAdmin(ctx))
			return service.WriteJSON(w, wr, http.StatusOK)
		}
}

func (api *API) postWorkflowRunV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workflowTrigger),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WrapError(sdk.ErrForbidden, "no user consumer")
			}

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

			var runRequest sdk.V2WorkflowRunManualRequest
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

			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, pKey, vcsProject.Name)
			if err != nil {
				return err
			}

			var workflowRef, workflowCommit string
			if runRequest.WorkflowBranch != "" {
				workflowRef = sdk.GitRefBranchPrefix + runRequest.WorkflowBranch
			} else if runRequest.WorkflowTag != "" {
				workflowRef = sdk.GitRefTagPrefix + runRequest.WorkflowTag
			} else if runRequest.Branch != "" {
				workflowRef = sdk.GitRefBranchPrefix + runRequest.Branch
				workflowCommit = runRequest.Sha
			} else if runRequest.Tag != "" {
				workflowRef = sdk.GitRefTagPrefix + runRequest.Tag
				workflowCommit = runRequest.Sha
			} else {
				// Retrieve default branch
				defaultBranch, err := vcsClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					return err
				}
				workflowRef = defaultBranch.ID
				workflowCommit = defaultBranch.LatestCommit
			}
			if workflowCommit == "" || workflowCommit == "HEAD" {
				switch {
				case strings.HasPrefix(workflowRef, sdk.GitRefBranchPrefix):
					// Retrieve branch to get commit
					b, err := vcsClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{BranchName: strings.TrimPrefix(workflowRef, sdk.GitRefBranchPrefix)})
					if err != nil {
						return err
					}
					workflowCommit = b.LatestCommit
				default:
					// Retrieve branch to get commit
					t, err := vcsClient.Tag(ctx, repo.Name, strings.TrimPrefix(workflowRef, sdk.GitRefTagPrefix))
					if err != nil {
						return err
					}
					workflowCommit = t.Hash
				}
			}

			hookRequest := sdk.HookManualWorkflowRun{
				UserRequest:    runRequest,
				Project:        proj.Key,
				VCSServer:      vcsProject.Name,
				Repository:     repo.Name,
				WorkflowRef:    workflowRef,
				WorkflowCommit: workflowCommit,
				Workflow:       workflowName,
				UserID:         u.AuthConsumerUser.AuthentifiedUserID,
				Username:       u.AuthConsumerUser.AuthentifiedUser.Username,
				AdminMFA:       isAdmin(ctx),
			}

			// Send start request to hooks
			srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeHooks)
			if err != nil {
				return err
			}
			var hookResponse sdk.HookRepositoryEvent
			_, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/v2/workflow/manual", hookRequest, &hookResponse)
			if err != nil || code >= 400 {
				return sdk.WrapError(err, "unable to start workflow")
			}

			runResponse := sdk.V2WorkflowRunManualResponse{
				HookEventUUID: hookResponse.UUID,
				UIUrl:         api.Config.URL.UI,
			}

			return service.WriteJSON(w, runResponse, http.StatusCreated)
		}
}

func (api *API) startWorkflowV2(ctx context.Context, proj sdk.Project, vcsProject sdk.VCSProject, repo sdk.ProjectRepository, wkEntity sdk.Entity, wk sdk.V2Workflow, runRequest sdk.V2WorkflowRunHookRequest, u *sdk.AuthentifiedUser) (*sdk.V2WorkflowRun, error) {
	log.Debug(ctx, "Start Workflow %s", wkEntity.Name)

	runEvent := sdk.V2WorkflowRunEvent{
		HookType:      runRequest.HookType,
		EventName:     runRequest.EventName,
		Ref:           runRequest.Ref,
		Sha:           runRequest.Sha,
		CommitMessage: runRequest.CommitMessage,
		SemverCurrent: runRequest.SemverCurrent,
		SemverNext:    runRequest.SemverNext,
		ChangeSets:    runRequest.ChangeSets,
		EntityUpdated: runRequest.EntityUpdated,
		Payload:       runRequest.Payload,
		Cron:          runRequest.Cron,
		CronTimezone:  runRequest.CronTimezone,
	}

	var msg string
	switch runEvent.HookType {
	case sdk.WorkflowHookTypeManual:
		msg = fmt.Sprintf("Workflow was manually triggered by user %s", u.Username)
	case sdk.WorkflowHookTypeRepository:
		msg = fmt.Sprintf("The workflow was triggered by the repository webhook event %s by user %s", runEvent.EventName, u.Username)
	case sdk.WorkflowHookTypeWorkflow:
		msg = fmt.Sprintf("Workflow was triggered by the workflow-update hook by user %s", u.Username)
	case sdk.WorkflowHookTypeWorkerModel:
		msg = fmt.Sprintf("Workflow was triggered by the model-update hook by user %s", u.Username)
	case sdk.WorkflowHookTypeScheduler:
		msg = fmt.Sprintf("Workflow was triggered by the scheduler %s %s", runEvent.Cron, runEvent.CronTimezone)
	default:
		return nil, sdk.WrapError(sdk.ErrNotImplemented, "event %s not implemented", runEvent.HookType)
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
		Status:       sdk.V2WorkflowRunStatusCrafting,
		RunAttempt:   0,
		Started:      time.Now(),
		LastModified: time.Now(),
		ToDelete:     false,
		WorkflowData: sdk.V2WorkflowRunData{Workflow: wk},
		UserID:       u.ID,
		AdminMFA:     runRequest.AdminMFA,
		Username:     u.Username,
		RunEvent:     runEvent,
		Contexts:     sdk.WorkflowRunContext{},
	}

	wrNumber, err := workflow_v2.WorkflowRunNextNumber(api.mustDB(), repo.ID, wk.Name)
	if err != nil {
		return nil, err
	}
	wr.RunNumber = wrNumber

	if proj.WorkflowRetention <= 0 {
		proj.WorkflowRetention = api.Config.WorkflowV2.WorkflowRunRetention
	}
	retention := time.Duration(proj.WorkflowRetention*24) * time.Hour
	if wk.Retention > 0 {
		retention = time.Duration(wk.Retention*24) * time.Hour
	}
	wr.RetentionDate = time.Now().Add(retention)

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
