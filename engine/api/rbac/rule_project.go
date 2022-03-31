package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

var (
	LogFieldRole = log.Field("action_metadata_role")
)

func hasRoleOnProject(ctx context.Context, auth *sdk.AuthConsumer, store cache.Store, db gorp.SqlExecutor, projectKey string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := HasRoleOnProjectAndUserID(ctx, db, role, auth.AuthentifiedUser.ID, projectKey)
	if err != nil {
		return err
	}

	log.RegisterField(LogFieldRole)
	ctx = context.WithValue(ctx, LogFieldRole, role)
	log.Info(ctx, "hasRole:%t", hasRole)

	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	return nil
}

// ProjectManage return nil if the current AuthConsumer have the RoleManage on current project KEY
func ProjectManage(ctx context.Context, auth *sdk.AuthConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	return hasRoleOnProject(ctx, auth, store, db, projectKey, sdk.RoleManage)
}
