package pipeline

import (
	"bytes"
	"context"
	"encoding/base64"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// UpdateWorkflowAsCode update an as code workflow
func UpdatePipelineAsCode(ctx context.Context, store cache.Store, db gorp.SqlExecutor, proj *sdk.Project, p sdk.Pipeline, branch string, message string, app *sdk.Application, u sdk.Identifiable) (*sdk.Operation, error) {

	var wp exportentities.WorkflowPulled
	buffw := new(bytes.Buffer)
	if _, err := ExportPipeline(p, exportentities.FormatYAML, buffw); err != nil {
		return nil, sdk.WrapError(err, "unable to export pipeline")
	}
	wp.Pipelines = make([]exportentities.WorkflowPulledItem, 1)
	wpi := exportentities.WorkflowPulledItem{
		Name:  p.Name,
		Value: base64.StdEncoding.EncodeToString(buffw.Bytes()),
	}
	wp.Pipelines[0] = wpi
	return operation.PushOperation(ctx, db, store, proj, app, wp, branch, message, u)
}
