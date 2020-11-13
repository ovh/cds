package cdn

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) sendToCDS(ctx context.Context, msgs []handledMessage) error {
	switch {
	case msgs[0].Signature.Service != nil:
		for _, msg := range msgs {
			// Format line
			msg.Msg.Full = buildMessage(msg)
			if msg.Signature.Service != nil {
				logs := sdk.ServiceLog{
					ServiceRequirementName: msg.Signature.Service.RequirementName,
					ServiceRequirementID:   msg.Signature.Service.RequirementID,
					WorkflowNodeJobRunID:   msg.Signature.JobID,
					WorkflowNodeRunID:      msg.Signature.NodeRunID,
					Val:                    msg.Msg.Full,
				}
				if err := s.Client.QueueServiceLogs(ctx, []sdk.ServiceLog{logs}); err != nil {
					return err
				}
			}
		}
		return nil
	default:
		// Aggregate messages by step
		hms := make(map[string]handledMessage, len(msgs))
		for _, msg := range msgs {
			// Format line
			msg.Msg.Full = buildMessage(msg)

			k := fmt.Sprintf("%d-%d-%d", msg.Signature.JobID, msg.Signature.NodeRunID, msg.Signature.Worker.StepOrder)
			// Aggregates lines in a single message
			if _, ok := hms[k]; ok {
				full := hms[k].Msg.Full
				msg.Msg.Full = fmt.Sprintf("%s%s", full, msg.Msg.Full)
				hms[k] = msg
			} else {
				hms[k] = msg
			}
		}

		// Send logs to CDS API by step
		for _, hm := range hms {
			now := time.Now()
			l := sdk.Log{
				JobID:        hm.Signature.JobID,
				NodeRunID:    hm.Signature.NodeRunID,
				LastModified: &now,
				StepOrder:    hm.Signature.Worker.StepOrder,
				Val:          hm.Msg.Full,
			}
			if err := s.Client.QueueSendLogs(ctx, hm.Signature.JobID, l); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) sendToBufferWithRetry(ctx context.Context, hms []handledMessage) error {
	if len(hms) == 0 {
		return nil
	}

	jobID := strconv.Itoa(int(hms[0].Signature.JobID))
	lineKey := cache.Key(keyJobLogLines, jobID)
	if hms[0].Signature.Service != nil {
		lineKey = cache.Key(keyServiceLogLines, jobID, strconv.Itoa(int(hms[0].Signature.Service.RequirementID)))
	}
	var currentLine int64
	if _, err := s.Cache.Get(lineKey, &currentLine); err != nil {
		return sdk.WithStack(err)
	}
	// Browse all messages
	for _, hm := range hms {

		var itemType sdk.CDNItemType
		if hm.Signature.Service != nil {
			itemType = sdk.CDNTypeItemServiceLog
		} else {
			itemType = sdk.CDNTypeItemStepLog
		}
		currentLog := buildMessage(hm)
		cpt := 0
		for {
			if err := s.storeLogs(ctx, itemType, hm.Signature, hm.IsTerminated, currentLog, currentLine); err != nil {
				if sdk.ErrorIs(err, sdk.ErrLocked) && cpt < 10 {
					cpt++
					time.Sleep(250 * time.Millisecond)
					continue
				}
				return err
			}
			break
		}
		currentLine++
		if err := s.Cache.SetWithTTL(lineKey, &currentLine, 3600*24); err != nil {
			return err
		}
	}
	return nil
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

	if err := s.Units.Buffer.Add(*iu, uint(line), content); err != nil {
		return err
	}

	// Send an event in WS broker to refresh streams on current item
	s.GoRoutines.Exec(ctx, "storeLogsPublishWSEvent", func(ctx context.Context) {
		s.publishWSEvent(*it)
	})

	maxLineKey := cache.Key("cdn", "log", "size", it.ID)
	maxItemLine := -1
	if terminated {
		maxItemLine = int(line)

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
