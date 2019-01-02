package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getWorkerModelExportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		workerModelID, errr := requestVarInt(r, "permModelID")
		if errr != nil {
			return sdk.WrapError(errr, "Invalid permModelID")
		}

		format := FormString(r, "format")
		if format == "" {
			format = "yaml"
		}

		wm, err := worker.LoadWorkerModelByID(api.mustDB(), workerModelID)
		if err != nil {
			return sdk.WrapError(err, "cannot load worker model id %d", workerModelID)
		}

		// Export
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.WrapError(err, "Format invalid")
		}

		if _, err := worker.Export(*wm, f, w); err != nil {
			return err
		}

		w.Header().Add("Content-Type", exportentities.GetContentType(f))
		w.WriteHeader(http.StatusOK)
		return nil
	}
}
