package cdn

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	ItemLogGC = 24 * 3600
)

func (s *Service) CompleteWaitingItems(ctx context.Context) {
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:CompleteWaitingItems: %v", ctx.Err())
			}
			return
		case <-tick.C:
			itemUnits, err := storage.LoadOldItemUnitByItemStatusAndDuration(ctx, s.Mapper, s.mustDBWithCtx(ctx), index.StatusItemIncoming, ItemLogGC)
			if err != nil {
				log.Warning(ctx, "cdn:CompleteWaitingItems: unable to get items ids: %v", err)
				continue
			}
			log.Debug("cdn:CompleteWaitingItems: %d items to complete", len(itemUnits))
			for _, itemUnit := range itemUnits {
				tx, err := s.mustDBWithCtx(ctx).Begin()
				if err != nil {
					err = sdk.WrapError(err, "unable to start transaction")
					log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
					continue
				}
				if err := s.completeItem(ctx, tx, itemUnit); err != nil {
					_ = tx.Rollback()
					log.Warning(ctx, "cdn:CompleteWaitingItems: unable to complete item %s: %v", itemUnit.ItemID, err)
					continue
				}
				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
					log.Warning(ctx, "cdn:CompleteWaitingItems: unable to commit transaction: %v", err)
					continue
				}
			}
		}
	}
}
