package api

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// getProjectVariableSetHandler retrieve the given variable set
func (api *API) getProjectVariableSetHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManageVariableSet),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			name := vars["variableSetName"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			canUseItem, err := rbac.HasRoleOnVariableSetAndUserID(ctx, api.mustDB(), sdk.VariableSetRoleUse, u.AuthConsumerUser.AuthentifiedUserID, pKey, name)
			if err != nil {
				return err
			}

			var opts []gorpmapper.GetOptionFunc
			if canUseItem || u.Maintainer() {
				opts = append(opts, project.WithVariableSetItems)
			}
			variableSet, err := project.LoadVariableSetByName(ctx, api.mustDB(), pKey, name, opts...)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, variableSet, http.StatusOK)
		}
}

// getProjectVariableSetsHandler Retrieve all variable set for the given project
func (api *API) getProjectVariableSetsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			vss, err := project.LoadVariableSetsByProject(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, vss, http.StatusOK)
		}
}

// postProjectVariableSetHandler creates a new variable set on the given project
func (api *API) postProjectVariableSetHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManageVariableSet),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var vs sdk.ProjectVariableSet
			if err := service.UnmarshalBody(req, &vs); err != nil {
				return err
			}

			p, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			_, err = project.LoadVariableSetByName(ctx, api.mustDB(), p.Key, vs.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}

			reg, err := regexp.Compile(sdk.EntityNamePattern)
			if err != nil {
				return sdk.WithStack(err)
			}
			if !reg.MatchString(vs.Name) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "variable set name doesn't match regexp %s", sdk.EntityNamePattern)
			}

			vs.ProjectKey = p.Key
			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() //

			if err := project.InsertVariableSet(ctx, tx, &vs); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishProjectVariableSetEvent(ctx, api.Cache, sdk.EventVariableSetCreated, p.Key, vs, *u.AuthConsumerUser.AuthentifiedUser)

			return service.WriteJSON(w, vs, http.StatusOK)
		}
}

// deleteProjectVariableSetHandler delete the given variable set
func (api *API) deleteProjectVariableSetHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManageVariableSet),
		func(ctx context.Context, _ http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			varSetName := vars["variableSetName"]

			force := QueryBool(req, "force")

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			vs, err := project.LoadVariableSetByName(ctx, api.mustDB(), pKey, varSetName)
			if err != nil {
				return err
			}

			items, err := project.LoadVariableSetAllItem(ctx, api.mustDB(), vs.ID)
			if err != nil {
				return err
			}

			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			if len(items) != 0 && !force {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to remove the variable set %s in project %s. It contains %d items", vs.Name, pKey, len(items))
			}

			// If force, remove all items in the variable set
			if force {
				for _, it := range items {
					switch it.Type {
					case sdk.ProjectVariableTypeSecret:
						if err := project.DeleteVariableSetItemSecret(ctx, tx, it); err != nil {
							return err
						}
					default:
						if err := project.DeleteVariableSetItemText(ctx, tx, it); err != nil {
							return err
						}
					}
				}
			}

			if err := project.DeleteVariableSet(ctx, tx, *vs); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			for _, it := range items {
				event_v2.PublishProjectVariableSetItemEvent(ctx, api.Cache, sdk.EventVariableSetItemDeleted, pKey, vs.Name, it, *u.AuthConsumerUser.AuthentifiedUser)

			}
			event_v2.PublishProjectVariableSetEvent(ctx, api.Cache, sdk.EventVariableSetDeleted, pKey, *vs, *u.AuthConsumerUser.AuthentifiedUser)
			return nil
		}
}
