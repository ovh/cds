package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postOrganizationMigrateUserHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		orgaIdentifier := vars["organizationIdentifier"]

		orga, err := api.getOrganizationByIdentifier(ctx, orgaIdentifier)
		if err != nil {
			return err
		}

		users, err := user.LoadUsersWithoutOrganization(ctx, api.mustDB())
		if err != nil {
			return err
		}

		for i := range users {
			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			u := &users[i]

			if err := api.userSetOrganization(ctx, tx, u, orga.Name); err != nil {
				_ = tx.Rollback()
				return err
			}
			if err := tx.Commit(); err != nil {
				_ = tx.Rollback()
				return sdk.WithStack(err)
			}
		}
		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postAdminOrganizationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var org sdk.Organization
		if err := service.UnmarshalBody(r, &org); err != nil {
			return err
		}
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		if err := organization.Insert(ctx, tx, &org); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
		return service.WriteMarshal(w, r, nil, http.StatusCreated)
	}
}

func (api *API) getAdminOrganizationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		orgas, err := organization.LoadAllOrganizations(ctx, api.mustDB())
		if err != nil {
			return err
		}
		return service.WriteMarshal(w, r, orgas, http.StatusOK)
	}
}

func (api *API) deleteAdminOrganizationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		orgaIdentifier := vars["organizationIdentifier"]

		orga, err := api.getOrganizationByIdentifier(ctx, orgaIdentifier)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		if err := organization.Delete(tx, orga.ID); err != nil {
			return err
		}
		return sdk.WithStack(tx.Commit())
	}
}

func (api *API) postMaintenanceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		enable := service.FormBool(r, "enable")
		hook := service.FormBool(r, "withHook")

		if hook {
			srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeHooks)
			if err != nil {
				return err
			}
			url := fmt.Sprintf("/admin/maintenance?enable=%v", enable)
			_, code, errHooks := services.NewClient(api.mustDB(), srvs).DoJSONRequest(ctx, http.MethodPost, url, nil, nil)
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

func (api *API) getAdminServicesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var srvs []sdk.Service
		var err error
		if r.FormValue("type") != "" {
			srvs, err = services.LoadAllByType(ctx, api.mustDB(), r.FormValue("type"), services.LoadOptions.WithStatus)
		} else {
			srvs, err = services.LoadAll(ctx, api.mustDB(), services.LoadOptions.WithStatus)
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
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()
		if err := services.Delete(tx, srv); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}
		return service.WriteJSON(w, srv, http.StatusOK)
	}
}

func (api *API) getAdminServiceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		srv, err := services.LoadByName(ctx, api.mustDB(), name, services.LoadOptions.WithStatus)
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
		btes, _, code, err := services.DoRequest(ctx, srvs, method, query, nil)
		if err != nil {
			return sdk.NewError(sdk.Error{
				Status:  code,
				Message: err.Error(),
			}, err)
		}
		reader := bytes.NewReader(btes)

		log.Debug(ctx, "selectDeleteAdminServiceCallHandler> %s : %s", query, string(btes))

		if strings.HasPrefix(query, "/debug/pprof/") {
			return service.Write(w, reader, code, "text/plain")
		}
		return service.Write(w, reader, code, "application/json")
	}
}

func putPostAdminServiceCallHandler(api *API, method string) service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		srvs, err := services.LoadAllByType(ctx, api.mustDB(), r.FormValue("type"))
		if err != nil {
			return err
		}

		query := r.FormValue("query")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}
		defer r.Body.Close()

		btes, _, code, err := services.DoRequest(ctx, srvs, method, query, body)
		if err != nil {
			return sdk.NewError(sdk.Error{
				Status:  code,
				Message: err.Error(),
			}, err)
		}

		return service.Write(w, bytes.NewReader(btes), code, "application/json")
	}
}

func (api *API) deleteDatabaseMigrationHandler() service.Handler {
	return database.AdminDeleteDatabaseMigration(api.mustDB)
}

func (api *API) postDatabaseMigrationUnlockedHandler() service.Handler {
	return database.AdminDatabaseMigrationUnlocked(api.mustDB)
}

func (api *API) getDatabaseMigrationHandler() service.Handler {
	return database.AdminGetDatabaseMigration(api.mustDB)
}

func (api *API) getAdminDatabaseSignatureResume() service.Handler {
	return database.AdminDatabaseSignatureResume(api.mustDB, gorpmapping.Mapper)
}

func (api *API) getAdminDatabaseSignatureTuplesBySigner() service.Handler {
	return database.AdminDatabaseSignatureTuplesBySigner(api.mustDB, gorpmapping.Mapper)
}

func (api *API) postAdminDatabaseSignatureRollEntityByPrimaryKey() service.Handler {
	return database.AdminDatabaseSignatureRollEntityByPrimaryKey(api.mustDB, gorpmapping.Mapper)
}

func (api *API) getAdminDatabaseEncryptedEntities() service.Handler {
	return database.AdminDatabaseEncryptedEntities(api.mustDB, gorpmapping.Mapper)
}

func (api *API) getAdminDatabaseEncryptedTuplesByEntity() service.Handler {
	return database.AdminDatabaseEncryptedTuplesByEntity(api.mustDB, gorpmapping.Mapper)
}

func (api *API) postAdminDatabaseRollEncryptedEntityByPrimaryKey() service.Handler {
	return database.AdminDatabaseRollEncryptedEntityByPrimaryKey(api.mustDB, gorpmapping.Mapper)
}

func (api *API) getAdminFeatureFlipping() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		all, err := featureflipping.LoadAll(ctx, gorpmapping.Mapper, api.mustDB())
		if err != nil {
			return err
		}
		return service.WriteJSON(w, all, http.StatusOK)
	}
}

func (api *API) getAdminFeatureFlippingByName() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := sdk.FeatureName(vars["name"])

		f, err := featureflipping.LoadByName(ctx, gorpmapping.Mapper, api.mustDB(), name)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, f, http.StatusOK)
	}
}

func (api *API) postAdminFeatureFlipping() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var f sdk.Feature
		if err := service.UnmarshalBody(r, &f); err != nil {
			return err
		}

		if err := featureflipping.Insert(gorpmapping.Mapper, api.mustDB(), &f); err != nil {
			return err
		}
		return service.WriteJSON(w, f, http.StatusOK)
	}
}

func (api *API) putAdminFeatureFlipping() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := sdk.FeatureName(vars["name"])

		var f sdk.Feature
		if err := service.UnmarshalBody(r, &f); err != nil {
			return err
		}

		oldF, err := featureflipping.LoadByName(ctx, gorpmapping.Mapper, api.mustDB(), name)
		if err != nil {
			return err
		}

		if name != f.Name {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		f.ID = oldF.ID
		if err := featureflipping.Update(gorpmapping.Mapper, api.mustDB(), &f); err != nil {
			return err
		}

		featureflipping.InvalidateCache(ctx, f.Name)

		return service.WriteJSON(w, f, http.StatusOK)
	}
}

func (api *API) deleteAdminFeatureFlipping() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := sdk.FeatureName(vars["name"])

		feature, err := featureflipping.LoadByName(ctx, gorpmapping.Mapper, api.mustDB(), name)
		if err != nil {
			return err
		}

		if err := featureflipping.Delete(api.mustDB(), feature.ID); err != nil {
			return err
		}

		featureflipping.InvalidateCache(ctx, feature.Name)

		return nil
	}
}

func (api *API) postWorkflowMaxRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		var request sdk.UpdateMaxRunRequest
		if err := service.UnmarshalBody(r, &request); err != nil {
			return err
		}

		proj, err := project.Load(ctx, api.mustDBWithCtx(ctx), key)
		if err != nil {
			return err
		}

		wf, err := workflow.Load(ctx, api.mustDBWithCtx(ctx), api.Cache, *proj, name, workflow.LoadOptions{})
		if err != nil {
			return err
		}

		if err := workflow.UpdateMaxRunsByID(api.mustDBWithCtx(ctx), wf.ID, request.MaxRuns); err != nil {
			return err
		}

		return nil
	}
}
