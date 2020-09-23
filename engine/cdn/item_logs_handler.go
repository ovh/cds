package cdn

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/authentication"
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

func (s *Service) getItemLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if err := itemType.IsLog(); err != nil {
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

func (s *Service) getItemLogsLinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if err := itemType.IsLog(); err != nil {
			return err
		}
		token, err := s.checkItemLogsAuth(r)
		if err != nil {
			return err
		}

		offset, err := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if err != nil {
			offset = 0 // offset can be lower than 0 if we want the n last lines
		}
		count, err := strconv.ParseInt(r.FormValue("count"), 10, 64)
		if err != nil || count < 0 {
			count = 100
		}

		rc, err := s.getItemLogValue(ctx, itemType, token.APIRefHash, sdk.CDNReaderFormatJSON, offset, uint(count))
		if err != nil {
			return err
		}
		if rc == nil {
			return sdk.WrapError(sdk.ErrNotFound, "no storage found that contains given item %s", token.APIRefHash)
		}

		return service.Write(w, rc, http.StatusOK, "application/json")
	}
}

func (s *Service) getItemLogsDownloadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := sdk.CDNItemType(vars["type"])
		if err := itemType.IsLog(); err != nil {
			return err
		}
		token, err := s.checkItemLogsAuth(r)
		if err != nil {
			return err
		}

		rc, err := s.getItemLogValue(ctx, itemType, token.APIRefHash, sdk.CDNReaderFormatText, 0, 0)
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

func (s *Service) checkItemLogsAuth(r *http.Request) (sdk.CDNAuthToken, error) {
	vars := mux.Vars(r)
	apiRef := vars["apiRef"]
	tokenRaw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	// Check Authorization header
	var token sdk.CDNAuthToken
	v := authentication.NewVerifier(s.ParsedAPIPublicKey)
	if err := v.VerifyJWS(tokenRaw, &token); err != nil {
		return token, sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
	}
	if token.APIRefHash != apiRef {
		return token, sdk.WithStack(sdk.ErrNotFound)
	}

	return token, nil
}
