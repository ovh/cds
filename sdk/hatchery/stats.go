package hatchery

import (
	"context"
	"sync"

	"github.com/rockbears/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/sdk/telemetry"
)

var (
	onceMetrics sync.Once
	metrics     Metrics
)

// GetMetrics returns the metric stats measures
func GetMetrics() *Metrics {
	return &metrics
}

func initMetrics(ctx context.Context) error {
	log.Debug(ctx, "hatchery> initializing metrics")
	var err error
	onceMetrics.Do(func() {
		metrics.Jobs = stats.Int64("cds/jobs", "number of analyzed jobs", stats.UnitDimensionless)
		metrics.JobsWebsocket = stats.Int64("cds/jobs_websocket", "number of analyzed jobs from SSE", stats.UnitDimensionless)
		metrics.SpawnedWorkers = stats.Int64("cds/spawned_workers", "number of spawned workers", stats.UnitDimensionless)
		metrics.PendingWorkers = stats.Int64("cds/pending_workers", "number of pending workers", stats.UnitDimensionless)
		metrics.RegisteringWorkers = stats.Int64("cds/registering_workers", "number of registering workers", stats.UnitDimensionless)
		metrics.WaitingWorkers = stats.Int64("cds/waiting_workers", "number of waiting workers", stats.UnitDimensionless)
		metrics.CheckingWorkers = stats.Int64("cds/checking_workers", "number of checking workers", stats.UnitDimensionless)
		metrics.BuildingWorkers = stats.Int64("cds/building_workers", "number of building workers", stats.UnitDimensionless)
		metrics.DisabledWorkers = stats.Int64("cds/disabled_workers", "number of disabled workers", stats.UnitDimensionless)

		tags := []tag.Key{telemetry.MustNewKey(telemetry.TagServiceType), telemetry.MustNewKey(telemetry.TagServiceName)}
		err = telemetry.RegisterView(ctx,
			telemetry.NewViewCount("cds/hatchery/jobs_count", metrics.Jobs, tags),
			telemetry.NewViewCount("cds/hatchery/jobs_websocket_count", metrics.JobsWebsocket, tags),
			telemetry.NewViewCount("cds/hatchery/spawned_worker_count", metrics.SpawnedWorkers, tags),
			telemetry.NewViewLast("cds/hatchery/pending_workers", metrics.PendingWorkers, tags),
			telemetry.NewViewLast("cds/hatchery/registering_workers", metrics.RegisteringWorkers, tags),
			telemetry.NewViewLast("cds/hatchery/waiting_workers", metrics.WaitingWorkers, tags),
			telemetry.NewViewLast("cds/hatchery/checking_workers", metrics.CheckingWorkers, tags),
			telemetry.NewViewLast("cds/hatchery/building_workers", metrics.BuildingWorkers, tags),
			telemetry.NewViewLast("cds/hatchery/disabled_workers", metrics.DisabledWorkers, tags),
		)
	})
	return err
}
