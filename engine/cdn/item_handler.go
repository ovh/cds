package cdn

import (
	"bytes"
	"context"
	"encoding/hex"
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

func (s *Service) getItemDownloadInUnitHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		apiRef := vars["apiRef"]
		unitName := vars["unit"]

		return s.downloadItemFromUnit(ctx, itemType, apiRef, unitName, w)
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
			if data.Consumer.AuthConsumerUser.AuthentifiedUser.Ring != sdk.UserRingAdmin {
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

		bufferUnit := s.Units.GetBuffer(itemType)

		iu, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), bufferUnit.ID(), it.ID, opts...)
		if err != nil {
			if !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
		}
		if iu != nil {
			res.Location[bufferUnit.Name()] = *iu
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

func (s *Service) getItemCheckSyncHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		apiRef := vars["apiRef"]

		it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRef, itemType, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			return err
		}

		itemsUnits, err := storage.LoadAllItemUnitsByItemIDs(ctx, s.Mapper, s.mustDBWithCtx(ctx), it.ID, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			return err
		}

		if len(itemsUnits) == 0 {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		itemsUnits = s.Units.FilterItemUnitReaderByType(itemsUnits)

		var contents = map[string]*bytes.Buffer{}
		for _, iu := range itemsUnits {
			src, err := s.Units.NewSource(ctx, iu)
			if err != nil {
				return err
			}
			reader, err := src.NewReader(ctx)
			if err != nil {
				return err
			}
			buf := new(bytes.Buffer)
			if err := src.Read(reader, buf); err != nil {
				return err
			}
			contents[src.Name()] = buf
		}

		var lastContent string
		for st, buffer := range contents {
			if lastContent == "" {
				lastContent = hex.EncodeToString(buffer.Bytes())
				continue
			}
			if lastContent != hex.EncodeToString(buffer.Bytes()) {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "content of %s on %s doesn't match", apiRef, st)
			}
		}

		var result = struct {
			ID         string
			APIRefHash string
			SHA512     string
			Content    string
		}{
			ID:         it.ID,
			APIRefHash: it.APIRefHash,
			SHA512:     it.Hash,
			Content:    lastContent,
		}

		return service.WriteJSON(w, result, http.StatusOK)
	}
}

func (s *Service) getItemsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])

		switch itemType {
		case sdk.CDNTypeItemRunResult:
			return s.getArtifacts(ctx, r, w)
		case sdk.CDNTypeItemWorkerCache, sdk.CDNTypeItemWorkerCacheV2:
			return s.getWorkerCache(ctx, r, w, string(itemType))
		}

		return sdk.WrapError(sdk.ErrInvalidData, "this type of items cannot be get")
	}
}

func (s *Service) getArtifacts(ctx context.Context, r *http.Request, w http.ResponseWriter) error {
	runID := r.FormValue("runid")
	if runID == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid workflow run")
	}
	items, err := item.LoadRunResultByRunID(ctx, s.Mapper, s.mustDBWithCtx(ctx), runID)
	if err != nil {
		return err
	}
	return service.WriteJSON(w, items, http.StatusOK)
}

func (s *Service) getWorkerCache(ctx context.Context, r *http.Request, w http.ResponseWriter, cacheType string) error {
	projectKey := r.FormValue("projectkey")
	cachetag := r.FormValue("cachetag")

	if projectKey == "" || cachetag == "" {
		return sdk.WrapError(sdk.ErrWrongRequest, "invalid data to get worker cache")
	}
	item, err := item.LoadWorkerCacheItemByProjectAndCacheTag(ctx, s.Mapper, s.mustDBWithCtx(ctx), cacheType, projectKey, cachetag)
	if err != nil {
		return err
	}
	return service.WriteJSON(w, []sdk.CDNItem{*item}, http.StatusOK)
}
