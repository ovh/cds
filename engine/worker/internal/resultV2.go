package internal

import (
	"context"

	"github.com/ovh/cds/sdk"
	"github.com/pkg/errors"
)

type ResultAddRequest struct {
	Type       string
	Identifier string
}

func (w *CurrentWorker) ResultAdd(ctx context.Context, req ResultAddRequest) error {
	// Check if there is a artifactory integration on the run
	var withArtifactory bool

	var runResult sdk.V2WorkflowRunResult

	if !withArtifactory {
		switch req.Type {
		case "file", "generic", "artifact":
			runResult.Type = sdk.V2WorkflowRunResultTypeGeneric

		case "coverage":

		case "docker":

		default:
			return errors.Errorf("result type %q is not support", req.Type)
		}
	}
	return nil
}
