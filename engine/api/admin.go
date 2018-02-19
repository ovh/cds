package api

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) adminTruncateWarningsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, err := api.mustDB().Exec("delete from warning"); err != nil {
			return sdk.WrapError(err, "adminTruncateWarningsHandler> Unable to truncate warning ")
		}
		return nil
	}
}

func (api *API) postAdminMaintenanceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		api.Cache.SetWithTTL("maintenance", true, -1)
		return nil
	}
}

func (api *API) getAdminMaintenanceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var m bool
		api.Cache.Get("maintenance", &m)
		return WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) deleteAdminMaintenanceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		api.Cache.Delete("maintenance")
		return nil
	}
}

func (api *API) getAdminServicesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		q := services.Querier(api.mustDB(), api.Cache)
		srvs, err := q.All()
		if err != nil {
			return sdk.WrapError(err, "getAdminServicesHandler")
		}
		return WriteJSON(w, srvs, http.StatusOK)
	}
}

func (api *API) getAdminServiceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		s := vars["service"]
		q := services.Querier(api.mustDB(), api.Cache)
		srvs, err := q.FindByType(s)
		if err != nil {
			return sdk.WrapError(err, "getAdminServiceHandler")
		}
		return WriteJSON(w, srvs, http.StatusOK)
	}
}

func (api *API) getAdminServiceCallHandler() Handler {
	return selectDeleteAdminServiceCallHandler(api, http.MethodGet)
}

func (api *API) deleteAdminServiceCallHandler() Handler {
	return selectDeleteAdminServiceCallHandler(api, http.MethodDelete)
}

func (api *API) postAdminServiceCallHandler() Handler {
	return putPostAdminServiceCallHandler(api, http.MethodPost)
}

func (api *API) putAdminServiceCallHandler() Handler {
	return putPostAdminServiceCallHandler(api, http.MethodPut)
}

func selectDeleteAdminServiceCallHandler(api *API, method string) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		s := vars["service"]
		q := services.Querier(api.mustDB(), api.Cache)
		srvs, err := q.FindByType(s)
		if err != nil {
			return sdk.WrapError(err, "selectDeleteAdminServiceCallHandler")
		}

		query := r.FormValue("query")
		btes, code, err := services.DoRequest(srvs, method, query, nil)
		if err != nil {
			sdkErr := sdk.Error{
				Status:  code,
				Message: err.Error(),
			}
			return sdkErr
		}

		log.Debug("selectDeleteAdminServiceCallHandler> %s : %s", query, string(btes))

		//TODO: assuming it's only json...
		return Write(w, btes, code, "application/json")
	}
}

func putPostAdminServiceCallHandler(api *API, method string) Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		s := vars["service"]
		q := services.Querier(api.mustDB(), api.Cache)
		srvs, err := q.FindByType(s)
		if err != nil {
			return sdk.WrapError(err, "putPostAdminServiceCallHandler")
		}

		query := r.FormValue("query")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "putPostAdminServiceCallHandler> Unable to read body")
		}
		defer r.Body.Close()

		btes, code, err := services.DoRequest(srvs, method, query, body)
		if err != nil {
			sdkErr := sdk.Error{
				Status:  code,
				Message: err.Error(),
			}
			return sdkErr
		}

		return Write(w, btes, code, "application/json")
	}
}
