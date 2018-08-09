package hatchery

import (
	"context"
	"fmt"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"go.opencensus.io/stats"
)

// WorkerPool returns all the worker owned by the hatchery h, registered or not on the CDS API
func WorkerPool(h Interface, status ...sdk.Status) ([]sdk.Worker, error) {
	ctx := WithTags(context.Background(), h)

	// First: call API
	registeredWorkers, err := h.CDSClient().WorkerList()
	if err != nil {
		return nil, fmt.Errorf("unable to get registered workers: %v", err)
	}

	// Then: get all worker in the orchestrator queue
	startedWorkers := h.WorkersStarted()
	if err != nil {
		return nil, fmt.Errorf("unable to get started workers: %v", err)
	}

	// Make the union of the two slices
	allWorkers := make([]sdk.Worker, 0, len(startedWorkers)+len(registeredWorkers))

	// Consider the registered worker
	for _, w := range registeredWorkers {
		var found bool
		for i := range startedWorkers {
			if startedWorkers[i] == w.Name {
				startedWorkers = append(startedWorkers[:i], startedWorkers[i+1:]...)
				found = true
				break
			}
		}
		if !found && w.Status != sdk.StatusDisabled {
			log.Error("Hatchery > WorkerPool> Worker %s (status = %s) inconsistency", w.Name, w.Status.String())
			if err := h.CDSClient().WorkerDisable(w.ID); err != nil {
				log.Error("Hatchery > WorkerPool> Unable to disable worker [%s]%s", w.ID, w.Name)
			}
			w.Status = sdk.StatusDisabled
		}
		allWorkers = append(allWorkers, w)
	}

	// And add the other worker with status pending of registering
	for _, w := range startedWorkers {
		name := w
		status := sdk.StatusWorkerPending
		if strings.HasPrefix(w, "register-") {
			name = strings.Replace(w, "register-", "", 1)
			status = sdk.StatusWorkerRegistering
		}
		allWorkers = append(allWorkers, sdk.Worker{
			Name:   name,
			Status: status,
		})
	}

	nbPerStatus := map[sdk.Status]int{}
	for _, w := range allWorkers {
		nbPerStatus[w.Status] = nbPerStatus[w.Status] + 1
	}

	measures := []stats.Measurement{
		h.Stats().PendingWorkers.M(int64(nbPerStatus[sdk.StatusWorkerPending])),
		h.Stats().RegisteringWorkers.M(int64(nbPerStatus[sdk.StatusWorkerPending])),
		h.Stats().WaitingWorkers.M(int64(nbPerStatus[sdk.StatusWaiting])),
		h.Stats().CheckingWorkers.M(int64(nbPerStatus[sdk.StatusChecking])),
		h.Stats().BuildingWorkers.M(int64(nbPerStatus[sdk.StatusBuilding])),
		h.Stats().DisabledWorkers.M(int64(nbPerStatus[sdk.StatusDisabled])),
	}
	stats.Record(ctx, measures...)

	// Filter by status
	res := make([]sdk.Worker, 0, len(allWorkers))
	if len(status) == 0 {
		res = allWorkers
	} else {
		for _, w := range allWorkers {
			for _, s := range status {
				if s == w.Status {
					res = append(res, w)
					break
				}
			}
		}
	}

	return res, nil
}
