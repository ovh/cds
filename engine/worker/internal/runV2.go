package internal

import (
	"github.com/ovh/cds/sdk"
)

func (w *CurrentWorker) V2ProcessJob() (res sdk.V2WorkflowRunJobResult) {
	return sdk.V2WorkflowRunJobResult{
		Status: sdk.StatusSuccess,
	}
	// TODO implement run job
}
