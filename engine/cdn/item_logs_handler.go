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
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) getItemLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		itemType := index.ItemType(vars["type"])
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
		itemType := index.ItemType(vars["type"])
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

		// Try to load item and item units for given api ref
		item, err := index.LoadItemByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), token.APIRefHash, itemType)
		if err != nil {
			return err
		}
		ius, err := storage.LoadAllItemUnitsByItemID(ctx, s.Mapper, s.mustDBWithCtx(ctx), item.ID)
		if err != nil {
			return err
		}
		if len(ius) == 0 {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		// Not complete item shoudl be loaded from buffer
		var itemUnitBuffer *storage.ItemUnit
		for i := range ius {
			if ius[i].UnitID == s.Units.Buffer.ID() {
				itemUnitBuffer = &ius[i]
				break
			}
		}
		if item.Status != index.StatusItemCompleted && itemUnitBuffer == nil {
			return sdk.WrapError(sdk.ErrNotFound, "missing item unit buffer for incoming log %s", token.APIRefHash)
		}

		// Always load from buffer if possible, if not in buffer try to load from another available storage unit
		var rc io.ReadCloser
		if itemUnitBuffer != nil {
			rc, err = s.Units.Buffer.NewReader(*itemUnitBuffer)
			if err != nil {
				return err
			}
			defer rc.Close()
		} else {
			for i := range ius {
				if ius[i].UnitID == s.Units.Buffer.ID() {
					continue
				}
				var su storage.StorageUnit
				for j := range s.Units.Storages {
					if ius[i].UnitID == s.Units.Storages[j].ID() {
						su = s.Units.Storages[j]
						break
					}
				}
				if su != nil {
					rc, err = s.Units.Buffer.NewReader(ius[i])
					if err != nil {
						return err
					}
					defer rc.Close()
					break
				}
			}
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
