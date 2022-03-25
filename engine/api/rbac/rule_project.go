package rbac

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// getProjectIDFromProjectKey returns an id from a project key.
// this id is stored in cache for 20min
func getProjectIDFromProjectKey(ctx context.Context, store cache.Store, db gorp.SqlExecutor, projectKey string) (int64, error) {
	var projectID int64
	var k = cache.Key("projet", "id", "from", "key", projectKey)
	if has, _ := store.Get(k, &projectID); has {
		return projectID, nil
	}
	project, err := project.Load(ctx, db, projectKey)
	if err != nil {
		return -1, sdk.WrapError(err, "cannot load project")
	}
	if err := store.SetWithTTL(k, projectID, int(time.Duration(20*time.Minute).Seconds())); err != nil {
		log.ErrorWithStackTrace(ctx, err)
	}
	return project.ID, nil

}

func hasRoleOnProject(ctx context.Context, auth *sdk.AuthConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	projectKey := vars["projectKey"]
	log.Debug(ctx, "Checking manage project role on %s", projectKey)

	projectID, err := getProjectIDFromProjectKey(ctx, store, db, projectKey)
	if err != nil {
		return err
	}

	hasRole, err := HasRoleOnProjectIDAndUserID(ctx, db, role, auth.AuthentifiedUser.ID, projectID)
	if err != nil {
		return err
	}

	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	return nil
}

// ProjectManage return nil if the current AuthConsumer have the RoleManage on current project KEY
func ProjectManage(ctx context.Context, auth *sdk.AuthConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	return hasRoleOnProject(ctx, auth, store, db, vars, sdk.RoleManage)
}
