package internal

import (
	"context"
	"errors"
	"os"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// Workers need to register to main api so they can run actions
func (w *CurrentWorker) Register(ctx context.Context) error {
	var form sdk.WorkerRegistrationForm
	log.Info(ctx, "Registering on %s", w.cfg.APIEndpoint)

	requirements, errR := w.client.Requirements()
	if errR != nil {
		log.Warn(ctx, "register> unable to get requirements: %v", errR)
		return errR
	}

	log.Debug(ctx, "Checking %d requirements for current PATH: %s", len(requirements), os.Getenv("PATH"))
	form.BinaryCapabilities = LoopPath(w, requirements)
	form.Version = sdk.VERSION
	form.OS = sdk.GOOS
	form.Arch = sdk.GOARCH

	worker, uptodate, err := w.client.WorkerRegister(context.Background(), w.cfg.APIToken, form)
	if err != nil {
		return sdk.WithStack(err)
	}

	if worker.ID == "" {
		return sdk.WithStack(errors.New("worker registration failed"))
	}

	w.id = worker.ID

	if worker.ModelID != nil {
		models, err := w.client.WorkerModelList(nil)
		if err != nil {
			return sdk.WrapError(err, "unable to get worker model list")
		}

		for _, m := range models {
			if m.ID == *worker.ModelID {
				w.model = m
				break
			}
		}

		if w.model.ID == 0 {
			return sdk.WithStack(errors.New("worker model not found"))
		}
	}

	if !uptodate {
		log.Warn(ctx, "-=-=-=-=- Please update your worker binary - Worker Version %s %s %s -=-=-=-=-", sdk.VERSION, sdk.GOOS, sdk.GOARCH)
	}

	return nil
}

func (w *CurrentWorker) Unregister(ctx context.Context) error {
	log.Info(ctx, "Unregistering worker")
	w.id = ""
	if err := w.Client().WorkerUnregister(context.TODO()); err != nil {
		return err
	}
	return nil
}

// LoopPath returns the list of available binaries in path
func LoopPath(w *CurrentWorker, reqs []sdk.Requirement) []string {
	binaries := []string{}
	for _, req := range reqs {
		if req.Type == sdk.BinaryRequirement {
			if b, _ := checkBinaryRequirement(w, req); b {
				binaries = append(binaries, req.Value)
			}
		}
	}
	return binaries
}
