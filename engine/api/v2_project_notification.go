package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/notification_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectNotifsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			notifs, err := notification_v2.LoadAll(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, notifs, http.StatusOK)
		}
}

// getProjectNotificationHandler Retrieve a project notification
func (api *API) getProjectNotificationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			notifName := vars["notification"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			n, err := notification_v2.LoadByName(ctx, api.mustDB(), pKey, notifName)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, n, http.StatusOK)
		}
}

// postProjectNotificationHandler Attach a new notification to the project
func (api *API) postProjectNotificationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManageNotification),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var n sdk.ProjectNotification
			if err := service.UnmarshalBody(req, &n); err != nil {
				return err
			}

			p, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			_, err = notification_v2.LoadByName(ctx, api.mustDB(), p.Key, n.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}

			n.ProjectKey = p.Key

			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() //

			if err := notification_v2.Insert(ctx, tx, &n); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishProjectNotificationEvent(ctx, api.Cache, sdk.EventNotificationCreated, p.Key, n, *u.AuthConsumerUser.AuthentifiedUser)

			return service.WriteJSON(w, n, http.StatusOK)
		}
}

// putProjectNotificationHandler Update a project notification
func (api *API) putProjectNotificationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManageNotification),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			notifName := vars["notification"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var n sdk.ProjectNotification
			if err := service.UnmarshalBody(req, &n); err != nil {
				return err
			}

			if n.Name != notifName {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "wrong notification name")
			}

			oldNotif, err := notification_v2.LoadByName(ctx, api.mustDB(), pKey, notifName)
			if err != nil {
				return err
			}

			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() //

			n.ID = oldNotif.ID
			n.ProjectKey = oldNotif.ProjectKey
			if err := notification_v2.Update(ctx, tx, &n); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishProjectNotificationEvent(ctx, api.Cache, sdk.EventNotificationUpdated, pKey, n, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteJSON(w, n, http.StatusOK)
		}
}

// deleteProjectNotificationHandler delete a project notification
func (api *API) deleteProjectNotificationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManageNotification),
		func(ctx context.Context, _ http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			notifName := vars["notification"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			n, err := notification_v2.LoadByName(ctx, api.mustDB(), pKey, notifName)
			if err != nil {
				return err
			}

			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() //

			if err := notification_v2.Delete(ctx, tx, n); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishProjectNotificationEvent(ctx, api.Cache, sdk.EventNotificationDeleted, pKey, *n, *u.AuthConsumerUser.AuthentifiedUser)
			return nil
		}
}
