package api

import (
	"context"
)

func (api *API) workerModelRead(ctx context.Context, vars map[string]string) error {
	if getHatcheryConsumer(ctx) != nil {
		return nil
	}
	return api.projectRead(ctx, vars)
}
