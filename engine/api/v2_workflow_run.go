package api

import (
	"context"
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

			for k := range runJob.StepsContext {
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

			u := getUserConsumer(ctx)

			wrNumber, err := workflow_v2.WorkflowRunNextNumber(api.mustDB(), repo.ID, wk.Name)
			if err != nil {
				return err
			}

			telemetry.MainSpan(ctx).AddAttributes(trace.StringAttribute(telemetry.TagWorkflowRunNumber, strconv.FormatInt(wrNumber, 10)))

			wr := sdk.V2WorkflowRun{
				ProjectKey:   proj.Key,
				VCSServerID:  vcsProject.ID,
				RepositoryID: repo.ID,
				WorkflowName: wk.Name,
				WorkflowRef:  workflowEntity.Branch,
				WorkflowSha:  workflowEntity.Commit,
				Status:       sdk.StatusCrafting,
				RunNumber:    wrNumber,
				RunAttempt:   0,
				Started:      time.Now(),
				LastModified: time.Now(),
				ToDelete:     false,
				WorkflowData: sdk.V2WorkflowRunData{Workflow: wk},
				UserID:       u.AuthConsumerUser.AuthentifiedUserID,
				Username:     u.AuthConsumerUser.AuthentifiedUser.Username,
				Event:        sdk.V2WorkflowRunEvent{},
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}

			wr.RunNumber = wrNumber
			if err := workflow_v2.InsertRun(ctx, tx, &wr); err != nil {
				return err
			}

			select {
			case api.workflowRunCraftChan <- wr.ID:
				log.Debug(ctx, "postWorkflowRunV2Handler: workflow run %s %d sent into chan", wr.WorkflowName, wr.RunNumber)
			default:
				// Default behaviour is made by a goroutine that call directly the database
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteJSON(w, wr, http.StatusCreated)
		}
}
