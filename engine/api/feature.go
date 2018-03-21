package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/feature"
)

func (api *API) cleanFeatureHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		feature.Clean(api.Cache)
		return nil
	}
}
