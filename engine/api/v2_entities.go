package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getEntityRefFromQueryParams(ctx context.Context, req *http.Request, projKey string, vcsName string, repoName string) (string, string, error) {
	branch := QueryString(req, "branch")
	tag := QueryString(req, "tag")
	ref := QueryString(req, "ref")
	commit := QueryString(req, "commit")

	if commit == "" {
		commit = "HEAD"
	}

	if ref != "" && (strings.HasPrefix(ref, sdk.GitRefBranchPrefix) || strings.HasPrefix(ref, sdk.GitRefTagPrefix)) {
		return ref, commit, nil
	}

	if branch != "" && tag != "" {
		return "", commit, sdk.NewErrorFrom(sdk.ErrWrongRequest, "Query params branch and tag cannot be used together")
	}

	if branch == "" {
		vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, projKey, vcsName)
		if err != nil {
			return "", commit, err
		}
		defaultBranch, err := vcsClient.Branch(ctx, repoName, sdk.VCSBranchFilters{Default: true})
		if err != nil {
			return "", "", err
		}
		ref = defaultBranch.ID
		commit = "HEAD"
	} else if tag != "" {
		ref = sdk.GitRefTagPrefix + tag
	} else {
		ref = sdk.GitRefBranchPrefix + branch
	}
	return ref, commit, nil

}

func (api *API) postEntityCheckHandler() ([]service.RbacChecker, service.Handler) {
	return nil, func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
		vars := mux.Vars(req)
		entityType := vars["entityType"]

		response := sdk.EntityCheckResponse{
			Messages: make([]string, 0),
		}
		switch entityType {
		case sdk.EntityTypeWorkerModel:
			var wm sdk.V2WorkerModel
			err := service.UnmarshalRequest(ctx, req, &wm)
			if err != nil {
				response.Messages = append(response.Messages, fmt.Sprintf("%q", err))
			}
			if err == nil {
				errs := wm.Lint()
				for _, err := range errs {
					response.Messages = append(response.Messages, err.Error())
				}
			}
		case sdk.EntityTypeAction:
			var a sdk.V2Action
			err := service.UnmarshalRequest(ctx, req, &a)
			if err != nil {
				response.Messages = append(response.Messages, fmt.Sprintf("%q", err))
			}
			if err == nil {
				errs := a.Lint()
				for _, err := range errs {
					response.Messages = append(response.Messages, err.Error())
				}
			}
		case sdk.EntityTypeWorkflow:
			var w sdk.V2Workflow
			err := service.UnmarshalRequest(ctx, req, &w)
			if err != nil {
				response.Messages = append(response.Messages, fmt.Sprintf("%q", err))
			}
			if err == nil {
				errs := w.Lint()
				for _, err := range errs {
					response.Messages = append(response.Messages, err.Error())
				}
			}
		case sdk.EntityTypeWorkflowTemplate:
			var wt sdk.V2WorkflowTemplate
			err := service.UnmarshalRequest(ctx, req, &wt)
			if err != nil {
				response.Messages = append(response.Messages, fmt.Sprintf("%q", err))
			}
			if err == nil {
				errs := wt.Lint()
				for _, err := range errs {
					response.Messages = append(response.Messages, err.Error())
				}
			}
		}
		return service.WriteJSON(w, response, http.StatusOK)
	}
}

func (api *API) getEntitiesHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			entityType := vars["entityType"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.ErrUnauthorized
			}

			var entities []sdk.EntityFullName
			if isAdmin(ctx) {
				var err error
				entities, err = entity.UnsafeLoadAllByType(ctx, api.mustDB(), entityType)
				if err != nil {
					return err
				}
			} else {
				projectKeys, err := rbac.LoadAllProjectKeysAllowed(ctx, api.mustDB(), sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
				if err != nil {
					return err
				}
				entities, err = entity.UnsafeLoadAllByTypeAndProjectKeys(ctx, api.mustDB(), entityType, projectKeys)
				if err != nil {
					return err
				}
			}

			return service.WriteJSON(w, entities, http.StatusOK)
		}
}

// getProjectEntitiesHandler
func (api *API) getProjectEntitiesHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			ref, commit, err := api.getEntityRefFromQueryParams(ctx, req, pKey, vcsProject.Name, repo.Name)
			if err != nil {
				return err
			}

			entities, err := entity.LoadByRepositoryAndRefAndCommit(ctx, api.mustDB(), repo.ID, ref, commit)
			if err != nil {
				return err
			}
			result := make([]sdk.ShortEntity, 0, len(entities))
			for _, e := range entities {
				result = append(result, sdk.ShortEntity{
					ID:   e.ID,
					Name: e.Name,
					Type: e.Type,
					Ref:  e.Ref,
				})
			}
			return service.WriteJSON(w, result, http.StatusOK)
		}
}

func (api *API) getProjectEntityHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.entityRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			entityType := vars["entityType"]
			entityName := vars["entityName"]

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			ref, commit, err := api.getEntityRefFromQueryParams(ctx, req, pKey, vcsProject.Name, repo.Name)
			if err != nil {
				return err
			}

			entity, err := entity.LoadByRefTypeNameCommit(ctx, api.mustDB(), repo.ID, ref, entityType, entityName, commit)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, entity, http.StatusOK)
		}
}
