package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postMaintenanceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		enable := FormBool(r, "enable")
		hook := FormBool(r, "withHook")

		if hook {
			srvs, err := services.LoadAllByType(ctx, api.mustDB(), services.TypeHooks)
			if err != nil {
				return err
			}
			url := fmt.Sprintf("/admin/maintenance?enable=%v", enable)
			_, code, errHooks := services.DoJSONRequest(ctx, api.mustDB(), srvs, http.MethodPost, url, nil, nil)
			if errHooks != nil || code >= 400 {
				return fmt.Errorf("unable to change hook maintenant state to %v. Code result %d: %v", enable, code, errHooks)
			}
		}

		if err := api.Cache.SetWithTTL(sdk.MaintenanceAPIKey, enable, 0); err != nil {
			return err
		}
		return api.Cache.Publish(ctx, sdk.MaintenanceQueueName, fmt.Sprintf("%v", enable))
	}
}

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
			srvs, err = services.LoadAllByType(ctx, api.mustDB(), r.FormValue("type"))
		} else {
			srvs, err = services.LoadAll(ctx, api.mustDB())
		}
		if err != nil {
			return err
		}

		return service.WriteJSON(w, srvs, http.StatusOK)
	}
}

func (api *API) deleteAdminServiceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		srv, err := services.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return err
		}
		if err := services.Delete(api.mustDB(), srv); err != nil {
			return err
		}
		return service.WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) getAdminServiceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		srv, err := services.LoadByName(ctx, api.mustDB(), name)
		if err != nil {
			return err
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
		var srvs []sdk.Service
		if r.FormValue("name") != "" {
			srv, err := services.LoadByName(ctx, api.mustDB(), r.FormValue("name"))
			if err != nil {
				return err
			}
			if srv != nil {
				srvs = []sdk.Service{*srv}
			}
		} else {
			var errFind error
			srvs, errFind = services.LoadAllByType(ctx, api.mustDB(), r.FormValue("type"))
			if errFind != nil {
				return errFind
			}
		}

		if len(srvs) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "No service found")
		}

		query := r.FormValue("query")
		btes, _, code, err := services.DoRequest(ctx, api.mustDB(), srvs, method, query, nil)
		if err != nil {
			return sdk.NewError(sdk.Error{
				Status:  code,
				Message: err.Error(),
			}, err)
		}

		log.Debug("selectDeleteAdminServiceCallHandler> %s : %s", query, string(btes))

		if strings.HasPrefix(query, "/debug/pprof/") {
			return service.Write(w, btes, code, "text/plain")
		}
		return service.Write(w, btes, code, "application/json")
	}
}

func putPostAdminServiceCallHandler(api *API, method string) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srvs, err := services.LoadAllByType(ctx, api.mustDB(), r.FormValue("type"))
		if err != nil {
			return err
		}

		query := r.FormValue("query")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}
		defer r.Body.Close()

		btes, _, code, err := services.DoRequest(ctx, api.mustDB(), srvs, method, query, body)
		if err != nil {
			return sdk.NewError(sdk.Error{
				Status:  code,
				Message: err.Error(),
			}, err)
		}

		return service.Write(w, btes, code, "application/json")
	}
}

func (api *API) getAdminDatabaseSignatureResume() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var entities = gorpmapping.ListSignedEntities()
		var resume = make(sdk.CanonicalFormUsageResume, len(entities))

		for _, e := range entities {
			data, err := gorpmapping.ListCanonicalFormsByEntity(api.mustDB(), e)
			if err != nil {
				return err
			}
			resume[e] = data
		}

		return service.WriteJSON(w, resume, http.StatusOK)
	}
}

func (api *API) getAdminDatabaseSignatureTuplesBySigner() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]
		signer := vars["signer"]

		pks, err := gorpmapping.ListTupleByCanonicalForm(api.mustDB(), entity, signer)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, pks, http.StatusOK)
	}
}

func (api *API) postAdminDatabaseSignatureRollEntityByPrimaryKey() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]
		pk := vars["pk"]

		if err := gorpmapping.RollSignedTupleByPrimaryKey(ctx, api.mustDB(), entity, pk); err != nil {
			return err
		}

		return nil
	}
}

func (api *API) getAdminDatabaseEncryptedEntities() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, gorpmapping.ListEncryptedEntities(), http.StatusOK)
	}
}

func (api *API) getAdminDatabaseEncryptedTuplesByEntity() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]

		pks, err := gorpmapping.ListTuplesByEntity(api.mustDB(), entity)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, pks, http.StatusOK)
	}
}

func (api *API) postAdminDatabaseRollEncryptedEntityByPrimaryKey() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		entity := vars["entity"]
		pk := vars["pk"]

		if err := gorpmapping.RollEncryptedTupleByPrimaryKey(api.mustDB(), entity, pk); err != nil {
			return err
		}

		return nil
	}
}
