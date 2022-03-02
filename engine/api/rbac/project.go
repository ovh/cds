package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func ProjectExist(ctx context.Context, db *gorp.DbMap, vars map[string]string) error {
	log.Debug(ctx, "Rbac.ProjectExist")
	projectKey := vars["projectKey"]
	exist, err := project.Exist(db, projectKey)
	if err != nil {
		return err
	}
	log.Info(ctx, "Result")
	if !exist {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	return nil
}

func ProjectManage(ctx context.Context, db *gorp.DbMap, vars map[string]string) error {
	log.Debug(ctx, "Rbac.ProjectManage")
	projectKey := vars["projectKey"]
	// TODO Check role manage project
	log.Debug(ctx, "Checking manage project role on %s", projectKey)
	return nil
}
