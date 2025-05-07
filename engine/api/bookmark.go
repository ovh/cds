package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/bookmark"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getBookmarksHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			consumer := getUserConsumer(ctx)
			data, err := bookmark.LoadAll(ctx, api.mustDB(), consumer.AuthConsumerUser.AuthentifiedUser.ID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, data, http.StatusOK)
		}
}

func (api *API) postBookmarkHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			var data sdk.Bookmark
			if err := service.UnmarshalBody(r, &data); err != nil {
				return err
			}

			c := getUserConsumer(ctx)

			switch data.Type {
			case sdk.WorkflowLegacyBookmarkType:
				splitted := strings.Split(data.ID, "/")
				if len(splitted) != 2 {
					return sdk.WithStack(sdk.ErrInvalidData)
				}
				projectKey := splitted[0]
				workflowName := splitted[1]

				perms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), []string{projectKey}, c.GetGroupIDs())
				if err != nil {
					return err
				}
				maxLevelPermission := perms.Level(projectKey)
				if maxLevelPermission < sdk.PermissionRead {
					return sdk.WithStack(sdk.ErrForbidden)
				}
				p, err := project.Load(ctx, api.mustDB(), projectKey)
				if err != nil {
					return sdk.WrapError(err, "unable to load project")
				}
				wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *p, workflowName, workflow.LoadOptions{Minimal: true})
				if err != nil {
					return sdk.WrapError(err, "cannot load workflow %s/%s", projectKey, workflowName)
				}
				if err := bookmark.InsertWorkflowLegacyFavorite(ctx, api.mustDB(), c.AuthConsumerUser.AuthentifiedUser.ID, wf.ID); err != nil {
					return sdk.WrapError(err, "cannot insert workflow legacy %s bookmark", data.ID)
				}
				return service.WriteJSON(w, sdk.Bookmark{
					Type:  sdk.WorkflowLegacyBookmarkType,
					ID:    data.ID,
					Label: workflowName,
				}, http.StatusOK)
			case sdk.ProjectBookmarkType:
				projectKey := data.ID
				perms, err := permission.LoadProjectMaxLevelPermission(ctx, api.mustDB(), []string{projectKey}, c.GetGroupIDs())
				if err != nil {
					return err
				}
				maxLevelPermission := perms.Level(projectKey)
				if maxLevelPermission < sdk.PermissionRead {
					return sdk.WithStack(sdk.ErrForbidden)
				}
				p, err := project.Load(ctx, api.mustDB(), projectKey)
				if err != nil {
					return sdk.WrapError(err, "unable to load project")
				}
				if err := bookmark.InsertProjectFavorite(ctx, api.mustDB(), c.AuthConsumerUser.AuthentifiedUserID, p.ID); err != nil {
					return sdk.WrapError(err, "cannot insert project %s bookmark", p.Key)
				}
				return service.WriteJSON(w, sdk.Bookmark{
					Type:  sdk.ProjectBookmarkType,
					ID:    data.ID,
					Label: p.Name,
				}, http.StatusOK)
			case sdk.WorkflowBookmarkType:
				splitted := strings.Split(data.ID, "/")
				if len(splitted) < 4 {
					return sdk.WithStack(sdk.ErrInvalidData)
				}
				projectKey := splitted[0]
				vcsName := splitted[1]
				repository := strings.Join(splitted[2:len(splitted)-1], "/")
				workflowName := splitted[len(splitted)-1]
				hasRole, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDBWithCtx(ctx), sdk.ProjectRoleRead, c.AuthConsumerUser.AuthentifiedUser.ID, projectKey)
				if err != nil {
					return err
				}
				if !hasRole {
					return sdk.WithStack(sdk.ErrForbidden)
				}
				vcsProject, err := api.getVCSByIdentifier(ctx, projectKey, vcsName)
				if err != nil {
					return err
				}
				repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repository)
				if err != nil {
					return err
				}
				entities, err := entity.LoadByRepository(ctx, api.mustDB(), repo.ID)
				if err != nil {
					return err
				}
				var found bool
				for i := 0; i < len(entities); i++ {
					if entities[i].Type == sdk.EntityTypeWorkflow && entities[i].Name == workflowName {
						found = true
						break
					}
				}
				if !found {
					return sdk.WithStack(sdk.ErrInvalidData)
				}
				if err := bookmark.InsertEntityFavorite(ctx, api.mustDB(), c.AuthConsumerUser.AuthentifiedUserID, repo.ID, sdk.EntityTypeWorkflow, workflowName); err != nil {
					return sdk.WrapError(err, "cannot insert workflow %s bookmark", data.ID)
				}
				return service.WriteJSON(w, sdk.Bookmark{
					Type:  sdk.WorkflowBookmarkType,
					ID:    data.ID,
					Label: workflowName,
				}, http.StatusOK)
			}

			return sdk.WithStack(sdk.ErrInvalidData)
		}
}

func (api *API) deleteBookmarkHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			c := getUserConsumer(ctx)

			vars := mux.Vars(r)
			bookmarkType := sdk.BookmarkType(vars["type"])
			bookmarkID, err := url.PathUnescape(vars["id"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}

			switch bookmarkType {
			case sdk.WorkflowLegacyBookmarkType:
				splitted := strings.Split(bookmarkID, "/")
				if len(splitted) != 2 {
					return sdk.WithStack(sdk.ErrInvalidData)
				}
				projectKey := splitted[0]
				workflowName := splitted[1]
				p, err := project.Load(ctx, api.mustDB(), projectKey)
				if err != nil {
					return sdk.WrapError(err, "unable to load project")
				}
				wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *p, workflowName, workflow.LoadOptions{Minimal: true})
				if err != nil {
					return sdk.WrapError(err, "cannot load workflow %s/%s", projectKey, workflowName)
				}
				if err := bookmark.DeleteWorkflowLegacyFavorite(ctx, api.mustDB(), c.AuthConsumerUser.AuthentifiedUser.ID, wf.ID); err != nil {
					return sdk.WrapError(err, "cannot delete workflow legacy %s bookmark", bookmarkID)
				}
				return service.WriteJSON(w, nil, http.StatusOK)
			case sdk.ProjectBookmarkType:
				p, err := project.Load(ctx, api.mustDB(), bookmarkID)
				if err != nil {
					return sdk.WrapError(err, "unable to load project")
				}
				if err := bookmark.DeleteProjectFavorite(ctx, api.mustDB(), c.AuthConsumerUser.AuthentifiedUserID, p.ID); err != nil {
					return sdk.WrapError(err, "cannot delete project %s bookmark", p.Key)
				}
				return service.WriteJSON(w, nil, http.StatusOK)
			case sdk.WorkflowBookmarkType:
				splitted := strings.Split(bookmarkID, "/")
				if len(splitted) < 4 {
					return sdk.WithStack(sdk.ErrInvalidData)
				}
				projectKey := splitted[0]
				vcsName := splitted[1]
				repository := strings.Join(splitted[2:len(splitted)-1], "/")
				workflowName := splitted[len(splitted)-1]
				vcsProject, err := api.getVCSByIdentifier(ctx, projectKey, vcsName)
				if err != nil {
					return err
				}
				repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repository)
				if err != nil {
					return err
				}
				if err := bookmark.DeleteEntityFavorite(ctx, api.mustDB(), c.AuthConsumerUser.AuthentifiedUserID, repo.ID, sdk.EntityTypeWorkflow, workflowName); err != nil {
					return sdk.WrapError(err, "cannot delete workflow %s bookmark", bookmarkID)
				}
				return service.WriteJSON(w, nil, http.StatusOK)
			}

			return sdk.WithStack(sdk.ErrInvalidData)
		}
}
