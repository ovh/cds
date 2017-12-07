package pipeline

import (
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export a pipeline
func Export(db gorp.SqlExecutor, cache cache.Store, key string, name string, f exportentities.Format, withPermissions bool, u *sdk.User, w io.Writer) (int, error) {
	p, errload := LoadPipeline(db, key, name, true)
	if errload != nil {
		return 0, sdk.WrapError(errload, "workflow.Export> Cannot load workflow %s", name)
	}

	return ExportPipeline(*p, f, withPermissions, w)
}

// ExportPipeline a pipeline
func ExportPipeline(p sdk.Pipeline, f exportentities.Format, withPermissions bool, w io.Writer) (int, error) {
	e := exportentities.NewPipeline(p, withPermissions)

	// Marshal to the desired format
	b, err := exportentities.Marshal(e, f)
	if err != nil {
		return 0, sdk.WrapError(err, "workflow.Export>")
	}

	return w.Write(b)
}
