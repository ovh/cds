package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/common/expfmt"

	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getMetricsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		mfs, err := metrics.GetGatherer().Gather()
		if err != nil {
			return sdk.WrapError(err, "An error has occurred during metrics gathering")
		}
		contentType := expfmt.Negotiate(r.Header)
		writer := &bytes.Buffer{}
		enc := expfmt.NewEncoder(writer, contentType)
		for _, mf := range mfs {
			if err := enc.Encode(mf); err != nil {
				return sdk.WrapError(err, "An error has occurred during metrics encoding")
			}
		}
		header := w.Header()
		header.Set("Content-Type", string(contentType))
		header.Set("Content-Length", fmt.Sprint(writer.Len()))

		contentTypeRequested := r.Header.Get("Content-Type")
		switch contentTypeRequested {
		case "application/json":
			return service.WriteJSON(w, mfs, http.StatusOK)
		default:
			w.Write(writer.Bytes()) // nolint
		}
		return nil
	}
}
