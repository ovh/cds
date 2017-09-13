package main

import (
	"bytes"
	"net/http"
	"regexp"
	"sort"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func releaseApplicationWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]
	nodeRunID, errN := requestVarInt(r, "nodeRunID")
	if errN != nil {
		return errN
	}

	number, errNRI := requestVarInt(r, "number")
	if errNRI != nil {
		return errNRI
	}

	var req sdk.WorkflowNodeRunRelease
	if errU := UnmarshalBody(r, &req); errU != nil {
		return errU
	}

	wNodeRun, errWNR := workflow.LoadNodeRun(db, key, name, number, nodeRunID)
	if errWNR != nil {
		return sdk.WrapError(errWNR, "releaseApplicationWorkflowHandler")
	}

	workflowRun, errWR := workflow.LoadRunByIDAndProjectKey(db, key, wNodeRun.WorkflowRunID)
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

	workflowNode := workflowRun.Workflow.GetNode(wNodeRun.WorkflowNodeID)
	if workflowNode == nil {
		return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "releaseApplicationWorkflowHandler")
	}

	if workflowNode.Context == nil || workflowNode.Context.Application == nil {
		return sdk.WrapError(sdk.ErrApplicationNotFound, "releaseApplicationWorkflowHandler")
	}

	if workflowNode.Context.Application.RepositoriesManager == nil {
		return sdk.WrapError(sdk.ErrNoReposManager, "releaseApplicationWorkflowHandler")
	}

	client, err := repositoriesmanager.AuthorizedClient(db, key, workflowNode.Context.Application.RepositoriesManager.Name)
	if err != nil {
		return sdk.WrapError(err, "releaseApplicationWorkflowHandler> Cannot get client got %s %s", key, workflowNode.Context.Application.RepositoriesManager.Name)
	}

	release, errRelease := client.Release(workflowNode.Context.Application.RepositoryFullname, req.TagName, req.ReleaseTitle, req.ReleaseContent)
	if errRelease != nil {
		return sdk.WrapError(errRelease, "releaseApplicationWorkflowHandler")
	}

	// Get artifacts to upload
	var artifactToUpload []sdk.WorkflowNodeRunArtifact
	for _, a := range workflowArtifacts {
		for _, aToUp := range req.Artifacts {
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

	for _, a := range artifactToUpload {
		b := &bytes.Buffer{}
		if err := artifact.StreamFile(b, &a); err != nil {
			return sdk.WrapError(err, "Cannot get artifact")
		}
		if err := client.UploadReleaseFile(workflowNode.Context.Application.RepositoryFullname, release, a, b); err != nil {
			return sdk.WrapError(err, "releaseApplicationWorkflowHandler")
		}
	}

	return nil
}
