package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) adminTruncateWarningsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, err := api.mustDB().Exec("delete from warning"); err != nil {
			return sdk.WrapError(err, "Unable to truncate warning ")
		}
		return nil
	}
}

func (api *API) getAdminServicesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srvs := []sdk.Service{}

		var err error
		if r.FormValue("type") != "" {
			srvs, err = services.FindByType(api.mustDB(), r.FormValue("type"))
		} else {
			srvs, err = services.All(api.mustDB())
		}
		if err != nil {
			return err
		}

		for i := range srvs {
			srv := &srvs[i]
			srv.Hash = ""
			srv.Token = ""
		}

		return service.WriteJSON(w, srvs, http.StatusOK)
	}
}

func (api *API) getAdminServiceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		srv, err := services.FindByName(api.mustDB(), name)
		if err != nil {
			return err
		}
		srv.Hash = ""
		srv.Token = ""
		if srv.GroupID != nil {
			g, err := group.LoadGroupByID(api.mustDB(), *srv.GroupID)
			if err != nil {
				if !sdk.ErrorIs(err, sdk.ErrGroupNotFound) {
					return sdk.WithStack(err)
				}
			} else {
				srv.Group = g
			}
		}
		return service.WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) getAdminServiceCallHandler() service.Handler {
	return selectDeleteAdminServiceCallHandler(api, http.MethodGet)
}

func (api *API) deleteAdminServiceCallHandler() service.Handler {
	return selectDeleteAdminServiceCallHandler(api, http.MethodDelete)
}

func (api *API) postAdminServiceCallHandler() service.Handler {
	return putPostAdminServiceCallHandler(api, http.MethodPost)
}

func (api *API) putAdminServiceCallHandler() service.Handler {
	return putPostAdminServiceCallHandler(api, http.MethodPut)
}

func selectDeleteAdminServiceCallHandler(api *API, method string) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srvs, err := services.FindByType(api.mustDB(), r.FormValue("type"))
		if err != nil {
			return err
		}
		if len(srvs) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "No hooks service found")
		}

		query := r.FormValue("query")
		btes, code, err := services.DoRequest(ctx, srvs, method, query, nil)
		if err != nil {
			return sdk.NewError(sdk.Error{
				Status:  code,
				Message: err.Error(),
			}, err)
		}

		log.Debug("selectDeleteAdminServiceCallHandler> %s : %s", query, string(btes))

		//TODO: assuming it's only json...
		return service.Write(w, btes, code, "application/json")
	}
}

func putPostAdminServiceCallHandler(api *API, method string) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srvs, err := services.FindByType(api.mustDB(), r.FormValue("type"))
		if err != nil {
			return err
		}

		query := r.FormValue("query")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}
		defer r.Body.Close()

		btes, code, err := services.DoRequest(ctx, srvs, method, query, body)
		if err != nil {
			return sdk.NewError(sdk.Error{
				Status:  code,
				Message: err.Error(),
			}, err)
		}

		return service.Write(w, btes, code, "application/json")
	}
}
