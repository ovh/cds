package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/broadcast"
	"github.com/ovh/cds/engine/api/project"
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

		if bc.ProjectKey != "" {
			proj, errProj := project.Load(api.mustDB(), api.Cache, bc.ProjectKey, getUser(ctx))
			if errProj != nil {
				return sdk.WrapError(sdk.ErrNoProject, "addBroadcast> Cannot load %s", bc.ProjectKey)
			}
			bc.ProjectID = &proj.ID
		}

		if err := broadcast.Insert(api.mustDB(), &bc); err != nil {
			return sdk.WrapError(err, "addBroadcast> cannot add broadcast")
		}

		return WriteJSON(w, bc, http.StatusCreated)
	}
}

func (api *API) updateBroadcastHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcastID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "updateBroadcast> Invalid id")
		}

		u := getUser(ctx)
		if _, err := broadcast.LoadByID(api.mustDB(), broadcastID, u); err != nil {
			return sdk.WrapError(err, "updateBroadcast> cannot load broadcast by id")
		}

		// Unmarshal body
		var bc sdk.Broadcast
		if err := UnmarshalBody(r, &bc); err != nil {
			return sdk.WrapError(err, "updateBroadcast> cannot unmarshal body")
		}

		if bc.ProjectKey != "" {
			proj, errProj := project.Load(api.mustDB(), api.Cache, bc.ProjectKey, u)
			if errProj != nil {
				return sdk.WrapError(sdk.ErrNoProject, "updateBroadcast> Cannot load %s", bc.ProjectKey)
			}
			bc.ProjectID = &proj.ID
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "updateBroadcast> unable to start transaction")
		}

		defer tx.Rollback()

		if bc.ID <= 0 || broadcastID != bc.ID {
			return sdk.WrapError(sdk.ErrWrongRequest, "requestVarInt> %s is not valid. id in path:%d", bc.ID, broadcastID)
		}

		// update broadcast in db
		if err := broadcast.Update(tx, &bc); err != nil {
			return sdk.WrapError(err, "updateBroadcast> cannot update broadcast")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateBroadcast> unable to commit transaction")
		}

		return WriteJSON(w, bc, http.StatusOK)
	}
}

func (api *API) postMarkAsReadBroadcastHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcastID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "updateBroadcast> Invalid id")
		}

		u := getUser(ctx)
		br, errL := broadcast.LoadByID(api.mustDB(), broadcastID, u)
		if errL != nil {
			return sdk.WrapError(errL, "postMarkAsReadBroadcastHandler> cannot load broadcast by id")
		}

		if !br.Read {
			if err := broadcast.MarkAsRead(api.mustDB(), broadcastID, u.ID); err != nil {
				return sdk.WrapError(err, "postMarkAsReadBroadcastHandler> cannot mark as read broadcast id %d and user id %d", broadcastID, u.ID)
			}
		}

		return nil
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

		if err := broadcast.Delete(tx, broadcastID); err != nil {
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

		broadcast, err := broadcast.LoadByID(api.mustDB(), id, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getBroadcast> cannot load broadcasts")
		}

		return WriteJSON(w, broadcast, http.StatusOK)
	}
}

func (api *API) getBroadcastsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcasts, err := broadcast.LoadAll(api.mustDB(), getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getBroadcasts> cannot load broadcasts")
		}

		return WriteJSON(w, broadcasts, http.StatusOK)
	}
}
