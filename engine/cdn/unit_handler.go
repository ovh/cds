package cdn

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/cdn/storage"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) markUnitAsDeletehandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		unitID := vars["id"]

		_, err := storage.LoadUnitByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), unitID)
		if err != nil {
			return err
		}
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		if err := storage.MarkUnitToDelete(tx, unitID); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}
}
