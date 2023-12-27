package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/actionplugin"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var p sdk.GRPCPlugin
		db := api.mustDB()
		if err := service.UnmarshalBody(r, &p); err != nil {
			return sdk.WithStack(err)
		}
		if err := p.Validate(); err != nil {
			return sdk.WithStack(err)
		}
		p.Binaries = nil

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() //nolint

		// a plugin can be attached to a integration model OR not, for "action plugin"
		if p.Integration != "" {
			integrationModel, err := integration.LoadModelByName(ctx, api.mustDB(), p.Integration)
			if err != nil {
				return err
			}
			p.IntegrationModelID = &integrationModel.ID
		}

		if p.Type == sdk.GRPCPluginAction {
			old, err := action.LoadByTypesAndName(ctx, db, []string{sdk.PluginAction}, p.Name, action.LoadOptions.Default)
			if err != nil {
				return sdk.WithStack(err)
			}
			if old != nil {
				if _, err := actionplugin.UpdateGRPCPlugin(ctx, tx, &p, p.Inputs); err != nil {
					return sdk.WrapError(err, "error while updating action %s in database", p.Name)
				}
			} else {
				if _, err := actionplugin.InsertWithGRPCPlugin(tx, &p, p.Inputs); err != nil {
					return sdk.WrapError(err, "error while inserting action %s in database", p.Name)
				}
			}
		}

		if err := plugin.Insert(tx, &p); err != nil {
			return sdk.WrapError(err, "unable to insert plugin")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getAllGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ps, err := plugin.LoadAll(ctx, api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "unable to load all plugins")
		}

		return service.WriteJSON(w, ps, http.StatusOK)
	}
}

func (api *API) getGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := mux.Vars(r)["name"]

		p, err := plugin.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getGRPCluginHandler")
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) putGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := mux.Vars(r)["name"]

		db := api.mustDB()
		var p sdk.GRPCPlugin
		if err := service.UnmarshalBody(r, &p); err != nil {
			return sdk.WithStack(err)
		}
		if err := p.Validate(); err != nil {
			return sdk.WithStack(err)
		}

		old, err := plugin.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "unable to load old plugin")
		}

		p.ID = old.ID
		p.Binaries = old.Binaries

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback() //nolint

		// a plugin can be attached to a integration model OR not, for "action plugin"
		if p.Integration != "" {
			integrationModel, err := integration.LoadModelByName(ctx, api.mustDB(), p.Integration)
			if err != nil {
				return err
			}
			p.IntegrationModelID = &integrationModel.ID
		}

		if p.Type == sdk.GRPCPluginAction {
			if _, err := actionplugin.UpdateGRPCPlugin(ctx, tx, &p, p.Inputs); err != nil {
				return sdk.WrapError(err, "Error while updating action %s in database", p.Name)
			}
		}

		if err := plugin.Update(tx, &p); err != nil {
			return sdk.WrapError(err, "unable to insert plugin")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) deleteGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := mux.Vars(r)["name"]

		old, err := plugin.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "unable to load old plugin")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()

		if old.Type == sdk.GRPCPluginAction {
			if err := actionplugin.DeleteGRPCPlugin(ctx, tx, old); err != nil {
				return err
			}
		}

		if err := plugin.Delete(ctx, tx, api.SharedStorage, old); err != nil {
			return sdk.WrapError(err, "unable to delete plugin")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event_v2.PublishPluginDeleteEvent(ctx, api.Cache, *old, getUserConsumer(ctx).AuthConsumerUser.AuthentifiedUser)

		return nil
	}
}

func (api *API) postGRPCluginBinaryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]

		var b sdk.GRPCPluginBinary
		if err := service.UnmarshalBody(r, &b); err != nil {
			return sdk.WrapError(err, "postGRPCluginBinaryHandler")
		}

		if len(b.FileContent) == 0 || b.OS == "" || b.Arch == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "postGRPCluginBinaryHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start tx")
		}
		defer tx.Rollback() // nolint

		p, err := plugin.LoadByName(ctx, tx, name)
		if err != nil {
			return sdk.WrapError(err, "postGRPCluginBinaryHandler")
		}

		buff := bytes.NewBuffer(b.FileContent)

		old := p.GetBinary(b.OS, b.Arch)
		if old == nil {
			if err := plugin.AddBinary(ctx, tx, api.SharedStorage, p, &b, io.NopCloser(buff)); err != nil {
				return sdk.WrapError(err, "unable to add plugin binary")
			}
		} else {
			if err := plugin.UpdateBinary(ctx, tx, api.SharedStorage, p, &b, io.NopCloser(buff)); err != nil {
				return sdk.WrapError(err, "unable to add plugin binary")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getGRPCluginBinaryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		os := vars["os"]
		arch := vars["arch"]

		p, err := plugin.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getGRPCluginBinaryHandler")
		}

		b := p.GetBinary(os, arch)
		if b == nil {
			return sdk.WrapError(sdk.ErrNotFound, "getGRPCluginBinaryHandler")
		}

		acceptRedirect := service.FormBool(r, "accept-redirect")

		s, temporaryURLSupported := api.SharedStorage.(objectstore.DriverWithRedirect)
		if acceptRedirect && api.SharedStorage.TemporaryURLSupported() && temporaryURLSupported {
			url, _, err := s.FetchURL(b)
			if err != nil {
				return sdk.WrapError(err, "unable to get a temp URL")
			}
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			return nil
		}

		f, err := api.SharedStorage.Fetch(ctx, b)
		if err != nil {
			return sdk.WrapError(err, "unable to get object")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", b.Name))

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "Cannot close artifact")
		}

		return nil
	}
}

func (api *API) getGRPCluginBinaryInfosHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		os := vars["os"]
		arch := vars["arch"]

		p, err := plugin.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return sdk.WithStack(err)
		}

		b := p.GetBinary(os, arch)
		if b == nil {
			return sdk.WrapError(sdk.ErrNotFound, "getGRPCluginBinaryInfosHandler>")
		}

		return service.WriteJSON(w, *b, http.StatusOK)
	}
}

func (api *API) deleteGRPCluginBinaryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		name := vars["name"]
		os := vars["os"]
		arch := vars["arch"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start tx")
		}
		defer tx.Rollback() // nolint

		p, err := plugin.LoadByName(ctx, tx, name)
		if err != nil {
			return sdk.WrapError(err, "unable to load plugin")
		}

		if err := plugin.DeleteBinary(ctx, tx, api.SharedStorage, p, os, arch); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
