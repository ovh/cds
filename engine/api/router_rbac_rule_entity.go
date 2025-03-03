package api

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (api *API) entityRead(ctx context.Context, vars map[string]string) error {
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
	return api.hasRoleOnProject(ctx, vars, sdk.ProjectRoleRead)
}
