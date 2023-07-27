package cdn

import (
	"context"
	"fmt"
	"time"

	"github.com/rockbears/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
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

	s.Metrics.ItemToSyncCount = stats.Int64("cdn/items/sync_lag", "number of items to sync per storage and type", stats.UnitDimensionless)
	itemToSyncCountView := telemetry.NewViewLast(s.Metrics.ItemToSyncCount.Name(), s.Metrics.ItemToSyncCount, []tag.Key{tagStorage, tagItemType})

	s.Metrics.WSClients = stats.Int64("cdn/websocket_clients", "number of  websocket clients", stats.UnitDimensionless)
	metricsWSClients := telemetry.NewViewCount(s.Metrics.WSClients.Name(), s.Metrics.WSClients, []tag.Key{tagServiceName, tagItemType})

	s.Metrics.WSEvents = stats.Int64("cdn/websocket_events", "number of websocket events", stats.UnitDimensionless)
	metricsWSEvents := telemetry.NewViewCount(s.Metrics.WSEvents.Name(), s.Metrics.WSEvents, []tag.Key{tagServiceName, tagItemType})

	s.Metrics.ItemToDelete = stats.Int64("cdn/items/to_delete", "number of items to delete per type", stats.UnitDimensionless)
	itemToDeleteView := telemetry.NewViewLast(s.Metrics.ItemToDelete.Name(), s.Metrics.ItemToDelete, []tag.Key{tagItemType})

	s.Metrics.ItemUnitToDelete = stats.Int64("cdn/item_units/to_delete", "number of item units to delete per storage and type", stats.UnitDimensionless)
	itemUnitToDeleteView := telemetry.NewViewLast(s.Metrics.ItemUnitToDelete.Name(), s.Metrics.ItemUnitToDelete, []tag.Key{tagStorage, tagItemType})

	if s.DBConnectionFactory != nil {
		s.GoRoutines.RunWithRestart(ctx, "cds-compute-metrics", func(ctx context.Context) {
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
		metricsWSClients,
		metricsWSEvents,
		itemToSyncCountView,
		itemToDeleteView,
		itemUnitToDeleteView,
	)
}

func (s *Service) ComputeMetrics(ctx context.Context) {
	tickPercentil := time.NewTicker(1 * time.Hour)
	defer tickPercentil.Stop()
	tickStatsItems := time.NewTicker(time.Duration(s.Cfg.Metrics.Frequency) * time.Second)
	defer tickStatsItems.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tickStatsItems.C:
			start := time.Now()
			// All Items by type
			allItemsByType, err := item.CountItems(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}
			for _, stat := range allItemsByType {
				ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagStatus, stat.Status)
				telemetry.Record(ctxItem, s.Metrics.itemInDatabaseCount, stat.Number)
			}

			// Count all unit_item by type
			var storageStatsBuffer []storage.Stat
			for _, bu := range s.Units.Buffers {
				storageStatsBuffer = append(storageStatsBuffer, s.countItemsForUnit(ctx, bu)...)
			}

			for _, stat := range storageStatsBuffer {
				ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagStorage, stat.StorageName)
				telemetry.Record(ctxItem, s.Metrics.itemPerStorageUnitCount, stat.Number)
			}

			var storageStats []storage.Stat
			for _, su := range s.Units.Storages {
				if !su.CanSync() {
					continue
				}
				storageStats = append(storageStats, s.countItemsForUnit(ctx, su)...)
			}

			for _, stat := range storageStats {
				ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagStorage, stat.StorageName)
				telemetry.Record(ctxItem, s.Metrics.itemPerStorageUnitCount, stat.Number)

				key := fmt.Sprintf("backend/%s/%s", stat.StorageName, stat.Type)
				if previous, ok := s.storageUnitSizes.Load(key); ok {
					s.storageUnitPreviousSizes.Store(key, previous)
				}
				s.storageUnitSizes.Store(key, stat.Number)

				// to synchronized
				for _, allItems := range allItemsByType {
					if allItems.Type == stat.Type && allItems.Status == sdk.CDNStatusItemCompleted {
						ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagStorage, stat.StorageName, telemetry.TagType, stat.Type)
						lag := allItems.Number - stat.Number
						telemetry.Record(ctxItem, s.Metrics.ItemToSyncCount, lag)
						if previous, ok := s.storageUnitLags.Load(key); ok {
							s.storageUnitPreviousLags.Store(key, previous)
						}
						s.storageUnitLags.Store(key, lag)
						break
					}
				}
			}

			nbItemsToDelete, err := item.CountItemsToDelete(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}

			telemetry.Record(ctx, s.Metrics.ItemToDelete, nbItemsToDelete)

			storageStats, err = storage.CountItemUnitToDelete(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}

			for _, stat := range storageStats {
				ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagStorage, stat.StorageName)
				telemetry.Record(ctxItem, s.Metrics.ItemUnitToDelete, stat.Number)
			}

			elapsed := time.Since(start)
			if elapsed > 5*time.Second {
				log.Warn(ctx, "ComputeMetrics is too long, it took %v", elapsed)
			} else if elapsed > 15*time.Second {
				log.Error(ctx, "ComputeMetrics is too long, it took %v", elapsed)
			}
		case <-tickPercentil.C:
			statsPercentils, err := item.CountItemSizePercentil(s.mustDBWithCtx(ctx))
			if err != nil {
				log.Error(ctx, "cdn> Unable to compute metrics: %v", err)
				continue
			}
			for _, stat := range statsPercentils {
				// Export only 50, 75, 90, 95, 99, 100 percentil
				switch stat.Percentile {
				case 50, 75, 90, 95, 99, 100:
					ctxItem := telemetry.ContextWithTag(ctx, telemetry.TagType, stat.Type, telemetry.TagPercentil, stat.Percentile)
					telemetry.Record(ctxItem, s.Metrics.ItemSize, stat.Size)
				}
			}
		}
	}
}

func (s *Service) countItemsForUnit(ctx context.Context, storageUnit storage.Interface) []storage.Stat {
	types := []sdk.CDNItemType{sdk.CDNTypeItemStepLog, sdk.CDNTypeItemServiceLog, sdk.CDNTypeItemRunResult, sdk.CDNTypeItemJobStepLog}
	var storageStats []storage.Stat
	for _, typ := range types {
		suStats, err := storage.CountItemsForUnitByType(s.mustDBWithCtx(ctx), storageUnit.ID(), string(typ))
		if err != nil {
			log.Error(ctx, "cdn> Unable to compute CountItemsForUnit for %s: %v", storageUnit.Name(), err)
			return nil
		}
		for i := range suStats {
			s := suStats[i]
			s.StorageName = storageUnit.Name()
			s.Type = string(typ)
			storageStats = append(storageStats, s)
		}
	}
	return storageStats
}
