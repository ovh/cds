package api

import (
	"context"
	"net/url"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func hasRoleOnWorkflow(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, projectKey string, vcs, repo, workflowName string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasRoleOnWorkflowAndUserID(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID, projectKey, vcs, repo, workflowName)
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
func (api *API) workflowTrigger(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	workflowName := vars["workflowName"]
	workflowRunID := vars["workflowRunID"]

	var vcsName, repoName string
	if workflowRunID != "" {
		run, err := workflow_v2.LoadRunByID(ctx, db, workflowRunID)
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
	return hasRoleOnWorkflow(ctx, auth, store, db, projectKey, vcsName, repoName, workflowName, sdk.WorkflowRoleTrigger)
}
