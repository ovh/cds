package cdn

import (
	"context"
	"net/http"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/service"
)

func (s *Service) postDuplicateItemForJobHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var duplicateRequest sdk.CDNDuplicateItemRequest
		if err := service.UnmarshalBody(r, &duplicateRequest); err != nil {
			return err
		}

		items, err := item.LoadByRunJobID(ctx, s.Mapper, s.mustDBWithCtx(ctx), duplicateRequest.FromJob, gorpmapping.GetAllOptions.WithDecryption)
		if err != nil {
			return err
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		var completedCopies []sdk.CDNItem
		for _, i := range items {

			// Reload the item with fresh data
			src, err := item.LoadByID(ctx, s.Mapper, tx, i.ID, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}
			storageUnitItems, err := storage.LoadAllItemUnitsByItemIDs(ctx, s.Mapper, tx, src.ID, gorpmapping.GetAllOptions.WithDecryption)
			if err != nil {
				return err
			}

			// Only the log buffer holds the content of an item still being received
			incoming := src.Status == sdk.CDNStatusItemIncoming
			if incoming {
				bufferUnitItems := make([]sdk.CDNItemUnit, 0, 1)
				for _, sui := range storageUnitItems {
					if sui.UnitID == s.Units.LogsBuffer().ID() {
						bufferUnitItems = append(bufferUnitItems, sui)
					}
				}
				if len(bufferUnitItems) == 0 {
					log.Warn(ctx, "postDuplicateItemForJobHandler> incoming item %s has no buffer copy, it is not duplicated", src.ID)
					continue
				}
				storageUnitItems = bufferUnitItems
			}

			// Copy the item
			newItem := *src
			newItem.ID = ""
			switch newItem.Type {
			case sdk.CDNTypeItemJobStepLog, sdk.CDNTypeItemServiceLogV2:
				logRef, _ := newItem.GetCDNLogApiRefV2()
				logRef.RunJobID = duplicateRequest.ToJob
				logRef.RunAttempt++
				newItem.APIRef = logRef

				hashRef, err := logRef.ToHash()
				if err != nil {
					return err
				}
				newItem.APIRefHash = hashRef
			case sdk.CDNTypeItemRunResultV2:
				logRef, _ := newItem.GetCDNRunResultApiRefV2()
				logRef.RunJobID = duplicateRequest.ToJob
				logRef.RunAttempt++
				newItem.APIRef = logRef
				hashRef, err := logRef.ToHash()
				if err != nil {
					return err
				}
				newItem.APIRefHash = hashRef
			default:
				return sdk.WrapError(sdk.ErrInvalidData, "wrong item type %s", newItem.Type)
			}
			if err := item.Insert(ctx, s.Mapper, tx, &newItem); err != nil {
				return err
			}

			var bufferCopy sdk.CDNItemUnit
			for _, sui := range storageUnitItems {
				newSUI := sui
				newSUI.ID = ""
				newSUI.ItemID = newItem.ID
				newSUI.Item = &newItem
				if err := storage.InsertItemUnit(ctx, s.Mapper, tx, &newSUI); err != nil {
					return err
				}

				if sui.UnitID == s.Units.LogsBuffer().ID() {
					// Copy logs in buffer
					if err := s.Units.LogsBuffer().Copy(ctx, src.ID, newItem.ID); err != nil {
						return err
					}
					bufferCopy = newSUI
				}
			}

			// Force completion for the copy of an incoming item
			if incoming {
				if err := s.completeItem(ctx, tx, bufferCopy); err != nil {
					return err
				}
				completedCopies = append(completedCopies, newItem)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		// Trigger backend sync for new completed items
		for _, it := range completedCopies {
			s.Units.PushInSyncQueue(ctx, it.ID, it.Created)
		}
		return nil
	}
}
