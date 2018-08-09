package hatchery

import (
	"fmt"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/log"
)

// Stats returns the metric stats measures
func (c *Common) Stats() *hatchery.Stats {
	return &c.stats
}

func (c *Common) initStats(hatcheryName string) error {
	label := fmt.Sprintf("cds/%s/%s/jobs", c.ServiceName(), hatcheryName)
	c.stats.Jobs = stats.Int64(label, "number of analyzed jobs", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/spawned_workers", c.ServiceName(), hatcheryName)
	c.stats.SpawnedWorkers = stats.Int64(label, "number of spawned workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/pending_workers", c.ServiceName(), hatcheryName)
	c.stats.PendingWorkers = stats.Int64(label, "number of pending workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/registering_workers", c.ServiceName(), hatcheryName)
	c.stats.RegisteringWorkers = stats.Int64(label, "number of registering workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/waiting_workers", c.ServiceName(), hatcheryName)
	c.stats.WaitingWorkers = stats.Int64(label, "number of waiting workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/checking_workers", c.ServiceName(), hatcheryName)
	c.stats.CheckingWorkers = stats.Int64(label, "number of checking workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/building_workers", c.ServiceName(), hatcheryName)
	c.stats.BuildingWorkers = stats.Int64(label, "number of building workers", stats.UnitDimensionless)

	label = fmt.Sprintf("cds/%s/%s/disabled_workers", c.ServiceName(), hatcheryName)
	c.stats.DisabledWorkers = stats.Int64(label, "number of disabled workers", stats.UnitDimensionless)

	log.Info("hatchery> Stats initialized on %s", c.ServiceName())

	tags := []tag.Key{hatchery.TagHatchery, hatchery.TagHatcheryName}

	return observability.RegisterView(
		&view.View{
			Name:        "jobs_count",
			Description: c.stats.Jobs.Description(),
			Measure:     c.stats.Jobs,
			Aggregation: view.Count(),
			TagKeys:     tags,
		},
		&view.View{
			Name:        "spawned_worker_count",
			Description: c.stats.SpawnedWorkers.Description(),
			Measure:     c.stats.SpawnedWorkers,
			Aggregation: view.Count(),
			TagKeys:     tags,
		},
		&view.View{
			Name:        "pending_workers",
			Description: c.stats.PendingWorkers.Description(),
			Measure:     c.stats.PendingWorkers,
			Aggregation: view.LastValue(),
			TagKeys:     tags,
		},
		&view.View{
			Name:        "registering_workers",
			Description: c.stats.RegisteringWorkers.Description(),
			Measure:     c.stats.RegisteringWorkers,
			Aggregation: view.LastValue(),
			TagKeys:     tags,
		},
		&view.View{
			Name:        "waiting_workers",
			Description: c.stats.WaitingWorkers.Description(),
			Measure:     c.stats.WaitingWorkers,
			Aggregation: view.LastValue(),
			TagKeys:     tags,
		},
		&view.View{
			Name:        "checking_workers",
			Description: c.stats.CheckingWorkers.Description(),
			Measure:     c.stats.CheckingWorkers,
			Aggregation: view.LastValue(),
			TagKeys:     tags,
		},
		&view.View{
			Name:        "building_workers",
			Description: c.stats.BuildingWorkers.Description(),
			Measure:     c.stats.BuildingWorkers,
			Aggregation: view.LastValue(),
			TagKeys:     tags,
		},
		&view.View{
			Name:        "disabled_workers",
			Description: c.stats.DisabledWorkers.Description(),
			Measure:     c.stats.DisabledWorkers,
			Aggregation: view.LastValue(),
			TagKeys:     tags,
		},
	)
}
