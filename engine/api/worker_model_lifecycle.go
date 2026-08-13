package api

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/workermodel"
)

// manageWorkerModelsLifecycle disables the deprecated worker models that reached their end of life
// date.
func (api *API) manageWorkerModelsLifecycle(ctx context.Context, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting manageWorkerModelsLifecycle: %v", ctx.Err())
			}
			return
		case <-ticker.C:
			if err := api.disableExpiredWorkerModels(ctx); err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
		}
	}
}

// disableExpiredWorkerModels disables the deprecated worker models whose end of life date is in the
// past. Once disabled a model is no longer served to the hatcheries, so it cannot be used anymore.
func (api *API) disableExpiredWorkerModels(ctx context.Context) error {
	now := time.Now()

	models, err := workermodel.LoadAllPastEOL(ctx, api.mustDB(), now, workermodel.LoadOptions.WithGroup)
	if err != nil {
		return err
	}

	for i := range models {
		disabled, err := workermodel.DisableExpired(ctx, api.mustDB(), models[i].ID, now)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}
		if disabled {
			log.Info(ctx, "worker model %s disabled: end of life date reached", models[i].Path())
		}
	}

	return nil
}
