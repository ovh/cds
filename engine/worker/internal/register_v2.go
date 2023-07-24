package internal

import (
	"context"
	"errors"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// V2Register Workers need to register to main api so they can run actions
func (w *CurrentWorker) V2Register(ctx context.Context, runJobID, region string) error {
	var form sdk.WorkerRegistrationForm
	log.Info(ctx, "Registering on %s", w.cfg.APIEndpoint)

	form.Version = sdk.VERSION
	form.OS = sdk.GOOS
	form.Arch = sdk.GOARCH

	worker, err := w.ClientV2().V2WorkerRegister(context.Background(), w.cfg.APIToken, form, region, runJobID)
	if err != nil {
		return sdk.WithStack(err)
	}

	if worker.ID == "" {
		return sdk.WithStack(errors.New("worker registration failed"))
	}
	w.id = worker.ID
	return nil
}

func (w *CurrentWorker) V2Unregister(ctx context.Context, region, runJobID string) error {
	log.Info(ctx, "Unregistering worker")
	w.id = ""
	if err := w.ClientV2().V2WorkerUnregister(context.TODO(), region, runJobID); err != nil {
		return err
	}
	return nil
}
