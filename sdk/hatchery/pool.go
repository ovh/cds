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

// pendingPoolWorker represents a worker that has been started by the hatchery
// but is not yet registered on the CDS API.
type pendingPoolWorker struct {
	name   string `json:"name"`
	status string `json:"status"`
}

func (w pendingPoolWorker) GetName() string   { return w.name }
func (w pendingPoolWorker) GetStatus() string { return w.status }
func (w pendingPoolWorker) GetID() string     { return "" }

// MarshalJSON implements json.Marshaler for proper JSON serialization.
func (w pendingPoolWorker) MarshalJSON() ([]byte, error) {
	return []byte(`{"name":"` + w.name + `","status":"` + w.status + `"}`), nil
}

// WorkerPool returns all the worker owned by the hatchery h, registered or not on the CDS API
func WorkerPool(ctx context.Context, h Interface, statusFilter ...string) ([]sdk.PoolWorker, error) {
	ctx, end := telemetry.Span(ctx, "hatchery.WorkerPool")
	defer end()

	ctx = telemetry.ContextWithTag(ctx,
		telemetry.TagServiceName, h.Name(),
		telemetry.TagServiceType, h.Type(),
	)

	// First: call API to get registered workers (v1 and v2)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var registeredWorkers []sdk.PoolWorker

	if h.CDSClient() != nil {
		workers, err := h.CDSClient().WorkerList(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to get registered v1 workers: %v", err)
		}
		for i := range workers {
			registeredWorkers = append(registeredWorkers, &workers[i])
		}
	}

	if h.CDSClientV2() != nil {
		v2Workers, err := h.CDSClientV2().V2WorkerList(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to get registered v2 workers: %v", err)
		}
		for i := range v2Workers {
			registeredWorkers = append(registeredWorkers, &v2Workers[i])
		}
	}

	// Then: get all workers in the orchestrator queue
	startedWorkers, err := h.WorkersStarted(ctx)
	if err != nil {
		return nil, err
	}

	// Make the union of the two slices
	allWorkers := make([]sdk.PoolWorker, 0, len(startedWorkers)+len(registeredWorkers))

	// Consider the registered workers
	for _, w := range registeredWorkers {
		var found bool
		for i := range startedWorkers {
			if startedWorkers[i] == w.GetName() {
				startedWorkers = append(startedWorkers[:i], startedWorkers[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			if w.GetStatus() != sdk.StatusDisabled {
				// Worker is registered on API but not found in the orchestrator
				switch v := w.(type) {
				case *sdk.Worker:
					log.Error(ctx, "Hatchery > WorkerPool> Worker %s (status = %s) inconsistency", v.Name, v.Status)
					if err := h.CDSClient().WorkerDisable(ctx, v.ID); err != nil {
						ctx = sdk.ContextWithStacktrace(ctx, err)
						log.Error(ctx, "Hatchery > WorkerPool> Unable to disable worker [%s]%s: %v", v.ID, v.Name, err)
					}
					v.Status = sdk.StatusDisabled
				case *sdk.V2Worker:
					log.Warn(ctx, "Hatchery > WorkerPool> V2 Worker %s (status = %s) inconsistency - no V2WorkerDisable available", v.Name, v.Status)
				}
			}
		}
		allWorkers = append(allWorkers, w)
	}

	// And add the other workers with status pending or registering
	for _, w := range startedWorkers {
		var found bool
		for _, wr := range registeredWorkers {
			if wr.GetName() == w {
				found = true
				break
			}
		}
		if found {
			continue // worker is registered
		}

		status := sdk.StatusWorkerPending
		if strings.HasPrefix(w, "register-") {
			status = sdk.StatusWorkerRegistering
		}

		allWorkers = append(allWorkers, pendingPoolWorker{
			name:   w,
			status: status,
		})
	}

	nbPerStatus := map[string]int{}
	for _, w := range allWorkers {
		nbPerStatus[w.GetStatus()] = nbPerStatus[w.GetStatus()] + 1
	}

	measures := []stats.Measurement{
		GetMetrics().PendingWorkers.M(int64(nbPerStatus[sdk.StatusWorkerPending])),
		GetMetrics().RegisteringWorkers.M(int64(nbPerStatus[sdk.StatusWorkerRegistering])),
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
	res := make([]sdk.PoolWorker, 0, len(allWorkers))
	for _, w := range allWorkers {
		for _, s := range statusFilter {
			if s == w.GetStatus() {
				res = append(res, w)
				break
			}
		}
	}

	return res, nil
}
