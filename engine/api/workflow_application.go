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
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) releaseApplicationWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		nodeRunID, errN := requestVarInt(r, "nodeRunID")
		if errN != nil {
			return errN
		}

		var req sdk.WorkflowNodeRunRelease
		if errU := service.UnmarshalBody(r, &req); errU != nil {
			return errU
		}

		proj, errprod := project.Load(ctx, api.mustDB(), key)
		if errprod != nil {
			return sdk.WrapError(errprod, "releaseApplicationWorkflowHandler")
		}
		loadOpts := workflow.LoadRunOptions{WithArtifacts: true}
		wNodeRun, errWNR := workflow.LoadNodeRun(api.mustDB(), key, name, nodeRunID, loadOpts)
		if errWNR != nil {
			return sdk.WrapError(errWNR, "releaseApplicationWorkflowHandler")
		}

		workflowRun, errWR := workflow.LoadRunByIDAndProjectKey(api.mustDB(), key, wNodeRun.WorkflowRunID, loadOpts)
		if errWR != nil {
			return sdk.WrapError(errWR, "releaseApplicationWorkflowHandler")
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
			return sdk.WithStack(sdk.ErrNoReposManager)
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
			f, err := api.SharedStorage.Fetch(ctx, &a)
			if err != nil {
				return sdk.WrapError(err, "Cannot fetch artifact")
			}

			if err := client.UploadReleaseFile(ctx, app.RepositoryFullname, fmt.Sprintf("%d", release.ID), release.UploadURL, a.Name, f); err != nil {
				return sdk.WrapError(err, "releaseApplicationWorkflowHandler")
			}
		}

		return nil
	}
}
