package api

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/project_secret"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectSecretsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			secrets, err := project_secret.LoadByProjectKey(ctx, api.mustDB(), proj.Key)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, secrets, http.StatusOK)
		}
}

func (api *API) deleteProjectSecretHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, _ http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			name := vars["name"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			secret, err := project_secret.LoadByName(ctx, api.mustDB(), proj.Key, name)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := project_secret.Delete(ctx, tx, *secret); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			event_v2.PublishProjectSecretEvent(ctx, api.Cache, sdk.EventSecretDeleted, *secret, *u.AuthConsumerUser.AuthentifiedUser)

			return nil
		}
}
func (api *API) putProjectSecretHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, _ http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			name := vars["name"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var secret sdk.ProjectSecret
			if err := service.UnmarshalBody(req, &secret); err != nil {
				return err
			}

			reg, err := regexp.Compile(sdk.SecretNamePattern)
			if err != nil {
				return sdk.WithStack(err)
			}

			if !reg.MatchString(secret.Name) {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "secret name doesn't match pattern %s", sdk.SecretNamePattern)
			}

			if name != secret.Name {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "wrong secret name")
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			existingSecret, err := project_secret.LoadByName(ctx, api.mustDB(), proj.Key, secret.Name)
			if err != nil {
				return err
			}

			secret.ProjectKey = existingSecret.ProjectKey
			secret.ID = existingSecret.ID

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := project_secret.Update(ctx, tx, &secret); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			event_v2.PublishProjectSecretEvent(ctx, api.Cache, sdk.EventSecretUpdated, secret, *u.AuthConsumerUser.AuthentifiedUser)

			return nil

		}
}

func (api *API) postProjectSecretHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, _ http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var secret sdk.ProjectSecret
			if err := service.UnmarshalBody(req, &secret); err != nil {
				return err
			}

			reg, err := regexp.Compile(sdk.SecretNamePattern)
			if err != nil {
				return sdk.WithStack(err)
			}

			if !reg.MatchString(secret.Name) {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "secret name doesn't match pattern %s", sdk.SecretNamePattern)
			}

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			existingSecret, err := project_secret.LoadByName(ctx, api.mustDB(), proj.Key, secret.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}

			if existingSecret != nil {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "this secret already exists")
			}
			secret.ProjectKey = proj.Key

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := project_secret.Insert(ctx, tx, &secret); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			event_v2.PublishProjectSecretEvent(ctx, api.Cache, sdk.EventSecretCreated, secret, *u.AuthConsumerUser.AuthentifiedUser)

			return nil
		}
}
