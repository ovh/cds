package action

import (
	"io"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export an action
func Export(db gorp.SqlExecutor, name string, f exportentities.Format, u *sdk.User, w io.Writer) (int, error) {
	a, err := LoadPublicAction(db, name)
	if err != nil {
		return 0, sdk.WrapError(sdk.ErrNotFound, "Export> Cannot load action")
	}

	return ExportAction(db, *a, f, w)
}

// ExportAction export
func ExportAction(db gorp.SqlExecutor, a sdk.Action, f exportentities.Format, w io.Writer) (int, error) {
	ea := exportentities.NewAction(a)
	b, err := exportentities.Marshal(ea, f)
	if err != nil {
		return 0, sdk.WrapError(err, "application.Export>")
	}

	return w.Write(b)
}
