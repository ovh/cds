package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/actionplugin"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postPGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var p sdk.GRPCPlugin
		db := api.mustDB()

		if err := UnmarshalBody(r, &p); err != nil {
			return sdk.WrapError(err, "postPGRPCluginHandler>")
		}
		p.Binaries = nil

		fmt.Printf("--- +%v\n", p)

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "postPGRPCluginHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if p.Type == sdk.GRPCPluginAction {
			// Check that action does not already exists
			conflict, err := action.Exists(db, p.Name)
			if err != nil {
				return sdk.WrapError(err, "postPGRPCluginHandler> %v", err)
			}
			if conflict {
				return sdk.ErrConflict
			}

			//Insert in database
			if _, err := actionplugin.InsertWithGRPCPlugin(tx, &p, p.Parameters); err != nil {
				return sdk.WrapError(err, "postPGRPCluginHandler> Error while inserting action %s in database", p.Name)
			}
		}

		if err := plugin.Insert(tx, &p); err != nil {
			return sdk.WrapError(err, "postPGRPCluginHandler> unable to insert plugin")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postPGRPCluginHandler> Cannot commit transaction")
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getAllGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		ps, err := plugin.LoadAll(api.mustDB())
		if err != nil {
			return sdk.WrapError(err, "getAllGRPCluginHandler> unable to load all plugins")
		}

		return service.WriteJSON(w, ps, http.StatusOK)
	}
}

func (api *API) getGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var name = mux.Vars(r)["name"]

		p, err := plugin.LoadByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getGRPCluginHandler")
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) putGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var p sdk.GRPCPlugin
		if err := UnmarshalBody(r, &p); err != nil {
			return sdk.WrapError(err, "putGRPCluginHandler>")
		}

		var name = mux.Vars(r)["name"]
		old, err := plugin.LoadByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "putGRPCluginHandler> unable to load old plugin")
		}

		p.ID = old.ID
		p.Binaries = old.Binaries

		if err := plugin.Update(api.mustDB(), &p); err != nil {
			return sdk.WrapError(err, "putGRPCluginHandler> unable to insert plugin")
		}

		return service.WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) deleteGRPCluginHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var name = mux.Vars(r)["name"]
		old, err := plugin.LoadByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "deleteGRPCluginHandler> unable to load old plugin")
		}

		if err := plugin.Delete(api.mustDB(), old); err != nil {
			return sdk.WrapError(err, "deleteGRPCluginHandler> unable to delete plugin")
		}

		return nil
	}
}

func (api *API) postGRPCluginBinaryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var b sdk.GRPCPluginBinary
		if err := UnmarshalBody(r, &b); err != nil {
			return sdk.WrapError(err, "postGRPCluginBinaryHandler")
		}

		if len(b.FileContent) == 0 || b.OS == "" || b.Arch == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "postGRPCluginBinaryHandler")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "postGRPCluginBinaryHandler> unable to start tx")
		}
		defer tx.Rollback()

		name := mux.Vars(r)["name"]
		p, err := plugin.LoadByName(tx, name)
		if err != nil {
			return sdk.WrapError(err, "postGRPCluginBinaryHandler")
		}

		buff := bytes.NewBuffer(b.FileContent)

		old := p.GetBinary(b.OS, b.Arch)
		if old == nil {
			if err := plugin.AddBinary(tx, p, &b, ioutil.NopCloser(buff)); err != nil {
				return sdk.WrapError(err, "postGRPCluginBinaryHandler> unable to add plugin binary")
			}
		} else {
			if err := plugin.UpdateBinary(tx, p, &b, ioutil.NopCloser(buff)); err != nil {
				return sdk.WrapError(err, "postGRPCluginBinaryHandler> unable to add plugin binary")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postGRPCluginBinaryHandler> unable to commit tx")
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

		p, err := plugin.LoadByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getGRPCluginBinaryHandler")
		}

		b := p.GetBinary(os, arch)
		if b == nil {
			return sdk.WrapError(sdk.ErrNotFound, "getGRPCluginBinaryHandler")
		}

		acceptRedirect := FormBool(r, "accept-redirect")
		if acceptRedirect && objectstore.Instance().TemporaryURLSupported {
			url, err := objectstore.FetchTempURL(b)
			if err != nil {
				return sdk.WrapError(err, "getGRPCluginBinaryHandler> unable to get a temp URL")
			}
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			return nil
		}

		f, err := objectstore.Fetch(b)
		if err != nil {
			return sdk.WrapError(err, "getGRPCluginBinaryHandler> unable to get object")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", b.Name))

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "getGRPCluginBinaryHandler> Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "getGRPCluginBinaryHandler> Cannot close artifact")
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

		p, err := plugin.LoadByName(api.mustDB(), name)
		if err != nil {
			return sdk.WrapError(err, "getGRPCluginBinaryInfosHandler>")
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
		return nil
	}
}
