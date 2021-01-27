package cdn

import (
	"context"
	"github.com/ovh/cds/engine/cdn/item"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getItemsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])

		switch itemType {
		case sdk.CDNTypeItemArtifact:
			return s.getArtifacts(ctx, r, w)
		}

		return sdk.WrapError(sdk.ErrInvalidData, "this type of items cannot be get")
	}
}

func (s *Service) getArtifacts(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	runID := r.FormValue("runid")
	if runID == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow run")
	}
	items, err := item.LoadByRunID(ctx, s.Mapper, s.mustDBWithCtx(ctx), sdk.CDNTypeItemArtifact, runID)
	if err != nil {
		return err
	}
	return service.WriteJSON(w, items, http.StatusOK)
}
