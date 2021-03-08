package hatchery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opencensus.io/stats"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

// WorkerPool returns all the worker owned by the hatchery h, registered or not on the CDS API
func WorkerPool(ctx context.Context, h Interface, statusFilter ...string) ([]sdk.Worker, error) {
	ctx = telemetry.ContextWithTag(ctx,
		telemetry.TagServiceName, h.Name(),
		telemetry.TagServiceType, h.Type(),
	)

	// First: call API
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	registeredWorkers, err := h.CDSClient().WorkerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get registered workers: %v", err)
	}

	// Then: get all workers in the orchestrator queue
	startedWorkers := h.WorkersStarted(ctx)

	// Make the union of the two slices
	allWorkers := make([]sdk.Worker, 0, len(startedWorkers)+len(registeredWorkers))

	// Consider the registered worker
	for k, w := range registeredWorkers {
		var found bool
		for i := range startedWorkers {
			if startedWorkers[i] == w.Name {
				startedWorkers = append(startedWorkers[:i], startedWorkers[i+1:]...)
				found = true

				if strings.HasPrefix(w.Name, "register-") {
					registeredWorkers[k].Status = sdk.StatusWorkerRegistering
				}

				break
			}
		}
		if !found && w.Status != sdk.StatusDisabled {
			log.Error(ctx, "Hatchery > WorkerPool> Worker %s (status = %s) inconsistency", w.Name, w.Status)
			if err := h.CDSClient().WorkerDisable(ctx, w.ID); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "Hatchery > WorkerPool> Unable to disable worker [%s]%s: %v", w.ID, w.Name, err)
			}
			registeredWorkers[k].Status = sdk.StatusDisabled
		}
		allWorkers = append(allWorkers, registeredWorkers[k])
	}

	// And add the other worker with status pending of registering
	for _, w := range startedWorkers {
		name := w
		var status string

		var found bool
		for _, wr := range registeredWorkers {
			if wr.Name == name {
				found = true
				break
			}
		}
		if found {
			continue // worker is registered
		}

		if strings.HasPrefix(w, "register-") {
			status = sdk.StatusWorkerRegistering
		}

		if status == "" {
			status = sdk.StatusWorkerPending
		}
		allWorkers = append(allWorkers, sdk.Worker{
			Name:   name,
			Status: status,
		})
	}

	nbPerStatus := map[string]int{}
	for _, w := range allWorkers {
		nbPerStatus[w.Status] = nbPerStatus[w.Status] + 1
	}

	measures := []stats.Measurement{
		GetMetrics().PendingWorkers.M(int64(nbPerStatus[sdk.StatusWorkerPending])),
		GetMetrics().RegisteringWorkers.M(int64(nbPerStatus[sdk.StatusWorkerPending])),
		GetMetrics().WaitingWorkers.M(int64(nbPerStatus[sdk.StatusWaiting])),
		GetMetrics().CheckingWorkers.M(int64(nbPerStatus[sdk.StatusChecking])),
		GetMetrics().BuildingWorkers.M(int64(nbPerStatus[sdk.StatusBuilding])),
		GetMetrics().DisabledWorkers.M(int64(nbPerStatus[sdk.StatusDisabled])),
	}
	stats.Record(ctx, measures...)

	// no filter on status, returns the workers list as is.
	if len(statusFilter) == 0 {
		return allWorkers, nil
	}

	// return workers list filtered by status
	res := make([]sdk.Worker, 0, len(allWorkers))
	for _, w := range allWorkers {
		for _, s := range statusFilter {
			if s == w.Status {
				res = append(res, w)
				break
			}
		}
	}

	return res, nil
}
