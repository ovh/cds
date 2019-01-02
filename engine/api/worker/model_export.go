package worker

import (
	"io"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// Export convert sdk.Model to an exportentities.WorkerModel, format and write into a io.Writer
func Export(wm sdk.Model, f exportentities.Format, w io.Writer) (int, error) {
	eWm := exportentities.NewWorkerModel(wm)

	// Marshal to the desired format
	b, err := exportentities.Marshal(eWm, f)
	if err != nil {
		return 0, sdk.WithStack(err)
	}

	return w.Write(b)
}
