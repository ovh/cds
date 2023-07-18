package internal

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func V2StartWorker(ctx context.Context, w *CurrentWorker, runJobID string, region string) (mainError error) {
	ctx = context.WithValue(ctx, log.Field("permJobID"), runJobID)

	log.Info(ctx, "Starting worker %s on job %s", w.Name(), runJobID)

	if runJobID == "0" || runJobID == "" {
		return errors.Errorf("startWorker: bookedJobID is mandatory. val: %s", runJobID)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	httpServerCtx, stopHTTPServer := context.WithCancel(ctx)
	defer stopHTTPServer()
	if err := w.Serve(httpServerCtx); err != nil {
		return err
	}

	//Register
	if err := w.V2Register(ctx, runJobID, region); err != nil {
		return sdk.WrapError(err, "unable to register to CDS")
	}

	//Register every 10 seconds
	refreshTick := time.NewTicker(30 * time.Second)

	// start queue polling
	errsChan := make(chan error, 1)

	//Definition of the function which must be called to stop the worker
	var endFunc = func() {
		log.Info(ctx, "Stopping worker %s", w.Name())
		if err := w.V2Unregister(ctx, runJobID, region); err != nil {
			log.Error(ctx, "Unable to unregister: %v", err)
			mainError = err
		}
		refreshTick.Stop()
		cancel()
		stopHTTPServer()

		if err := ctx.Err(); err != nil {
			log.Warn(ctx, "Exiting worker: %v", err)
		} else {
			log.Warn(ctx, "Exiting worker")
		}
	}

	// Errors check loops
	go func() {
		for err := range errsChan {
			log.Error(ctx, "An error has occured: %v", err)
			if strings.Contains(err.Error(), "not authenticated") {
				endFunc()
				return
			}
		}
	}()

	// Register (heartbeat loop)
	go func() {
		var nbErrors int
		for {
			select {
			case <-ctx.Done():
				return
			case <-refreshTick.C:
				if err := w.ClientV2().V2WorkerRefresh(ctx, region, runJobID); err != nil {
					log.Error(ctx, "Heartbeat failed: %v", err)
					nbErrors++
					if nbErrors == 5 {
						errsChan <- err
					}
				}
				nbErrors = 0
			}
		}
	}()

	//Take the job
	log.Debug(ctx, "checkQueue> Try take the job %d", runJobID)
	if err := w.V2Take(ctx, region, runJobID); err != nil {
		log.Info(ctx, "Unable to run this job  %s. Take info: %v", runJobID, err)
		errsChan <- err
	}

	// Unregister from engine
	log.Info(ctx, "Job is done. Unregistering...")
	endFunc()
	return nil

}
