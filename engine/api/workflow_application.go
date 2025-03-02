package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) releaseApplicationWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isWorker := isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		nodeRunID, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return err
		}

		var req sdk.WorkflowNodeRunRelease
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}

		proj, err := project.Load(ctx, api.mustDB(), key)
		if err != nil {
			return err
		}
		wNodeRun, err := workflow.LoadNodeRun(api.mustDB(), key, name, nodeRunID, workflow.LoadRunOptions{})
		if err != nil {
			return err
		}

		workflowRun, err := workflow.LoadRunByIDAndProjectKey(ctx, api.mustDB(), key, wNodeRun.WorkflowRunID, workflow.LoadRunOptions{})
		if err != nil {
			return err
		}

		node := workflowRun.Workflow.WorkflowData.NodeByID(wNodeRun.WorkflowNodeID)
		if node.Context == nil || node.Context.ApplicationID == 0 {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		app := workflowRun.Workflow.Applications[node.Context.ApplicationID]

		if app.VCSServer == "" {
			return sdk.NewErrorFrom(sdk.ErrNoReposManager, "app.VCSServer is empty")
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, proj.Key, app.VCSServer)
		if err != nil {
			return sdk.WrapError(err, "cannot get client got %s %s", key, app.VCSServer)
		}

		release, errRelease := client.Release(ctx, app.RepositoryFullname, req.TagName, req.ReleaseTitle, req.ReleaseContent)
		if errRelease != nil {
			return sdk.WithStack(errRelease)
		}

		results, err := workflow.LoadRunResultsByRunIDAndType(ctx, api.mustDB(), workflowRun.ID, sdk.WorkflowRunResultTypeArtifact)
		if err != nil {
			return err
		}
		var resultToUpload []sdk.WorkflowRunResultArtifact
		for _, r := range results {
			artiData, err := r.GetArtifact()
			if err != nil {
				return err
			}
			for _, aToUp := range req.Artifacts {
				if len(aToUp) > 0 {
					ok, err := regexp.MatchString(aToUp, artiData.Name)
					if err != nil {
						return sdk.WrapError(err, "releaseApplicationWorkflowHandler> %s is not a valid regular expression", aToUp)
					}
					if ok {
						resultToUpload = append(resultToUpload, artiData)
						break
					}
				}
			}
		}

		if len(resultToUpload) == 0 {
			return nil
		}
		cdnHTTP, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}
		for _, r := range resultToUpload {
			// Do manual retry because if http call failed, reader is closed
			attempt := 0
			var lastErr error
			for {
				attempt++
				reader, err := api.Client.CDNItemStream(ctx, cdnHTTP, r.CDNRefHash, sdk.CDNTypeItemRunResult)
				if err != nil {
					return err
				}
				if err := client.UploadReleaseFile(ctx, app.RepositoryFullname, fmt.Sprintf("%d", release.ID), release.UploadURL, r.Name, reader, int(r.Size)); err != nil {
					lastErr = err
					if attempt >= 5 {
						break
					}
					continue
				}
				break
			}
			if lastErr != nil {
				return lastErr
			}
		}
		return nil
	}
}
