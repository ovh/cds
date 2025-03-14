package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectConcurrenciesHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			key := vars["projectKey"]

			concurrencies, err := project.LoadConcurrenciesByProjectKey(ctx, api.mustDB(), key)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, concurrencies, http.StatusOK)
		}
}

func (api *API) postProjectConcurrencyHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			key := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var projConcu sdk.ProjectConcurrency
			if err := service.UnmarshalBody(r, &projConcu); err != nil {
				return sdk.WrapError(err, "cannot read body")
			}
			projConcu.ProjectKey = key

			if err := (&projConcu).Check(); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()
			if err := project.InsertConcurrency(ctx, tx, &projConcu); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			event_v2.PublishConcurrencyEvent(ctx, api.Cache, sdk.EventConcurrencyCreated, key, projConcu, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteJSON(w, projConcu, http.StatusOK)
		}
}

func (api *API) putProjectConcurrencyHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			key := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var projConcu sdk.ProjectConcurrency
			if err := service.UnmarshalBody(r, &projConcu); err != nil {
				return sdk.WrapError(err, "cannot read body")
			}

			if err := (&projConcu).Check(); err != nil {
				return err
			}

			if _, err := project.LoadConcurrencyByIDAndProjectKey(ctx, api.mustDB(), key, projConcu.ID); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()
			if err := project.UpdateConcurrency(ctx, tx, &projConcu); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			event_v2.PublishConcurrencyEvent(ctx, api.Cache, sdk.EventConcurrencyUpdated, key, projConcu, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteJSON(w, projConcu, http.StatusOK)
		}
}

func (api *API) getProjectConcurrencyHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			key := vars["projectKey"]
			concurrencyName := vars["concurrencyName"]

			concurrency, err := project.LoadConcurrencyByNameAndProjectKey(ctx, api.mustDB(), key, concurrencyName)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, concurrency, http.StatusOK)
		}
}

func (api *API) getProjectConcurrencyRunsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			key := vars["projectKey"]
			concurrencyName := vars["concurrencyName"]

			pcrs, err := workflow_v2.LoadProjectConccurencyRunObjects(ctx, api.mustDB(), key, concurrencyName)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, pcrs, http.StatusOK)
		}
}

func (api *API) deleteProjectConcurrencyHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			key := vars["projectKey"]
			concurrencyName := vars["concurrencyName"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			concurrency, err := project.LoadConcurrencyByNameAndProjectKey(ctx, api.mustDB(), key, concurrencyName)
			if err != nil {
				return err
			}
			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()
			if err := project.DeleteConcurrency(tx, key, concurrency.ID); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			event_v2.PublishConcurrencyEvent(ctx, api.Cache, sdk.EventConcurrencyDeleted, key, *concurrency, *u.AuthConsumerUser.AuthentifiedUser)

			return nil
		}
}
