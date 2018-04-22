package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/broadcast"
	"github.com/ovh/cds/sdk"
)

func (api *API) addBroadcastHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var bc sdk.Broadcast
		if err := UnmarshalBody(r, &bc); err != nil {
			return sdk.WrapError(err, "addBroadcast> cannot unmarshal body")
		}

		if bc.Title == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "updateBroadcast> wrong title")
		}
		now := time.Now()
		bc.Created = now
		bc.Updated = now

		if err := broadcast.InsertBroadcast(api.mustDB(), &bc); err != nil {
			return sdk.WrapError(err, "addBroadcast> cannot add broadcast")
		}

		return WriteJSON(w, bc, http.StatusOK)
	}
}

func (api *API) updateBroadcastHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcastID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "updateBroadcast> Invalid id")
		}

		if _, err := broadcast.LoadBroadcastByID(api.mustDB(), broadcastID); err != nil {
			return sdk.WrapError(err, "updateBroadcast> cannot load broadcast by id")
		}

		// Unmarshal body
		var bc sdk.Broadcast
		if err := UnmarshalBody(r, &bc); err != nil {
			return sdk.WrapError(err, "updateBroadcast> cannot unmarshal body")
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "updateBroadcast> unable to start transaction")
		}

		defer tx.Rollback()

		// update broadcast in db
		if err := broadcast.UpdateBroadcast(tx, bc); err != nil {
			return sdk.WrapError(err, "updateBroadcast> cannot update broadcast")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateBroadcast> unable to commit transaction")
		}

		return WriteJSON(w, bc, http.StatusOK)
	}
}

func (api *API) deleteBroadcastHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcastID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "deleteBroadcast> Invalid id")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteBroadcast> Cannot start transaction")
		}

		if err := broadcast.DeleteBroadcast(tx, broadcastID); err != nil {
			return sdk.WrapError(err, "deleteBroadcast: cannot delete broadcast")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteBroadcast> Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) getBroadcastHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "getBroadcast> Invalid id")
		}

		broadcast, err := broadcast.LoadBroadcastByID(api.mustDB(), id)
		if err != nil {
			return sdk.WrapError(err, "getBroadcast> cannot load broadcasts")
		}

		return WriteJSON(w, broadcast, http.StatusOK)
	}
}

func (api *API) getBroadcastsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getBroadcasts> cannot parse form")
		}

		broadcasts, err := broadcast.LoadBroadcasts(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getBroadcasts> cannot load broadcasts")
		}

		return WriteJSON(w, broadcasts, http.StatusOK)
	}
}
