package hatchery

import (
	"context"
	"sync"

	"github.com/rockbears/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

var (
	onceMetrics sync.Once
	metrics     sdk.HatcheryMetrics
)

// GetMetrics returns the metric stats measures
func GetMetrics() *sdk.HatcheryMetrics {
	return &metrics
}

func InitMetrics(ctx context.Context) error {
	log.Debug(ctx, "hatchery> initializing metrics")
	var err error
	onceMetrics.Do(func() {
		metrics.Jobs = stats.Int64("cds/jobs", "number of analyzed jobs", stats.UnitDimensionless)
		metrics.JobsWebsocket = stats.Int64("cds/jobs_websocket", "number of analyzed jobs from SSE", stats.UnitDimensionless)
		metrics.JobsProcessed = stats.Int64("cds/jobs_processed", "number of process jobs in main routine", stats.UnitDimensionless)
		metrics.SpawnedWorkers = stats.Int64("cds/spawned_workers", "number of spawned workers", stats.UnitDimensionless)
		metrics.SpawningWorkersErrors = stats.Int64("cds/spawning_workers_errors", "number of error in spawning workers", stats.UnitDimensionless)
		metrics.SpawningWorkers = stats.Int64("cds/spawning_workers", "number of spawning workers", stats.UnitDimensionless)
		metrics.JobReceivedInQueuePollingWSv1 = stats.Int64("cds/job_received_in_queue_polling_ws_v1", "number of job received in queue polling v1 ws", stats.UnitDimensionless)
		metrics.JobReceivedInQueuePollingWSv2 = stats.Int64("cds/job_received_in_queue_polling_ws_v2", "number of job received in queue polling v2 ws", stats.UnitDimensionless)
		metrics.ChanV1JobAdd = stats.Int64("cds/chan_v1_job_add", "number of add into chan jobs v1", stats.UnitDimensionless)
		metrics.ChanV2JobAdd = stats.Int64("cds/chan_v2_job_add", "number of add into chan jobs v2", stats.UnitDimensionless)
		metrics.ChanWorkerStarterPop = stats.Int64("cds/chan_worker_starter_job_pop", "number of pop from chan in a worker starter func", stats.UnitDimensionless)
		metrics.WorkerStarterRunning = stats.Int64("cds/worker_starter_running", "number current running worker starter", stats.UnitDimensionless)
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
			telemetry.NewViewCount("cds/hatchery/jobs_processed_count", metrics.JobsProcessed, tags),
			telemetry.NewViewCount("cds/hatchery/spawned_worker_count", metrics.SpawnedWorkers, tags),
			telemetry.NewViewCount("cds/hatchery/spawning_worker_count", metrics.SpawningWorkers, tags),
			telemetry.NewViewCount("cds/hatchery/spawning_worker_errors_count", metrics.SpawningWorkersErrors, tags),
			telemetry.NewViewCount("cds/hatchery/chan_v1_job_add_count", metrics.ChanV1JobAdd, tags),
			telemetry.NewViewCount("cds/hatchery/chan_v2_job_add_count", metrics.ChanV2JobAdd, tags),
			telemetry.NewViewCount("cds/hatchery/chan_worker_starter_job_pop", metrics.ChanWorkerStarterPop, tags),
			telemetry.NewViewCount("cds/hatchery/worker_starter_running", metrics.WorkerStarterRunning, tags),
			telemetry.NewViewCount("cds/hatchery/job_received_in_queue_polling_ws_v1_count", metrics.JobReceivedInQueuePollingWSv1, tags),
			telemetry.NewViewCount("cds/hatchery/job_received_in_queue_polling_ws_v2_count", metrics.JobReceivedInQueuePollingWSv2, tags),
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
