package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (api *API) entityRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	entityType := vars["entityType"]
	hatch := getHatcheryConsumer(ctx)

	// hatchery can get every worker model
	if hatch != nil && entityType == sdk.EntityTypeWorkerModel {
		return nil
	}
	// Hook can get workflow
	if isHooks(ctx) {
		return nil
	}
	return hasRoleOnProject(ctx, auth, store, db, projectKey, sdk.ProjectRoleRead)
}
