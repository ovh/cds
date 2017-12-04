package api

import (
	"context"
	"net/http"
)

func (api *API) postWorkflowImportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, r, nil, http.StatusOK)
	}
}
