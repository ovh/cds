package cdn

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) markItemToDeleteHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !s.Cfg.EnableLogProcessing {
			return nil
		}
		var req sdk.CDNMarkDelete
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}

		if req.WorkflowID > 0 && req.RunID > 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "invalid data")
		}
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() //nolint

		if req.WorkflowID > 0 {
			if err := item.MarkToDeleteByWorkflowID(tx, req.WorkflowID); err != nil {
				return err
			}
		} else {
			if err := item.MarkToDeleteByRunIDs(tx, req.RunID); err != nil {
				return err
			}
		}
		return sdk.WrapError(tx.Commit(), "unable to commit transaction")
	}
}

func (s *Service) getItemHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if err := itemType.Validate(); err != nil {
			return err
		}
		apiRef := vars["apiRef"]

		// Try to load item and item units for given api ref
		it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRef, itemType)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, it, http.StatusOK)
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

func (s *Service) getItemLogsLinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if !itemType.IsLog() {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid item log type")
		}

		apiRef := vars["apiRef"]

		// offset can be lower than 0 if we want the n last lines
		offset := service.FormInt64(r, "offset")
		count := service.FormUInt(r, "count")
		sort := service.FormInt64(r, "sort") // < 0 for latest logs first, >= 0 for older logs first

		_, rc, _, err := s.getItemLogValue(ctx, itemType, apiRef, sdk.CDNReaderFormatJSON, offset, count, sort)
		if err != nil {
			return err
		}
		if rc == nil {
			return sdk.WrapError(sdk.ErrNotFound, "no storage found that contains given item %s", apiRef)
		}

		return service.Write(w, rc, http.StatusOK, "application/json")
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
