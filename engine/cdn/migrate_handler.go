package cdn

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
)

func (s *Service) migrateArtifactInCDNHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["projectKey"]
		workflowName := vars["workflowName"]

		artifactIDS := vars["artifactID"]
		artifactID, err := strconv.ParseInt(artifactIDS, 10, 64)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "wrong artifact id")
		}

		var sign cdn.Signature
		if err := service.UnmarshalBody(r, &sign); err != nil {
			return err
		}

		nodeRun, err := s.Client.WorkflowNodeRun(projectKey, workflowName, sign.RunNumber, sign.NodeRunID)
		if err != nil {
			return err
		}

		if sign.WorkflowID != nodeRun.WorkflowID || sign.NodeRunID != nodeRun.ID || sign.RunID != nodeRun.WorkflowRunID || nodeRun.Number != sign.RunNumber {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "signature doesn't match request")
		}

		// Check if artifact exist
		found := false
		for _, a := range nodeRun.Artifacts {
			if a.ID == artifactID {
				found = true
				break
			}
		}
		if !found {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find artifact in the given run")
		}

		// Retrieve Artifact from CDS API
		url := fmt.Sprintf("/project/%s/workflows/%s/artifact/%d", projectKey, workflowName, artifactID)
		readcloser, _, code, err := s.Client.Stream(ctx, s.Client.HTTPNoTimeoutClient(), "GET", url, nil)
		if err != nil {
			return sdk.WithStack(err)
		}
		if code >= 400 {
			var bodyBtes []byte
			bodyBtes, errR := ioutil.ReadAll(readcloser)
			if errR != nil {
				return errR
			}
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to get artifact: %s", string(bodyBtes))
		}
		return s.storeFile(ctx, sign, readcloser, StoreFileOptions{DisableApiRunResult: true})
	}
}
