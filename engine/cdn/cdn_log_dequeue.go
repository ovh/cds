package cdn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) dequeueJobLogs(ctx context.Context) error {
	defer func() {
		log.Info(ctx, "cdn: leaving dequeue job logs")
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dequeuCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			var hm handledMessage
			if err := s.Cache.DequeueWithContext(dequeuCtx, keyJobLogIncomingQueue, 1*time.Millisecond, &hm); err != nil {
				cancel()
				if !strings.Contains(err.Error(), "context deadline exceeded") {
					log.Error(ctx, "dequeueJobLogs: unable to dequeue job logs queue: %v", err)
				}
				continue
			}
			cancel()
			if hm.Signature.Worker == nil {
				continue
			}
			s.storeLogsWithRetry(ctx, sdk.CDNTypeItemStepLog, hm)
		}
	}
}

func (s *Service) dequeueServiceLogs(ctx context.Context) error {
	defer func() {
		log.Info(ctx, "cdn: leaving dequeue service logs")
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dequeuCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			var hm handledMessage
			if err := s.Cache.DequeueWithContext(dequeuCtx, keyServiceLogIncomingQueue, 30*time.Millisecond, &hm); err != nil {
				cancel()
				if !strings.Contains(err.Error(), "context deadline exceeded") {
					log.Error(ctx, "dequeueServiceLogs: unable to dequeue service logs queue: %v", err)
				}
				continue
			}
			cancel()
			if hm.Signature.Service == nil {
				continue
			}
			s.storeLogsWithRetry(ctx, sdk.CDNTypeItemServiceLog, hm)
		}
	}
}

func (s *Service) storeLogsWithRetry(ctx context.Context, itemType sdk.CDNItemType, hm handledMessage) {
	currentLog := buildMessage(hm)
	cpt := 0
	for {
		if err := s.storeLogs(ctx, itemType, hm.Signature, hm.IsTerminated, currentLog, hm.Line); err != nil {
			if sdk.ErrorIs(err, sdk.ErrLocked) && cpt < 10 {
				cpt++
				time.Sleep(250 * time.Millisecond)
				continue
			}
			err = sdk.WrapError(err, "unable to store log")
			log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			break
		}
		break
	}
}

func (s *Service) storeLogs(ctx context.Context, itemType sdk.CDNItemType, signature log.Signature, terminated bool, content string, line int64) error {
	it, err := s.loadOrCreateItem(ctx, itemType, signature)
	if err != nil {
		return err
	}

	iu, err := s.loadOrCreateItemUnitBuffer(ctx, it.ID)
	if err != nil {
		return err
	}

	// In case where the item was marked as complete we don't allow append of other logs
	if it.Status == sdk.CDNStatusItemCompleted {
		log.WarningWithFields(ctx, log.Fields{"item_apiref": it.APIRefHash}, "cdn:storeLogs: a log was received for item %s but status in already complete", it.ID)
		return nil
	}

	_, err = s.Units.Buffer.Add(*iu, uint(line), content, storage.WithOption{IslastLine: terminated})
	if err != nil {
		return err
	}

	// Send an event in WS broker to refresh streams on current item
	s.GoRoutines.Exec(ctx, "storeLogsPublishWSEvent", func(ctx context.Context) {
		s.publishWSEvent(*it)
	})

	maxLineKey := cache.Key("cdn", "log", "size", it.ID)
	maxItemLine := -1
	var bufferFull bool
	if terminated {
		maxItemLine = int(line)
		currentSize, err := s.Units.Buffer.Size(*iu)
		if err != nil {
			return err
		}

		// check if buffer is full
		switch it.Type {
		case sdk.CDNTypeItemStepLog:
			bufferFull = currentSize >= s.Cfg.Log.StepMaxSize
		case sdk.CDNTypeItemServiceLog:
			bufferFull = currentSize >= s.Cfg.Log.ServiceMaxSize
		}

		// store the score of last line
		if err := s.Cache.SetWithTTL(maxLineKey, maxItemLine, ItemLogGC); err != nil {
			return err
		}
	} else {
		_, err = s.Cache.Get(maxLineKey, &maxItemLine)
		if err != nil {
			log.Warning(ctx, "cdn:storeLogs: unable to get max line expected for current job: %v", err)
		}
	}

	logsSize, err := s.Units.Buffer.Card(*iu)
	if err != nil {
		return err
	}
	// If we have all lines or buffer is full and we received the last line
	if (terminated && bufferFull) || (maxItemLine >= 0 && maxItemLine+1 == logsSize) {
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
		_ = s.Cache.Delete(maxLineKey)
	}
	return nil
}

func (s *Service) loadOrCreateItem(ctx context.Context, itemType sdk.CDNItemType, signature log.Signature) (*sdk.CDNItem, error) {
	// Build cds api ref
	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     signature.ProjectKey,
		WorkflowName:   signature.WorkflowName,
		WorkflowID:     signature.WorkflowID,
		RunID:          signature.RunID,
		NodeRunName:    signature.NodeRunName,
		NodeRunID:      signature.NodeRunID,
		NodeRunJobName: signature.JobName,
		NodeRunJobID:   signature.JobID,
	}
	if signature.Worker != nil {
		apiRef.StepName = signature.Worker.StepName
		apiRef.StepOrder = signature.Worker.StepOrder
	}
	if signature.Service != nil {
		apiRef.RequirementServiceID = signature.Service.RequirementID
		apiRef.RequirementServiceName = signature.Service.RequirementName
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
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		if errInsert := item.Insert(ctx, s.Mapper, tx, it); errInsert == nil {
			if err := tx.Commit(); err != nil {
				return nil, sdk.WithStack(err)
			}
			log.InfoWithFields(ctx, log.Fields{
				"item_apiref": it.APIRefHash,
			}, "storeLogs> new item %s has been stored", it.ID)

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

func (s *Service) loadOrCreateItemUnitBuffer(ctx context.Context, itemID string) (*sdk.CDNItemUnit, error) {
	unit, err := storage.LoadUnitByName(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Buffer.Name())
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

		itemUnit, err = s.Units.NewItemUnit(ctx, s.Units.Buffer, it)
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
			return nil, err
		}
	}

	return itemUnit, nil
}
