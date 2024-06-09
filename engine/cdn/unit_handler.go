package cdn

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getUnitsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		units, err := storage.LoadAllUnits(ctx, s.Mapper, s.mustDBWithCtx(ctx))
		if err != nil {
			return err
		}

		response := make([]sdk.CDNUnitHandlerRequest, 0, len(units))
		for _, u := range units {
			nb, err := storage.CountItemsForUnit(s.mustDBWithCtx(ctx), u.ID)
			if err != nil {
				return err
			}
			response = append(response, sdk.CDNUnitHandlerRequest{
				ID:      u.ID,
				Name:    u.Name,
				NbItems: nb,
			})
		}
		return service.WriteJSON(w, response, http.StatusOK)
	}
}
func (s *Service) deleteUnitHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		unitID := vars["id"]

		unit, err := storage.LoadUnitByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), unitID)
		if err != nil {
			return err
		}
		nbItem, err := storage.CountItemsForUnit(s.mustDBWithCtx(ctx), unit.ID)
		if err != nil {
			return err
		}

		if nbItem > 0 {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "unable to delete unit %s because there are still item units", unit.Name)
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		if err := storage.DeleteUnit(s.Mapper, tx, unit); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}
}

func (s *Service) markItemUnitAsDeleteHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		unitID := vars["id"]

		unit, err := storage.LoadUnitByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), unitID)
		if err != nil {
			return err
		}

		go func() {
			ctx = context.Background()
			offset := int64(0)
			limit := int64(1000)
			for {
				ids, err := storage.LoadAllItemUnitsIDsByUnitID(s.mustDBWithCtx(ctx), unit.ID, offset, limit)
				if err != nil {
					log.Error(ctx, "unable to load item unit: %v", err)
					return
				}
				tx, err := s.mustDBWithCtx(ctx).Begin()
				if err != nil {
					log.Error(ctx, "unable to start transaction: %v", err)
					return
				}
				if _, err := storage.MarkItemUnitToDelete(tx, ids); err != nil {
					_ = tx.Rollback()
					log.Error(ctx, "unable to mark item unit to delete: %v", err)
					return
				}
				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
					log.Error(ctx, "unable to commit transaction: %v", err)
					return
				}
				if int64(len(ids)) < limit {
					log.Info(ctx, "All items unit have been marked as delete for unit %d: %v", unit.ID, err)
					return
				}
			}
		}()
		return nil
	}
}

func (s *Service) postAdminResyncBackendWithDatabaseHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		unitID := vars["id"]
		itemType := vars["type"]

		it := sdk.CDNItemType(itemType)
		if err := it.Validate(); err != nil {
			return err
		}

		dryRunString := r.FormValue("dryRun")
		dryRun := dryRunString != "false"

		for _, u := range s.Units.Buffers {
			if u.ID() == unitID {
				chosenUnit := u
				s.GoRoutines.Exec(context.Background(), "ResyncWithDB-"+unitID, func(ctx context.Context) {
					chosenUnit.ResyncWithDatabase(ctx, s.mustDBWithCtx(ctx), it, dryRun)
				})
			}
		}
		for _, u := range s.Units.Storages {
			if u.ID() == unitID {
				s.GoRoutines.Exec(context.Background(), "ResyncWithDB-"+unitID, func(ctx context.Context) {
					u.ResyncWithDatabase(ctx, s.mustDBWithCtx(ctx), it, dryRun)
				})
			}
		}
		return nil
	}
}
