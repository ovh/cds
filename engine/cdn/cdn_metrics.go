package cdn

import (
	"context"
	"time"

	"go.opencensus.io/stats"
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
	tagItemType := telemetry.MustNewKey(telemetry.TagType)
	tagStatus := telemetry.MustNewKey(telemetry.TagStatus)
	tagPercentil := telemetry.MustNewKey(telemetry.TagPercentil)

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

	s.Metrics.itemInDatabaseCount = stats.Int64("cdn/items/count", "number of items in database by type and status", stats.UnitDimensionless)
	itemInDatabaseCountView := telemetry.NewViewLast(s.Metrics.itemInDatabaseCount.Name(), s.Metrics.itemInDatabaseCount, []tag.Key{tagItemType, tagStatus})

	s.Metrics.itemPerStorageUnitCount = stats.Int64("cdn/items/count_per_storage", "number of items per storage and type", stats.UnitDimensionless)
	itemPerStorageUnitCountView := telemetry.NewViewLast(s.Metrics.itemPerStorageUnitCount.Name(), s.Metrics.itemPerStorageUnitCount, []tag.Key{tagStorage, tagItemType})

	s.Metrics.ItemSize = stats.Int64("cdn/items/size", "size items by type (in bytes) by percentil", stats.UnitBytes)
	itemSizeView := telemetry.NewViewLast(s.Metrics.ItemSize.Name(), s.Metrics.ItemSize, []tag.Key{tagItemType, tagPercentil})

	s.Metrics.ItemToSyncCount = stats.Int64("cdn/items/sync", "number of items to sync per storage and type", stats.UnitDimensionless)
	itemToSyncCountView := telemetry.NewViewLast(s.Metrics.ItemToSyncCount.Name(), s.Metrics.ItemToSyncCount, []tag.Key{tagStorage, tagItemType})

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
		itemInDatabaseCountView,
		itemPerStorageUnitCountView,
		itemSizeView,
		itemToSyncCountView,
	)
}

func (s *Service) ComputeMetrics(ctx context.Context) {
	for {
		time.Sleep(30 * time.Second)

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

			statsSize, err := item.CountItemSizePercentil(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}

			for _, stat := range statsSize {
				// Export only 50, 75, 90, 95, 99, 100 percentil
				switch stat.Percentile {
				case 50, 75, 90, 95, 99, 100:
					ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagPercentil, stat.Percentile)
					telemetry.Record(ctxItem, s.Metrics.ItemSize, stat.Size)
				}
			}

			statsItemToSync, err := storage.CountUnknownItemsByStorage(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}

			for _, stat := range statsItemToSync {
				ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagStorage, stat.StorageName, telemetry.TagType, stat.Type)
				telemetry.Record(ctxItem, s.Metrics.ItemToSyncCount, stat.Number)
			}
		}
	}
}
