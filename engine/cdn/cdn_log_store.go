package cdn

import (
	"context"
	"strings"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
)

// storeLogsCache memoizes item and item unit lookups for the lifetime of a dequeued batch:
// without it, every single log line costs ~4 SELECTs (incl. 2 decryptions) against the
// database. Entries are keyed by apiRefHash / itemID because one job queue carries several
// steps and service logs, each being a distinct item.
type storeLogsCache struct {
	items     map[string]*sdk.CDNItem     // by apiRefHash
	itemUnits map[string]*sdk.CDNItemUnit // by itemID
}

func newStoreLogsCache() *storeLogsCache {
	return &storeLogsCache{
		items:     make(map[string]*sdk.CDNItem),
		itemUnits: make(map[string]*sdk.CDNItemUnit),
	}
}

func (s *Service) sendToBufferWithRetry(ctx context.Context, hms []handledMessage) error {
	if len(hms) == 0 {
		return nil
	}

	batchCache := newStoreLogsCache()

	// Browse all messages
	for _, hm := range hms {
		var itemType sdk.CDNItemType
		if hm.Signature.JobID != 0 {
			if hm.Signature.Service != nil {
				itemType = sdk.CDNTypeItemServiceLog
			} else {
				itemType = sdk.CDNTypeItemStepLog
			}
		} else {
			if hm.Signature.HatcheryService != nil {
				itemType = sdk.CDNTypeItemServiceLogV2
			} else {
				itemType = sdk.CDNTypeItemJobStepLog
			}
		}
		currentLog := buildMessage(hm)
		cpt := 0
		for {
			if err := s.storeLogsWithCache(ctx, batchCache, itemType, hm.Signature, hm.IsTerminated, currentLog); err != nil {
				if sdk.ErrorIs(err, sdk.ErrLocked) && cpt < 10 {
					cpt++
					time.Sleep(250 * time.Millisecond)
					continue
				}
				return err
			}
			break
		}
	}
	return nil
}

func (s *Service) storeLogs(ctx context.Context, itemType sdk.CDNItemType, signature cdn.Signature, terminated bool, content string) error {
	return s.storeLogsWithCache(ctx, newStoreLogsCache(), itemType, signature, terminated, content)
}

func (s *Service) storeLogsWithCache(ctx context.Context, c *storeLogsCache, itemType sdk.CDNItemType, signature cdn.Signature, terminated bool, content string) error {
	apiRef, err := sdk.NewCDNApiRef(itemType, signature)
	if err != nil {
		return err
	}
	hashRef, err := apiRef.ToHash()
	if err != nil {
		return err
	}

	it, ok := c.items[hashRef]
	if !ok {
		it, err = s.loadOrCreateItem(ctx, itemType, signature)
		if err != nil {
			return err
		}
		c.items[hashRef] = it
	}

	var t0 = it.Created.UnixNano() / 1000000 // convert to ms
	var t1 = signature.Timestamp / 1000000

	ctx = context.WithValue(ctx, storage.FieldAPIRef, it.APIRefHash)

	iu, ok := c.itemUnits[it.ID]
	if !ok {
		iu, err = s.loadOrCreateItemUnitBuffer(ctx, it.ID, itemType)
		if err != nil {
			return err
		}
		c.itemUnits[it.ID] = iu
	}

	// In case where the item was marked as complete we don't allow append of other logs
	if it.Status == sdk.CDNStatusItemCompleted {
		log.Warn(ctx, "cdn:storeLogs: a log was received for item %s but status in already complete", it.ID)
		return nil
	}

	bufferUnit := s.Units.LogsBuffer()
	countLine, err := bufferUnit.Card(*iu)
	if err != nil {
		return err
	}

	// Add the number of millisecond since creation
	ms := t1 - t0
	if ms < 0 {
		ms = 0
	}

	if terminated {
		content = strings.Trim(content, " ")
	}
	emptyLastLogLine := (content == "" || content == "\n") && terminated
	if !emptyLastLogLine {
		// Build the score from the "countLine" as the interger part and "ms" as floating part
		if err := bufferUnit.Add(*iu, uint(countLine), uint(ms), content); err != nil {
			return err
		}
	}

	// Send an event in WS broker to refresh streams on current item
	s.publishWSEvent(*iu)

	// If we have all lines or buffer is full and we received the last line
	if terminated {
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}

		defer tx.Rollback() // nolint
		if err := s.completeItem(ctx, tx, *iu); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		// completeItem changed the item status in database: evict the cached entries so a
		// late line for this item reloads a fresh state and gets skipped as completed.
		delete(c.items, hashRef)
		delete(c.itemUnits, it.ID)

		s.Units.PushInSyncQueue(ctx, it.ID, it.Created)
	}

	return nil
}

func (s *Service) loadOrCreateItem(ctx context.Context, itemType sdk.CDNItemType, signature cdn.Signature) (*sdk.CDNItem, error) {
	// Build cds api ref
	apiRef, err := sdk.NewCDNApiRef(itemType, signature)
	if err != nil {
		return nil, err
	}
	hashRef, err := apiRef.ToHash()
	if err != nil {
		return nil, err
	}

	it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), hashRef, itemType)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, err
		}
		// Insert data
		it = &sdk.CDNItem{
			APIRef:     apiRef,
			Type:       itemType,
			APIRefHash: hashRef,
			Status:     sdk.CDNStatusItemIncoming,
			Created:    time.Unix(0, signature.Timestamp),
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		ctx = context.WithValue(ctx, storage.FieldAPIRef, it.APIRefHash)

		if errInsert := item.Insert(ctx, s.Mapper, tx, it); errInsert == nil {
			if err := tx.Commit(); err != nil {
				return nil, sdk.WithStack(err)
			}
			log.Info(ctx, "storeLogs> new item %s has been stored", it.ID)

			return it, nil
		} else if !sdk.ErrorIs(errInsert, sdk.ErrConflictData) {
			return nil, errInsert
		}

		// reload if item already exist
		it, err = item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), hashRef, itemType)
		if err != nil {
			return nil, err
		}
	}

	return it, nil
}

func (s *Service) loadOrCreateItemUnitBuffer(ctx context.Context, itemID string, itemType sdk.CDNItemType) (*sdk.CDNItemUnit, error) {
	bufferUnit := s.Units.GetBuffer(itemType)
	unit, err := storage.LoadUnitByName(ctx, s.Mapper, s.mustDBWithCtx(ctx), bufferUnit.Name())
	if err != nil {
		return nil, err
	}

	it, err := item.LoadByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), itemID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return nil, err
	}

	itemUnit, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), unit.ID, itemID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, err
		}

		itemUnit, err = s.Units.NewItemUnit(ctx, bufferUnit, it)
		if err != nil {
			return nil, err
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		if errInsert := storage.InsertItemUnit(ctx, s.Mapper, tx, itemUnit); errInsert == nil {
			if err := tx.Commit(); err != nil {
				return nil, sdk.WithStack(err)
			}
			return itemUnit, nil
		} else if !sdk.ErrorIs(errInsert, sdk.ErrConflictData) {
			return nil, errInsert
		}

		itemUnit, err = storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), unit.ID, itemID, gorpmapper.GetOptions.WithDecryption)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to load item unit %s/%s", unit.ID, itemID)
		}
	}

	return itemUnit, nil
}
