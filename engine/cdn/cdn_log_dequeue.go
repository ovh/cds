package cdn

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/symmecrypt/convergent"
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
			currentLog := buildMessage(hm.Signature, hm.Msg)
			cpt := 0
			for {
				if err := s.storeLogs(ctx, index.TypeItemStepLog, hm.Signature, hm.Status, currentLog, hm.Line); err != nil {
					if sdk.ErrorIs(err, sdk.ErrLocked) && cpt < 10 {
						cpt++
						time.Sleep(250 * time.Millisecond)
						continue
					}
					log.Error(ctx, "dequeueJobLogs: unable to store step log: %v", err)
					break
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
			if err := s.storeLogs(ctx, index.TypeItemServiceLog, hm.Signature, hm.Status, hm.Msg.Full, 0); err != nil {
				log.Error(ctx, "dequeueServiceLogs: unable to store service log: %v", err)
			}
		}
	}
}

func (s *Service) storeLogs(ctx context.Context, typ string, signature log.Signature, status string, content string, line int64) error {
	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// Build cds api ref
	apiRef := index.ApiRef{
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

	hashRefU, err := hashstructure.Hash(apiRef, nil)
	if err != nil {
		return sdk.WithStack(err)
	}
	hashRef := strconv.FormatUint(hashRefU, 10)

	item, err := index.LoadItemByApiRefHashAndType(ctx, s.Mapper, tx, hashRef, typ)
	if err != nil {
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		// Insert data
		item = &index.Item{
			ApiRef:     apiRef,
			Type:       typ,
			ApiRefHash: hashRef,
			Status:     index.StatusItemIncoming,
		}
		if err := index.InsertItem(ctx, s.Mapper, tx, item); err != nil {
			if !sdk.ErrorIs(err, sdk.ErrConflictData) {
				return err
			}
			// reload if item already exist
			item, err = index.LoadItemByApiRefHashAndType(ctx, s.Mapper, tx, hashRef, typ)
			if err != nil {
				return err
			}
		}
	}

	switch typ {
	case index.TypeItemStepLog:
		if err := s.Units.Buffer.Add(*item, uint(line), content); err != nil {
			return err
		}
	case index.TypeItemServiceLog:
		if err := s.Units.Buffer.Append(*item, content); err != nil {
			return err
		}
	}

	// If last log or update of a complete step
	if sdk.StatusIsTerminated(status) || item.Status == index.StatusItemCompleted {
		alreadyExist := item.Status == index.StatusItemCompleted

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
		reader, err := s.Units.Buffer.NewReader(*item)
		if err != nil {
			return err
		}

		h, err := convergent.NewHash(reader)
		if err != nil {
			return err
		}
		item.Hash = hex.EncodeToString(h.Sum(nil))

		if err := index.UpdateItem(ctx, s.Mapper, tx, item); err != nil {
			return err
		}

		unit, err := storage.LoadUnitByName(ctx, s.Mapper, tx, s.Units.Buffer.Name())
		if err != nil {
			return err
		}

		if !alreadyExist {
			if _, err := storage.InsertItemUnit(ctx, s.Mapper, tx, *unit, *item); err != nil {
				return err
			}
		}
	}

	return sdk.WithStack(tx.Commit())

}
