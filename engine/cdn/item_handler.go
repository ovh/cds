package cdn

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) bulkDeleteItemsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !s.Cfg.EnableLogProcessing {
			return nil
		}
		var req sdk.CDNMarkDelete
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}

		if req.RunID <= 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "missing runID")
		}
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() //nolint

		if err := item.MarkToDeleteByRunIDs(tx, req.RunID); err != nil {
			return err
		}

		return sdk.WrapError(tx.Commit(), "unable to commit transaction")
	}
}

func (s *Service) getItemDownloadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		apiRef := vars["apiRef"]

		var opts downloadOpts
		// User can give a refresh delay in seconds, Refresh header value will be set if item is not complete
		opts.Log.Refresh = service.FormInt64(r, "refresh")
		opts.Log.Sort = service.FormInt64(r, "sort") // < 0 for latest logs first, >= 0 for older logs first

		return s.downloadItem(ctx, itemType, apiRef, w, opts)
	}
}

func (s *Service) getSizeByProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["projectKey"]

		// get size used by a project key
		size, err := item.ComputeSizeByProjectKey(s.mustDBWithCtx(ctx), projectKey)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, size, http.StatusOK)
	}
}

func (s *Service) getItemHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		apiRef := vars["apiRef"]

		withDecryption := service.FormBool(r, "withDecryption")
		// Only admin can use the parameter 'withDecryption'
		var opts []gorpmapper.GetOptionFunc
		if withDecryption {
			sessionID := s.sessionID(ctx)
			data, err := s.Client.AuthSessionGet(sessionID)
			if err != nil {
				return err
			}

			if data.Consumer.AuthentifiedUser.Ring != sdk.UserRingAdmin {
				return sdk.WithStack(sdk.ErrUnauthorized)
			}

			opts = append(opts, gorpmapper.GetOptions.WithDecryption)
		}

		it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRef, itemType, opts...)
		if err != nil {
			return err
		}

		var res sdk.CDNItemResume
		res.CDNItem = *it
		res.Location = make(map[string]sdk.CDNItemUnit)

		iu, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Buffer.ID(), it.ID, opts...)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
		}

		if iu != nil {
			res.Location[s.Units.Buffer.Name()] = *iu
		}

		for _, strg := range s.Units.Storages {
			iu, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), strg.ID(), it.ID, opts...)
			if err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					continue
				}
				return err
			}
			res.Location[strg.Name()] = *iu
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func (s *Service) deleteItemHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		apiRef := vars["apiRef"]

		it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRef, itemType)
		if err != nil {
			return err
		}

		it.ToDelete = true

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() //nolint

		if err := item.Update(ctx, s.Mapper, tx, it); err != nil {
			return err
		}

		return sdk.WithStack(tx.Commit())
	}
}
