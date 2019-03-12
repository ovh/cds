package action

import (
	"io"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export given action to writer.
func Export(a sdk.Action, f exportentities.Format, w io.Writer) error {
	ea := exportentities.NewAction(a)

	b, err := exportentities.Marshal(ea, f)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return sdk.WithStack(err)
}
