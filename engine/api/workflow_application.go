package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"

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
		loadOpts := workflow.LoadRunOptions{WithArtifacts: true}
		wNodeRun, err := workflow.LoadNodeRun(api.mustDB(), key, name, nodeRunID, loadOpts)
		if err != nil {
			return err
		}

		workflowRun, err := workflow.LoadRunByIDAndProjectKey(ctx, api.mustDB(), key, wNodeRun.WorkflowRunID, loadOpts)
		if err != nil {
			return err
		}

		workflowArtifacts := []sdk.WorkflowNodeRunArtifact{}
		for _, runs := range workflowRun.WorkflowNodeRuns {
			if len(runs) == 0 {
				continue
			}
			sort.Slice(runs, func(i, j int) bool {
				return runs[i].SubNumber > runs[j].SubNumber
			})
			workflowArtifacts = append(workflowArtifacts, runs[0].Artifacts...)
		}

		node := workflowRun.Workflow.WorkflowData.NodeByID(wNodeRun.WorkflowNodeID)
		if node.Context == nil || node.Context.ApplicationID == 0 {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		app := workflowRun.Workflow.Applications[node.Context.ApplicationID]

		if app.VCSServer == "" {
			return sdk.NewErrorFrom(sdk.ErrNoReposManager, "app.VCSServer is empty")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		rm, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, api.mustDB(), key, app.VCSServer)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "cannot get client %s %s", key, app.VCSServer))
		}

		client, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, proj.Key, rm)
		if err != nil {
			return sdk.WrapError(err, "cannot get client got %s %s", key, app.VCSServer)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		release, errRelease := client.Release(ctx, app.RepositoryFullname, req.TagName, req.ReleaseTitle, req.ReleaseContent)
		if errRelease != nil {
			return sdk.WithStack(errRelease)
		}

		// Get artifacts to upload
		var artifactToUpload []sdk.WorkflowNodeRunArtifact
		for _, a := range workflowArtifacts {
			for _, aToUp := range req.Artifacts {
				if len(aToUp) > 0 {
					ok, errRX := regexp.Match(aToUp, []byte(a.Name))
					if errRX != nil {
						return sdk.WrapError(errRX, "releaseApplicationWorkflowHandler> %s is not a valid regular expression", aToUp)
					}
					if ok {
						artifactToUpload = append(artifactToUpload, a)
						break
					}
				}
			}
		}

		for _, a := range artifactToUpload {
			// Do manual retry because if http call failed, reader is closed
			attempt := 0
			var lastErr error
			for {
				attempt++
				f, err := api.SharedStorage.Fetch(ctx, &a)
				if err != nil {
					return sdk.WrapError(err, "Cannot fetch artifact")
				}

				if err := client.UploadReleaseFile(ctx, app.RepositoryFullname, fmt.Sprintf("%d", release.ID), release.UploadURL, a.Name, f, int(a.Size)); err != nil {
					lastErr = err
					if attempt >= 5 {
						break
					}
					continue
				}
				break
			}
			if lastErr != nil {
				return err
			}

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
					ok, err := regexp.Match(aToUp, []byte(artiData.Name))
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
				return err
			}
		}
		return nil
	}
}
