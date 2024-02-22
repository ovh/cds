package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func hasRoleOnProject(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, projectKey string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasRoleOnProjectAndUserID(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID, projectKey)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.RbacRole, role)
	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	return nil
}

// ProjectManage return nil if the current AuthUserConsumer have the ProjectRoleManage on current project KEY
func (api *API) projectManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	return hasRoleOnProject(ctx, auth, store, db, projectKey, sdk.ProjectRoleManage)
}

// projectManageNotification return nil if the current AuthUserConsumer have the role ProjectRoleManageNotification on current project KEY
func (api *API) projectManageNotification(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	return hasRoleOnProject(ctx, auth, store, db, projectKey, sdk.ProjectRoleManageNotification)
}

// ProjectRead return nil if the current AuthUserConsumer have the ProjectRoleRead on current project KEY
func (api *API) projectRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	entityType := vars["entityType"]
	hatch := getHatcheryConsumer(ctx)

	// hatchery can get every worker model
	if hatch != nil && entityType == sdk.EntityTypeWorkerModel {
		return nil
	}

	return hasRoleOnProject(ctx, auth, store, db, projectKey, sdk.ProjectRoleRead)
}

func (api *API) analysisRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	if isHooks(ctx) {
		return nil
	}
	return api.projectRead(ctx, auth, store, db, vars)
}

func (api *API) triggerAnalysis(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	if isHooks(ctx) {
		return nil
	}
	return api.projectManage(ctx, auth, store, db, vars)
}
