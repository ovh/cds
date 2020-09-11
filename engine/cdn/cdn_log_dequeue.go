package cdn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) dequeueJobLogs(ctx context.Context) error {
	log.Info(ctx, "dequeueJobLogs: start")
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
			if err := s.Cache.DequeueWithContext(dequeuCtx, keyJobLogIncomingQueue, 30*time.Millisecond, &hm); err != nil {
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
			currentLog := buildMessage(hm)
			cpt := 0
			for {
				if err := s.storeLogs(ctx, sdk.CDNTypeItemStepLog, hm.Signature, hm.Status, currentLog, hm.Line); err != nil {
					if sdk.ErrorIs(err, sdk.ErrLocked) && cpt < 10 {
						cpt++
						time.Sleep(250 * time.Millisecond)
						continue
					}
					err = sdk.WrapError(err, "unable to store step log")
					log.ErrorWithFields(ctx, logrus.Fields{
						"stack_trace": fmt.Sprintf("%+v", err),
					}, "%s", err)
					break
				}
				break
			}
		}
	}
}

func (s *Service) dequeueServiceLogs(ctx context.Context) error {
	log.Info(ctx, "dequeueServiceLogs: start")
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
			if hm.Msg.Full == "" {
				continue
			}
			if !strings.HasSuffix(hm.Msg.Full, "\n") {
				hm.Msg.Full += "\n"
			}
			if err := s.storeLogs(ctx, sdk.CDNTypeItemServiceLog, hm.Signature, hm.Status, hm.Msg.Full, 0); err != nil {
				err = sdk.WrapError(err, "unable to store service log")
				log.ErrorWithFields(ctx, logrus.Fields{
					"stack_trace": fmt.Sprintf("%+v", err),
				}, "%s", err)
			}
		}
	}
}

func (s *Service) storeLogs(ctx context.Context, itemType sdk.CDNItemType, signature log.Signature, status string, content string, line int64) error {
	item, err := s.loadOrCreateIndexItem(ctx, itemType, signature)
	if err != nil {
		return err
	}

	iu, err := s.loadOrCreateIndexItemUnitBuffer(ctx, item.ID)
	if err != nil {
		return err
	}

	// In case where the item was marked as complete we don't allow append of other logs
	if item.Status == index.StatusItemCompleted {
		log.Warning(ctx, "cdn:storeLogs: a log was received for item %s but status in already complete", item.Hash)
		return nil
	}

	switch itemType {
	case sdk.CDNTypeItemStepLog:
		if err := s.Units.Buffer.Add(*iu, uint(line), content); err != nil {
			return err
		}
	case sdk.CDNTypeItemServiceLog:
		if err := s.Units.Buffer.Append(*iu, content); err != nil {
			return err
		}
	}

	maxLineKey := cache.Key("cdn", "log", "size", item.ID)
	maxIndexLine := -1
	if sdk.StatusIsTerminated(status) {
		maxIndexLine = int(line)
		// store the score of last line
		if err := s.Cache.SetWithTTL(maxLineKey, maxIndexLine, ItemLogGC); err != nil {
			return err
		}
	} else {
		_, err = s.Cache.Get(maxLineKey, &maxIndexLine)
		if err != nil {
			log.Warning(ctx, "cdn:storeLogs: unable to get max line expected for current job: %v", err)
		}
	}

	logsSize, err := s.Units.Buffer.Card(*iu)
	if err != nil {
		return err
	}
	// If we have all lines
	if maxIndexLine >= 0 && maxIndexLine+1 == logsSize {
		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback()
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

func (s *Service) loadOrCreateIndexItem(ctx context.Context, itemType sdk.CDNItemType, signature log.Signature) (*index.Item, error) {
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

	item, err := index.LoadItemByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), hashRef, itemType)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, err
		}
		// Insert data
		item = &index.Item{
			APIRef:     apiRef,
			Type:       itemType,
			APIRefHash: hashRef,
			Status:     index.StatusItemIncoming,
		}

		tx, err := s.mustDBWithCtx(ctx).Begin()
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		if errInsert := index.InsertItem(ctx, s.Mapper, tx, item); errInsert == nil {
			if err := tx.Commit(); err != nil {
				return nil, sdk.WithStack(err)
			}
			return item, nil
		} else if !sdk.ErrorIs(errInsert, sdk.ErrConflictData) {
			return nil, errInsert
		}

		// reload if item already exist
		item, err = index.LoadItemByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), hashRef, itemType)
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func (s *Service) loadOrCreateIndexItemUnitBuffer(ctx context.Context, itemID string) (*storage.ItemUnit, error) {
	unit, err := storage.LoadUnitByName(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Buffer.Name())
	if err != nil {
		return nil, err
	}

	item, err := index.LoadItemByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), itemID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return nil, err
	}

	itemUnit, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), unit.ID, itemID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil, err
		}

		itemUnit, err = s.Units.NewItemUnit(ctx, s.Units.Buffer, item)
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
