package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func hasRoleOnWorkflow(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, projectKey string, workflowName string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasRoleOnWorkflowAndUserID(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID, projectKey, workflowName)
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

	if workflowRunID != "" {
		run, err := workflow_v2.LoadRunByID(ctx, db, workflowRunID)
		if err != nil {
			return err
		}
		workflowName = fmt.Sprintf("%s/%s/%s", run.Contexts.Git.Server, run.Contexts.Git.Repository, run.WorkflowName)
	} else {
		var vcsName, repoName string

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

		workflowName = fmt.Sprintf("%s/%s/%s", vcsName, repoName, workflowName)
	}
	return hasRoleOnWorkflow(ctx, auth, store, db, projectKey, workflowName, sdk.WorkflowRoleTrigger)
}
