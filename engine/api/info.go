package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/info"
	"github.com/ovh/cds/sdk"
)

func (api *API) addInfoHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inf sdk.Info
		if err := UnmarshalBody(r, &inf); err != nil {
			return sdk.WrapError(err, "addInfo> cannot unmarshal body")
		}

		if inf.Title == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "updateInfo> wrong title")
		}
		now := time.Now()
		inf.Created = now
		inf.Updated = now

		if err := info.InsertInfo(api.mustDB(), inf); err != nil {
			return sdk.WrapError(err, "addInfo> cannot add info")
		}

		return WriteJSON(w, inf, http.StatusOK)
	}
}

func (api *API) updateInfoHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		infoID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "updateInfo> Invalid id")
		}

		if _, err := info.LoadInfoByID(api.mustDB(), infoID); err != nil {
			return sdk.WrapError(err, "updateInfo> cannot load info by id")
		}

		// Unmarshal body
		var inf sdk.Info
		if err := UnmarshalBody(r, &inf); err != nil {
			return sdk.WrapError(err, "updateInfo> cannot unmarshal body")
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "updateInfo> unable to start transaction")
		}

		defer tx.Rollback()

		// update info in db
		if err := info.UpdateInfo(tx, inf); err != nil {
			return sdk.WrapError(err, "updateInfo> cannot update info")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateInfo> unable to commit transaction")
		}

		return WriteJSON(w, inf, http.StatusOK)
	}
}

func (api *API) deleteInfoHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		infoID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "deleteInfo> Invalid id")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteInfo> Cannot start transaction")
		}

		if err := info.DeleteInfo(tx, infoID); err != nil {
			return sdk.WrapError(err, "deleteInfo: cannot delete info")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteInfo> Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) getInfoHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "getInfo> Invalid id")
		}

		info, err := info.LoadInfoByID(api.mustDB(), id)
		if err != nil {
			return sdk.WrapError(err, "getInfo> cannot load infos")
		}

		return WriteJSON(w, info, http.StatusOK)
	}
}

func (api *API) getInfosHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getInfos> cannot parse form")
		}

		infos, err := info.LoadInfos(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getInfos> cannot load infos")
		}

		return WriteJSON(w, infos, http.StatusOK)
	}
}
