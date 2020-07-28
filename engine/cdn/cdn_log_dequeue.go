package cdn

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/symmecrypt/convergent"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) dequeueJobLogs(ctx context.Context) error {
	log.Info(ctx, "dequeueJobLogs: start")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dequeuCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
			var hm handledMessage
			if err := s.Cache.DequeueWithContext(dequeuCtx, keyJobLogIncomingQueue, 30*time.Millisecond, &hm); err != nil {
				cancel()
				if strings.Contains(err.Error(), "context deadline exceeded") {
					return nil
				}
				log.Error(ctx, "dequeueJobLogs: unable to dequeue job logs queue: %v", err)
				continue
			}
			cancel()
			if hm.Signature.Worker == nil {
				continue
			}

			cpt := 0
			for {
				if err := s.storeStepLogs(ctx, hm); err != nil {
					if sdk.ErrorIs(err, sdk.ErrLocked) && cpt < 10 {
						cpt++
						time.Sleep(250 * time.Millisecond)
						continue
					}
					log.Error(ctx, "dequeueJobLogs: unable to store step log: %v", err)
				}
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
			var serviceLog sdk.ServiceLog
			if err := s.Cache.DequeueWithContext(dequeuCtx, keyServiceLogIncomingQueue, 30*time.Millisecond, &serviceLog); err != nil {
				cancel()
				if strings.Contains(err.Error(), "context deadline exceeded") {
					return nil
				}
				log.Error(ctx, "dequeueServiceLogs: unable to dequeue service logs queue: %v", err)
				continue
			}
			cancel()
			if serviceLog.Val == "" {
				continue
			}
			// TODO Store service logs
			log.Info(ctx, "Service log: %s", serviceLog.Val)
		}
	}
}

func (s *Service) storeStepLogs(ctx context.Context, hm handledMessage) error {
	tx, err := s.Db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()

	apiRef := index.ApiRef{
		ProjectKey:     hm.Signature.ProjectKey,
		WorkflowName:   hm.Signature.WorkflowName,
		WorkflowID:     hm.Signature.WorkflowID,
		RunID:          hm.Signature.RunID,
		NodeRunName:    hm.Signature.NodeRunName,
		NodeRunID:      hm.Signature.NodeRunID,
		NodeRunJobName: hm.Signature.JobName,
		NodeRunJobID:   hm.Signature.JobID,
		StepName:       hm.Signature.Worker.StepName,
		StepOrder:      hm.Signature.Worker.StepOrder,
	}
	hashRefU, err := hashstructure.Hash(apiRef, nil)
	if err != nil {
		return sdk.WithStack(err)
	}
	hashRef := strconv.FormatUint(hashRefU, 10)

	item, err := index.LoadItemByApiRefHashAndType(ctx, s.Mapper, tx, hashRef, index.TypeItemStepLog)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		// Insert data
		item = &index.Item{
			ApiRef:     apiRef,
			Type:       index.TypeItemStepLog,
			ApiRefHash: hashRef,
			Status:     index.StatusItemIncoming,
		}
		if err := index.InsertItem(ctx, s.Mapper, tx, item); err != nil {
			if !sdk.ErrorIs(err, sdk.ErrConflictData) {
				return err
			}
			item, err = index.LoadItemByApiRefHashAndType(ctx, s.Mapper, tx, hashRef, index.TypeItemStepLog)
			if err != nil {
				return err
			}
		}
	}
	currentLog := buildMessage(hm.Signature, hm.Msg)

	// Do this before adding in buffer, to be able to rollback
	// If last log or update of a complete step
	if sdk.StatusIsTerminated(hm.Status) || item.Status == index.StatusItemCompleted {
		// In this case, we need to lock item.
		item, err = index.LoadAndLockItemByID(ctx, s.Mapper, tx, item.ID)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				return sdk.WrapError(sdk.ErrLocked, "item already locked")
			}
			return err
		}

		// Update index with final data
		item.Status = index.StatusItemCompleted

		// Get all data from buffer and add manually last line
		lines, err := s.StorageUnits.Buffer.Get(*item, cache.MIN, cache.MAX)
		if err != nil {
			return err
		}
		lines = append(lines, currentLog)
		buf := &bytes.Buffer{}
		gob.NewEncoder(buf).Encode(lines)
		h, err := convergent.NewHash(bytes.NewReader(buf.Bytes()))
		if err != nil {
			return err
		}
		item.Hash = hex.EncodeToString(h.Sum(nil))
		if err := index.UpdateItem(ctx, s.Mapper, tx, item); err != nil {
			return err
		}

		// TODO add association

	}

	if err := s.StorageUnits.Buffer.Add(*item, float64(hm.Line), currentLog); err != nil {
		return err
	}
	return sdk.WithStack(tx.Commit())
}
