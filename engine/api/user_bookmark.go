package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// postUserFavoriteHandler post favorite user for workflow or project
func (api *API) postUserFavoriteHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		params := sdk.FavoriteParams{}
		if err := service.UnmarshalBody(r, &params); err != nil {
			return err
		}

		consumer := getUserConsumer(ctx)

		p, err := project.Load(ctx, api.mustDB(), params.ProjectKey,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithFavorites(consumer.AuthConsumerUser.AuthentifiedUser.ID),
		)
		if err != nil {
			return sdk.WrapError(err, "unable to load project")
		}

		switch params.Type {
		case "workflow":
			wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *p, params.WorkflowName, workflow.LoadOptions{})
			if err != nil {
				return sdk.WrapError(err, "cannot load workflow %s/%s", params.ProjectKey, params.WorkflowName)
			}

			wf.Favorite, err = workflow.IsFavorite(api.mustDB(), wf, consumer.AuthConsumerUser.AuthentifiedUserID)
			if err != nil {
				return sdk.WrapError(err, "cannot load workflow favorite %s/%s", params.ProjectKey, params.WorkflowName)
			}
			if err := workflow.UpdateFavorite(api.mustDB(), wf.ID, consumer.AuthConsumerUser.AuthentifiedUser.ID, !wf.Favorite); err != nil {
				return sdk.WrapError(err, "cannot change workflow %s/%s favorite", params.ProjectKey, params.WorkflowName)
			}
			wf.Favorite = !wf.Favorite

			return service.WriteJSON(w, wf, http.StatusOK)
		case "project":
			if err := project.UpdateFavorite(api.mustDB(), p.ID, consumer.AuthConsumerUser.AuthentifiedUser.ID, !p.Favorite); err != nil {
				return sdk.WrapError(err, "cannot change workflow %s favorite", p.Key)
			}
			p.Favorite = !p.Favorite

			return service.WriteJSON(w, p, http.StatusOK)
		}

		return sdk.WithStack(sdk.ErrInvalidFavoriteType)
	}
}
