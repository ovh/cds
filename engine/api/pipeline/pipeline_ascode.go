package pipeline

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// UpdateWorkflowAsCode update an as code workflow
func UpdatePipelineAsCode(ctx context.Context, store cache.Store, db gorp.SqlExecutor, proj sdk.Project, p sdk.Pipeline, branch string, message string, app *sdk.Application, u sdk.Identifiable) (*sdk.Operation, error) {
	wpi := exportentities.NewPipelineV1(p)
	wp := exportentities.WorkflowComponents{
		Pipelines: []exportentities.PipelineV1{wpi},
	}
	return operation.PushOperation(ctx, db, store, proj, app, wp, branch, message, true, u)
}
