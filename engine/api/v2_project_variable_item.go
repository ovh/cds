package api

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// getProjectVariableSetItemHandler Retrieve the given item in the variable set
func (api *API) getProjectVariableSetItemHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.variableSetItemRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vsName := vars["variableSetName"]
			itemName := vars["itemName"]

			vs, err := project.LoadVariableSetByName(ctx, api.mustDB(), pKey, vsName)
			if err != nil {
				return err
			}

			item, err := project.LoadVariableSetItem(ctx, api.mustDB(), vs.ID, itemName)
			if err != nil {
				return err
			}

			return service.WriteJSON(w, item, http.StatusOK)
		}
}

// postProjectVariableSetItemHandler creates a new item on the given variable set
func (api *API) postProjectVariableSetItemHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.variableSetItemManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vsName := vars["variableSetName"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var item sdk.ProjectVariableSetItem
			if err := service.UnmarshalBody(req, &item); err != nil {
				return err
			}

			itemPattern, err := regexp.Compile(sdk.ProjectVariableSetItemNamePattern)
			if err != nil {
				return sdk.WrapError(err, "unable to compile regexp %s", sdk.ProjectVariableSetItemNamePattern)
			}
			if !itemPattern.MatchString(item.Name) {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "name %s doesn't match %s", item.Name, sdk.ProjectVariableSetItemNamePattern)
			}

			vs, err := project.LoadVariableSetByName(ctx, api.mustDB(), pKey, vsName, project.WithVariableSetItems)
			if err != nil {
				return err
			}

			for _, it := range vs.Items {
				if it.Name == item.Name {
					return sdk.NewErrorFrom(sdk.ErrConflictData, "variable set item already exists")
				}
			}

			reg, err := regexp.Compile(sdk.EntityNamePattern)
			if err != nil {
				return sdk.WithStack(err)
			}
			if !reg.MatchString(vs.Name) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "variable set name doesn't match regexp %s", sdk.EntityNamePattern)
			}

			item.ProjectVariableSetID = vs.ID
			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() //

			switch item.Type {
			case sdk.ProjectVariableTypeSecret:
				if err := project.InsertVariableSetItemSecret(ctx, tx, &item); err != nil {
					return err
				}
			default:
				if err := project.InsertVariableSetItemText(ctx, tx, &item); err != nil {
					return err
				}
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishProjectVariableSetItemEvent(ctx, api.Cache, sdk.EventVariableSetItemCreated, pKey, vs.Name, item, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteJSON(w, vs, http.StatusOK)
		}
}

// putProjectVariableSetItemHandler updates an item on the given variable set
func (api *API) putProjectVariableSetItemHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.variableSetItemManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vsName := vars["variableSetName"]
			itemName := vars["itemName"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var item sdk.ProjectVariableSetItem
			if err := service.UnmarshalBody(req, &item); err != nil {
				return err
			}

			if item.Name != itemName {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "item name doesn't match")
			}

			vs, err := project.LoadVariableSetByName(ctx, api.mustDB(), pKey, vsName)
			if err != nil {
				return err
			}

			itemDB, err := project.LoadVariableSetItem(ctx, api.mustDB(), vs.ID, item.Name)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}

			item.ProjectVariableSetID = vs.ID
			item.ID = itemDB.ID
			item.Type = itemDB.Type
			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() //

			switch item.Type {
			case sdk.ProjectVariableTypeSecret:
				if err := project.UpdateVariableSetItemSecret(ctx, tx, &item); err != nil {
					return err
				}
			default:
				if err := project.UpdateVariableSetItemText(ctx, tx, &item); err != nil {
					return err
				}
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishProjectVariableSetItemEvent(ctx, api.Cache, sdk.EventVariableSetItemUpdated, pKey, vs.Name, item, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteJSON(w, vs, http.StatusOK)
		}
}

// deleteProjectVariableSetItemHandler delete the given variable set item
func (api *API) deleteProjectVariableSetItemHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.variableSetItemManage),
		func(ctx context.Context, _ http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			varSetName := vars["variableSetName"]
			itemName := vars["itemName"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			vs, err := project.LoadVariableSetByName(ctx, api.mustDB(), pKey, varSetName)
			if err != nil {
				return err
			}

			item, err := project.LoadVariableSetItem(ctx, api.mustDB(), vs.ID, itemName)
			if err != nil {
				return err
			}

			tx, err := api.mustDBWithCtx(ctx).Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			switch item.Type {
			case sdk.ProjectVariableTypeSecret:
				if err := project.DeleteVariableSetItemSecret(ctx, tx, *item); err != nil {
					return err
				}
			default:
				if err := project.DeleteVariableSetItemText(ctx, tx, *item); err != nil {
					return err
				}
			}

			if err := tx.Commit(); err != nil {
				return err
			}
			event_v2.PublishProjectVariableSetItemEvent(ctx, api.Cache, sdk.EventVariableSetItemDeleted, pKey, vs.Name, *item, *u.AuthConsumerUser.AuthentifiedUser)
			return nil
		}
}
