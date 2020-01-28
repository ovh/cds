package api

import (
	"context"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/broadcast"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) addBroadcastHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var bc sdk.Broadcast
		if err := service.UnmarshalBody(r, &bc); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal body")
		}

		if bc.Title == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Wrong title")
		}
		now := time.Now()
		bc.Created = now
		bc.Updated = now

		if bc.ProjectKey != "" {
			proj, errProj := project.Load(api.mustDB(), api.Cache, bc.ProjectKey)
			if errProj != nil {
				return sdk.WrapError(sdk.ErrNoProject, "Cannot load %s", bc.ProjectKey)
			}
			bc.ProjectID = &proj.ID
		}

		if err := broadcast.Insert(api.mustDB(), &bc); err != nil {
			return sdk.WrapError(err, "Cannot add broadcast")
		}

		event.PublishBroadcastAdd(ctx, bc, getAPIConsumer(ctx))
		return service.WriteJSON(w, bc, http.StatusCreated)
	}
}

func (api *API) updateBroadcastHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcastID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "Invalid id")
		}

		consumer := getAPIConsumer(ctx)
		oldBC, err := broadcast.LoadByID(api.mustDB(), broadcastID, consumer.AuthentifiedUser)
		if err != nil {
			return sdk.WrapError(err, "Cannot load broadcast by id")
		}

		// Unmarshal body
		var bc sdk.Broadcast
		if err := service.UnmarshalBody(r, &bc); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal body")
		}

		if bc.ProjectKey != "" {
			proj, errProj := project.Load(api.mustDB(), api.Cache, bc.ProjectKey)
			if errProj != nil {
				return sdk.WrapError(sdk.ErrNoProject, "Cannot load %s", bc.ProjectKey)
			}
			bc.ProjectID = &proj.ID
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "Unable to start transaction")
		}

		defer tx.Rollback() // nolint

		if bc.ID <= 0 || broadcastID != bc.ID {
			return sdk.WrapError(sdk.ErrWrongRequest, "%d is not valid. id in path:%d", bc.ID, broadcastID)
		}

		// update broadcast in db
		if err := broadcast.Update(tx, &bc); err != nil {
			return sdk.WrapError(err, "Cannot update broadcast")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Unable to commit transaction")
		}

		event.PublishBroadcastUpdate(ctx, *oldBC, bc, getAPIConsumer(ctx))
		return service.WriteJSON(w, bc, http.StatusOK)
	}
}

func (api *API) postMarkAsReadBroadcastHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcastID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "Invalid id")
		}

		consumer := getAPIConsumer(ctx)
		br, errL := broadcast.LoadByID(api.mustDB(), broadcastID, consumer.AuthentifiedUser)
		if errL != nil {
			return sdk.WrapError(errL, "Cannot load broadcast by id")
		}

		if !br.Read {
			if err := broadcast.MarkAsRead(api.mustDB(), broadcastID, consumer.AuthentifiedUser.ID); err != nil {
				return sdk.WrapError(err, "Cannot mark as read broadcast id %d and user id %d", broadcastID, broadcastID)
			}
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) deleteBroadcastHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		broadcastID, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "Invalid id")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}

		if err := broadcast.Delete(tx, broadcastID); err != nil {
			return sdk.WrapError(err, "Cannot delete broadcast")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishBroadcastDelete(ctx, broadcastID, getAPIConsumer(ctx))
		return nil
	}
}

func (api *API) getBroadcastHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, errr := requestVarInt(r, "id")
		if errr != nil {
			return sdk.WrapError(errr, "Invalid id")
		}

		consumer := getAPIConsumer(ctx)
		broadcast, err := broadcast.LoadByID(api.mustDB(), id, consumer.AuthentifiedUser)
		if err != nil {
			return sdk.WrapError(err, "Cannot load broadcasts")
		}

		return service.WriteJSON(w, broadcast, http.StatusOK)
	}
}

func (api *API) getBroadcastsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		consumer := getAPIConsumer(ctx)
		broadcasts, err := broadcast.LoadAll(api.mustDB(), consumer.AuthentifiedUser)
		if err != nil {
			return sdk.WrapError(err, "Cannot load broadcasts")
		}

		return service.WriteJSON(w, broadcasts, http.StatusOK)
	}
}
