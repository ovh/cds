package cdn

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) initMetrics(ctx context.Context) error {
	tagServiceType := telemetry.MustNewKey(telemetry.TagServiceType)
	tagServiceName := telemetry.MustNewKey(telemetry.TagServiceName)
	tagStorage := telemetry.MustNewKey(telemetry.TagStorage)
	tagStorageSource := telemetry.MustNewKey(telemetry.TagStorage + "_source")
	tagStorageDest := telemetry.MustNewKey(telemetry.TagStorage + "_dest")

	tagItemType := telemetry.MustNewKey(telemetry.TagType)
	tagStatus := telemetry.MustNewKey(telemetry.TagStatus)

	s.RegisterCommonMetricsView(ctx)

	s.Metrics.tcpServerErrorsCount = stats.Int64("cdn/tcp/errors", "tcp server number of errors", stats.UnitDimensionless)
	tcpServerErrorsCountView := telemetry.NewViewCount(s.Metrics.tcpServerErrorsCount.Name(), s.Metrics.tcpServerErrorsCount, []tag.Key{tagServiceName, tagServiceType})

	s.Metrics.tcpServerHitsCount = stats.Int64("cdn/tcp/hits", "tcp server number of hits", stats.UnitDimensionless)
	tcpServerHitsCountView := telemetry.NewViewCount(s.Metrics.tcpServerHitsCount.Name(), s.Metrics.tcpServerHitsCount, []tag.Key{tagServiceName, tagServiceType})

	s.Metrics.tcpServerStepLogCount = stats.Int64("cdn/tcp/step_log_count", "number of worker log received", stats.UnitDimensionless)
	tcpServerStepLogCountView := telemetry.NewViewCount(s.Metrics.tcpServerStepLogCount.Name(), s.Metrics.tcpServerStepLogCount, []tag.Key{tagServiceName, tagServiceType})

	s.Metrics.tcpServerServiceLogCount = stats.Int64("cdn/tcp/service_log_count", "number of service log received", stats.UnitDimensionless)
	tcpServerServiceLogCountView := telemetry.NewViewCount(s.Metrics.tcpServerServiceLogCount.Name(), s.Metrics.tcpServerServiceLogCount, []tag.Key{tagServiceName, tagServiceType})

	s.Metrics.itemCompletedByGCCount = stats.Int64("cdn/items/completed_by_gc", "number of items completed by GC", stats.UnitDimensionless)
	itemCompletedByGCCountView := telemetry.NewViewCount(s.Metrics.itemCompletedByGCCount.Name(), s.Metrics.itemCompletedByGCCount, []tag.Key{tagServiceName, tagServiceType})

	s.Metrics.StorageThroughput = stats.Float64("cdn/storage/throughput", "read throughput per storages (in MBytes per seconds)", stats.UnitDimensionless)
	StorageThroughputView := &view.View{
		Name:        s.Metrics.StorageThroughput.Name(),
		Description: s.Metrics.StorageThroughput.Description(),
		TagKeys:     []tag.Key{tagServiceType, tagServiceName, tagStorageSource, tagStorageDest},
		Measure:     s.Metrics.StorageThroughput,
		Aggregation: telemetry.DefaultSizeDistribution,
	}

	s.Metrics.itemInDatabaseCount = stats.Int64("cdn/items/count", "number of items in database by type and status", stats.UnitDimensionless)
	itemInDatabaseCountView := telemetry.NewViewCount(s.Metrics.itemInDatabaseCount.Name(), s.Metrics.itemInDatabaseCount, []tag.Key{tagItemType, tagStatus})

	s.Metrics.itemPerStorageUnitCount = stats.Int64("cdn/items/count_per_storage", "number of items per storage type", stats.UnitDimensionless)
	itemPerStorageUnitCountView := telemetry.NewViewCount(s.Metrics.itemPerStorageUnitCount.Name(), s.Metrics.itemPerStorageUnitCount, []tag.Key{tagStorage, tagItemType})

	s.Metrics.ItemSize = stats.Float64("cdn/items/size", "size items by types (in KBytes)", stats.UnitBytes)
	itemSizeView := &view.View{
		Name:        s.Metrics.ItemSize.Name(),
		Description: s.Metrics.ItemSize.Description(),
		TagKeys:     []tag.Key{tagItemType},
		Measure:     s.Metrics.ItemSize,
		Aggregation: telemetry.DefaultSizeDistribution,
	}

	if s.DBConnectionFactory != nil {
		s.GoRoutines.Run(ctx, "cds-compute-metrics", func(ctx context.Context) {
			s.ComputeMetrics(ctx)
		})
	}

	return telemetry.RegisterView(ctx,
		tcpServerErrorsCountView,
		tcpServerHitsCountView,
		tcpServerStepLogCountView,
		tcpServerServiceLogCountView,
		itemCompletedByGCCountView,
		StorageThroughputView,
		itemInDatabaseCountView,
		itemPerStorageUnitCountView,
		itemSizeView,
	)
}

func (s *Service) ComputeMetrics(ctx context.Context) {
	for {
		time.Sleep(10 * time.Second)

		select {
		case <-ctx.Done():
			return
		default:
			stats, err := item.CountItems(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}

			for _, stat := range stats {
				ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagStatus, stat.Status)
				telemetry.Record(ctxItem, s.Metrics.itemInDatabaseCount, stat.Number)
			}

			storageStats, err := storage.CountItems(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}

			for _, stat := range storageStats {
				ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagStorage, stat.StorageName)
				telemetry.Record(ctxItem, s.Metrics.itemPerStorageUnitCount, stat.Number)
			}
		}
	}
}
