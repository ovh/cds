package service

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"
)

// VersionHandler returns version of current uservice
func VersionHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, sdk.VersionCurrent(), http.StatusOK)
	}
}
