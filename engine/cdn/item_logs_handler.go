package cdn

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/authentication"
	"github.com/ovh/cds/engine/cdn/index"
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
			if err := index.MarkItemToDeleteByWorkflowID(tx, req.WorkflowID); err != nil {
				return err
			}
		} else {
			if err := index.MarkItemToDeleteByRunIDs(tx, req.RunID); err != nil {
				return err
			}
		}
		return sdk.WrapError(tx.Commit(), "unable to commit transaction")
	}
}

func (s *Service) getItemLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if err := itemType.Validate(); err != nil {
			return err
		}
		apiRef := vars["apiRef"]

		// Try to load item and item units for given api ref
		item, err := index.LoadItemByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRef, itemType)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, item, http.StatusOK)
	}
}

func (s *Service) getItemLogsDownloadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if err := itemType.Validate(); err != nil {
			return err
		}
		apiRef := vars["apiRef"]
		tokenRaw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

		// Check Authorization header
		var token sdk.CDNAuthToken
		v := authentication.NewVerifier(s.ParsedAPIPublicKey)
		if err := v.VerifyJWS(tokenRaw, &token); err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}
		if token.APIRefHash != apiRef {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		rc, err := s.getItemLogValue(ctx, itemType, token.APIRefHash, 0, 0)
		if err != nil {
			return err
		}

		if rc == nil {
			return sdk.WrapError(sdk.ErrNotFound, "no storage found that contains given item %s", token.APIRefHash)
		}

		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.log\"", token.APIRefHash))

		if _, err := io.Copy(w, rc); err != nil {
			return sdk.WithStack(err)
		}

		return nil
	}
}
