package pipeline

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export a pipeline
func Export(ctx context.Context, db gorp.SqlExecutor, key string, name string) (exportentities.PipelineV1, error) {
	p, err := LoadPipeline(ctx, db, key, name, true)
	if err != nil {
		return exportentities.PipelineV1{}, sdk.WrapError(err, "cannot load workflow %s", name)
	}
	return exportentities.NewPipelineV1(*p), nil
}
