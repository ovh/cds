package api

import (
	"context"
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
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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

			runJobs, err := workflow_v2.LoadRunJobsByRunID(ctx, api.mustDB(), wr.ID)
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
			jobName := vars["jobName"]

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

			runJob, err := workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobName)
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
			jobName := vars["jobName"]

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

			runJob, err := workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobName)
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
			jobName := vars["jobName"]

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

			runJob, err := workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobName)
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
				return sdk.WithStack(err)
			}
			workflowName := vars["workflow"]
			runNumberS := vars["runNumber"]
			runNumber, err := strconv.ParseInt(runNumberS, 10, 64)
			if err != nil {
				return err
			}
			jobName := vars["jobName"]

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

			runJob, err := workflow_v2.LoadRunJobByName(ctx, api.mustDB(), wr.ID, jobName)
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
					APIRef:    apiRefHash,
					StepOrder: r.StepOrder,
					StepName:  r.StepName,
				})
			}

			httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
			if err != nil {
				return err
			}

			return service.WriteJSON(w, sdk.CDNLogLinks{
				CDNURL:   httpURL,
				ItemType: sdk.CDNTypeItemJobStepLog,
				Data:     datas,
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

			wr, err := workflow_v2.LoadRunByRunNumber(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName, runNumber)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, wr, http.StatusOK)
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

			runs, err := workflow_v2.LoadRuns(ctx, api.mustDB(), proj.Key, vcsProject.ID, repo.ID, workflowName)
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

			return sdk.WithStack(tx.Commit())
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
			branch := QueryString(req, "branch")

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

			if branch == "" {
				tx, err := api.mustDB().Begin()
				if err != nil {
					return err
				}
				vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, proj.Key, vcsProject.Name)
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				defaultBranch, err := vcsClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
					return err
				}
				branch = defaultBranch.DisplayID
			}

			workflowEntity, err := entity.LoadByBranchTypeName(ctx, api.mustDB(), repo.ID, branch, sdk.EntityTypeWorkflow, workflowName)
			if err != nil {
				return err
			}

			var wk sdk.V2Workflow
			if err := yaml.Unmarshal([]byte(workflowEntity.Data), &wk); err != nil {
				return err
			}

			var runRequest sdk.V2WorkflowRunHookRequest
			if err := service.UnmarshalRequest(ctx, req, &runRequest); err != nil {
				return err
			}

			u, err := user.LoadByID(ctx, api.mustDB(), runRequest.UserID)
			if err != nil {
				return err
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
					Payload:   runRequest.Payload,
					EventName: runRequest.EventName,
					Ref:       runRequest.Ref,
					Sha:       runRequest.Sha,
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
			branch := QueryString(req, "branch")

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

			if branch == "" {
				tx, err := api.mustDB().Begin()
				if err != nil {
					return err
				}
				vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, proj.Key, vcsProject.Name)
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				defaultBranch, err := vcsClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
					return err
				}
				branch = defaultBranch.DisplayID
			}

			workflowEntity, err := entity.LoadByBranchTypeName(ctx, api.mustDB(), repo.ID, branch, sdk.EntityTypeWorkflow, workflowName)
			if err != nil {
				return err
			}

			var wk sdk.V2Workflow
			if err := yaml.Unmarshal([]byte(workflowEntity.Data), &wk); err != nil {
				return err
			}

			runEvent := sdk.V2WorkflowRunEvent{
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
		RepositoryID: repo.ID,
		WorkflowName: wk.Name,
		WorkflowRef:  wkEntity.Branch,
		WorkflowSha:  wkEntity.Commit,
		Status:       sdk.StatusCrafting,
		RunAttempt:   0,
		Started:      time.Now(),
		LastModified: time.Now(),
		ToDelete:     false,
		WorkflowData: sdk.V2WorkflowRunData{Workflow: wk},
		UserID:       u.ID,
		Username:     u.Username,
		Event:        runEvent,
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

	select {
	case api.workflowRunCraftChan <- wr.ID:
		log.Debug(ctx, "postWorkflowRunV2Handler: workflow run %s %d sent into chan", wr.WorkflowName, wr.RunNumber)
	default:
		// Default behaviour is made by a goroutine that call directly the database
	}

	if err := tx.Commit(); err != nil {
		return nil, sdk.WithStack(err)
	}
	return &wr, nil
}
