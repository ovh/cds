package pipeline

import (
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export a pipeline
func Export(db gorp.SqlExecutor, cache cache.Store, key string, name string, f exportentities.Format, w io.Writer) (int, error) {
	p, err := LoadPipeline(db, key, name, true)
	if err != nil {
		return 0, sdk.WrapError(err, "Cannot load workflow %s", name)
	}

	return ExportPipeline(*p, f, w)
}

// ExportPipeline a pipeline
func ExportPipeline(p sdk.Pipeline, f exportentities.Format, w io.Writer) (int, error) {
	e := exportentities.NewPipelineV1(p)

	// Marshal to the desired format
	b, err := exportentities.Marshal(e, f)
	if err != nil {
		return 0, err
	}

	return w.Write(b)
}
