package api

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// postAdminCopyProjectVariableSetHandler copies a variable set and all its items into a target project, optionally under a new name;
// items get new IDs so secrets are decrypted then re-encrypted, and the source variable set is left untouched.
func (api *API) postAdminCopyProjectVariableSetHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		pKey := vars["projectKey"]
		vsName := vars["variableSetName"]

		u := getUserConsumer(ctx)
		if u == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		var copyRequest sdk.CopyVariableSetToProject
		if err := service.UnmarshalBody(r, &copyRequest); err != nil {
			return err
		}
		if copyRequest.TargetProjectKey == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing target project key")
		}
		targetName := copyRequest.NewName
		if targetName == "" {
			targetName = vsName
		}
		reg, err := regexp.Compile(sdk.EntityNamePattern)
		if err != nil {
			return sdk.WithStack(err)
		}
		if !reg.MatchString(targetName) {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "variable set name doesn't match regexp %s", sdk.EntityNamePattern)
		}

		targetProj, err := project.Load(ctx, api.mustDB(), copyRequest.TargetProjectKey)
		if err != nil {
			return err
		}

		srcVS, err := project.LoadVariableSetByName(ctx, api.mustDB(), pKey, vsName)
		if err != nil {
			return err
		}
		srcItems, err := project.LoadVariableSetAllItem(ctx, api.mustDB(), srcVS.ID, gorpmapper.GetAllOptions.WithDecryption)
		if err != nil {
			return err
		}

		// Never merge into an existing variable set; the unique index still guards against a concurrent creation.
		_, err = project.LoadVariableSetByName(ctx, api.mustDB(), targetProj.Key, targetName)
		if err == nil {
			return sdk.NewErrorFrom(sdk.ErrConflictData, "variable set %s already exists in project %s", targetName, targetProj.Key)
		}
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}

		tx, err := api.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() //nolint

		newVS := sdk.ProjectVariableSet{
			ProjectKey: targetProj.Key,
			Name:       targetName,
		}
		if err := project.InsertVariableSet(ctx, tx, &newVS); err != nil {
			return err
		}

		newItems := make([]sdk.ProjectVariableSetItem, 0, len(srcItems))
		for _, srcItem := range srcItems {
			it := sdk.ProjectVariableSetItem{
				ProjectVariableSetID: newVS.ID,
				Name:                 srcItem.Name,
				Type:                 srcItem.Type,
				Value:                srcItem.Value,
			}
			switch it.Type {
			case sdk.ProjectVariableTypeSecret:
				if err := project.InsertVariableSetItemSecret(ctx, tx, &it); err != nil {
					return err
				}
				it.Value = sdk.PasswordPlaceholder
			default:
				if err := project.InsertVariableSetItemText(ctx, tx, &it); err != nil {
					return err
				}
			}
			newItems = append(newItems, it)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event_v2.PublishProjectVariableSetEvent(ctx, api.Cache, sdk.EventVariableSetCreated, targetProj.Key, newVS, *u.AuthConsumerUser.AuthentifiedUser)
		for _, it := range newItems {
			event_v2.PublishProjectVariableSetItemEvent(ctx, api.Cache, sdk.EventVariableSetItemCreated, targetProj.Key, newVS.Name, it, *u.AuthConsumerUser.AuthentifiedUser)
		}
		log.Info(ctx, "variable set %s/%s copied to %s/%s with %d items by %s", pKey, vsName, targetProj.Key, newVS.Name, len(newItems), u.AuthConsumerUser.AuthentifiedUser.Username)

		newVS.Items = newItems
		return service.WriteJSON(w, newVS, http.StatusOK)
	}
}
