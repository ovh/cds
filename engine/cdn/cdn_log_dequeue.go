package cdn

import (
	"bytes"
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"
	"github.com/ovh/symmecrypt/convergent"

	"github.com/ovh/cds/engine/cache"
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

			if err := s.storeStepLogs(ctx, hm); err != nil {
				log.Error(ctx, "dequeueJobLogs: unable to store step log: %v", err)
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
		item := &index.Item{
			ApiRef:     apiRef,
			Type:       index.TypeItemStepLog,
			ApiRefHash: hashRef,
			Status:     index.StatusItemIncoming,
		}
		if err := index.InsertItem(ctx, s.Mapper, tx, item); err != nil {
			if !sdk.ErrorIs(err, sdk.ErrConflictData) {
				return err
			}
		}
	}

	// Use buffer backend previously loaded to store data
	currentLog := buildMessage(hm.Signature, hm.Msg)
	jobString := strconv.FormatInt(hm.Signature.JobID, 10)
	stepString := strconv.FormatInt(hm.Signature.Worker.StepOrder, 10)

	jobStepKey := cache.Key(keyStoreJobPrefix, jobString, "step", stepString)

	// FIXME DO NOT USE s.CACHE but a REDIS BACKEND
	if err := s.Cache.ScoredSetAdd(ctx, jobStepKey, currentLog, float64(hm.Line)); err != nil {
		return err
	}

	item, err = index.LoadItemByApiRefHashAndType(ctx, s.Mapper, tx, hashRef, index.TypeItemStepLog)
	if err != nil {
		return err
	}

	// If last log or update of a complete step
	if sdk.StatusIsTerminated(hm.Status) || item.Status == index.StatusItemCompleted {
		// Update index with final data
		item.Status = index.StatusItemCompleted

		// TODO Get all data from redis to compute hash.
		var data []byte

		h, err := convergent.NewHash(bytes.NewReader(data))
		if err != nil {
			return err
		}
		item.Hash = hex.EncodeToString(h.Sum(nil))
		if err := index.UpdateItem(ctx, s.Mapper, tx, item); err != nil {
			return err
		}
	}
	log.Info(ctx, "Job log: %s", currentLog)

	return sdk.WithStack(tx.Commit())
}
