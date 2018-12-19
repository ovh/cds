package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var cacheOperationKey = cache.Key("repositories", "operation", "push")

// postWorkflowAsCodeHandler Make the workflow as code
// @title Make the workflow as code
func (api *API) postWorkflowAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]

		u := getUser(ctx)

		proj, errP := project.Load(api.mustDB(), api.Cache, key, u, project.LoadOptions.WithApplicationWithDeploymentStrategies, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "unable to load project")
		}
		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, workflowName, u, workflow.LoadOptions{
			DeepPipeline: true,
			WithLabels:   true,
			WithIcon:     true,
		})
		if errW != nil {
			return sdk.WrapError(errW, "unable to load workflow")
		}

		ope, err := workflow.MigrateAsCode(ctx, api.mustDB(), api.Cache, proj, wf, project.EncryptWithBuiltinKey, u)
		if err != nil {
			return sdk.WrapError(errW, "unable to migrate workflow as code")
		}

		api.Cache.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)

		go func(ope sdk.Operation, p *sdk.Project, wf *sdk.Workflow) {
			ctx := context.TODO()
			counter := 0
			for {
				counter++
				if err := workflow.GetRepositoryOperation(ctx, api.mustDB(), &ope); err != nil {
					log.Error("unable to get repository operation %s: %v", ope.UUID, err)
					continue
				}
				if ope.Status == sdk.OperationStatusError || ope.Status == sdk.OperationStatusDone {
					api.Cache.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)
				}

				if ope.Status == sdk.OperationStatusDone {
					app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]
					vcsServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
					if vcsServer == nil {
						log.Error("postWorkflowAsCodeHandler> No vcsServer found")
						return
					}
					client, errclient := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, vcsServer)
					if errclient != nil {
						log.Error("postWorkflowAsCodeHandler> unable to create repositories manager client: %v", err)
						return
					}
					request := sdk.VCSPullRequest{
						Title: ope.Setup.Push.Message,
						Head: sdk.VCSPushEvent{
							Branch: sdk.VCSBranch{
								DisplayID: ope.Setup.Push.FromBranch,
							},
							Repo: app.RepositoryFullname,
						},
						Base: sdk.VCSPushEvent{
							Branch: sdk.VCSBranch{
								DisplayID: ope.Setup.Push.ToBranch,
							},
							Repo: app.RepositoryFullname,
						},
					}
					pr, err := client.PullRequestCreate(ctx, app.RepositoryFullname, request)
					if err != nil {
						log.Error("postWorkflowAsCodeHandler> unable to create pull request")
						return
					}
					ope.Setup.Push.PRLink = pr.URL
					api.Cache.SetWithTTL(cache.Key(cacheOperationKey, ope.UUID), ope, 300)
				}

				if counter == 30 {
					ope.Status = sdk.OperationStatusError
					ope.Error = "Unable to enable workflow as code"
					break
				}
				time.Sleep(2 * time.Second)
			}
		}(ope, proj, wf)

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}
