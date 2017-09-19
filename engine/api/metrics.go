package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/prometheus/common/expfmt"

	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/sdk"
)

func (api *API) getMetricsHandler() Handler {
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
				return sdk.WrapError(err, "metrics> An error has occurred during metrics encoding")
			}
		}
		header := w.Header()
		header.Set("Content-Type", string(contentType))
		header.Set("Content-Length", fmt.Sprint(writer.Len()))
		w.Write(writer.Bytes())
		return nil
	}
}
