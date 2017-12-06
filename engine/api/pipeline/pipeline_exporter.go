package pipeline

import (
	"io"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export a pipeline
func Export(db gorp.SqlExecutor, cache cache.Store, key string, name string, f exportentities.Format, withPermissions bool, u *sdk.User, w io.Writer) error {
	p, errload := LoadPipeline(db, key, name, true)
	if errload != nil {
		return sdk.WrapError(errload, "workflow.Export> Cannot load workflow %s", name)
	}

	e := exportentities.NewPipeline(p, withPermissions)

	// Marshal to the desired format
	b, err := exportentities.Marshal(e, f)
	if err != nil {
		return sdk.WrapError(err, "workflow.Export>")
	}

	_, errw := w.Write(b)
	return errw
}
