package api

import (
	"context"
	"net/url"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (api *API) hasRoleOnWorkflow(ctx context.Context, vars map[string]string, role string) error {
	auth := getUserConsumer(ctx)
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	projectKey := vars["projectKey"]
	workflowName := vars["workflowName"]
	workflowRunID := vars["workflowRunID"]

	var vcsName, repoName string
	if workflowRunID != "" {
		run, err := workflow_v2.LoadRunByID(ctx, api.mustDBWithCtx(ctx), workflowRunID)
		if err != nil {
			return err
		}
		vcsName = run.Contexts.Git.Server
		repoName = run.Contexts.Git.Repository
	} else {
		// Retrieve VCSName
		vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		vcsProject, err := api.getVCSByIdentifier(ctx, projectKey, vcsIdentifier)
		if err != nil {
			return err
		}
		vcsName = vcsProject.Name

		// Retrieve Repo name
		repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		if sdk.IsValidUUID(repositoryIdentifier) {
			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}
			repoName = repo.Name
		} else {
			repoName = repositoryIdentifier
		}
	}

	hasRole, err := rbac.HasRoleOnWorkflowAndUserID(ctx, api.mustDBWithCtx(ctx), role, auth.AuthConsumerUser.AuthentifiedUser.ID, projectKey, vcsName, repoName, workflowName)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.RbacRole, role)
	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	return nil
}

// workflowTrigger return nil if the current AuthUserConsumer have the WorkflowRoleTrigger on current workflow
func (api *API) workflowTrigger(ctx context.Context, vars map[string]string) error {
	return api.hasRoleOnWorkflow(ctx, vars, sdk.WorkflowRoleTrigger)
}
