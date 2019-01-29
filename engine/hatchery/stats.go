package hatchery

import (
	"fmt"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// Metrics returns the metric stats measures
func (c *Common) Metrics() *hatchery.Metrics {
	return &c.metrics
}

func (c *Common) initMetrics(hatcheryName string) error {
	label := fmt.Sprintf("cds/%s/%s/jobs", c.ServiceName(), hatcheryName)
	c.metrics.Jobs = stats.Int64(label, "number of analyzed jobs", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/jobs_sse", c.ServiceName(), hatcheryName)
	c.metrics.JobsSSE = stats.Int64(label, "number of analyzed jobs from SSE", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/spawned_workers", c.ServiceName(), hatcheryName)
	c.metrics.SpawnedWorkers = stats.Int64(label, "number of spawned workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/pending_workers", c.ServiceName(), hatcheryName)
	c.metrics.PendingWorkers = stats.Int64(label, "number of pending workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/registering_workers", c.ServiceName(), hatcheryName)
	c.metrics.RegisteringWorkers = stats.Int64(label, "number of registering workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/waiting_workers", c.ServiceName(), hatcheryName)
	c.metrics.WaitingWorkers = stats.Int64(label, "number of waiting workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/checking_workers", c.ServiceName(), hatcheryName)
	c.metrics.CheckingWorkers = stats.Int64(label, "number of checking workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/building_workers", c.ServiceName(), hatcheryName)
	c.metrics.BuildingWorkers = stats.Int64(label, "number of building workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/disabled_workers", c.ServiceName(), hatcheryName)
	c.metrics.DisabledWorkers = stats.Int64(label, "number of disabled workers", stats.UnitDimensionless)

	log.Info("hatchery> Stats initialized on %s", c.ServiceName())

	tagCDSInstance, _ := tag.NewKey("cds")
	tags := []tag.Key{tagCDSInstance, hatchery.TagHatchery, hatchery.TagHatcheryName}

	return observability.RegisterView(
		observability.NewViewCount("jobs_count", c.metrics.Jobs),
		observability.NewViewCount("jobs_sse_count", c.metrics.JobsSSE),
		observability.NewViewCount("spawned_worker_count", c.metrics.SpawnedWorkers),
		observability.NewViewLast("pending_workers", c.metrics.PendingWorkers, tags),
		observability.NewViewLast("registering_workers", c.metrics.RegisteringWorkers, tags),
		observability.NewViewLast("waiting_workers", c.metrics.WaitingWorkers, tags),
		observability.NewViewLast("checking_workers", c.metrics.CheckingWorkers, tags),
		observability.NewViewLast("building_workers", c.metrics.BuildingWorkers, tags),
		observability.NewViewLast("disabled_workers", c.metrics.DisabledWorkers, tags),
	)
}
