package cdn

import (
	"context"
	"encoding/hex"
	"io"

	"github.com/ovh/symmecrypt/convergent"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
)

func (s *Service) completeItem(ctx context.Context, itemID string) error {
	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// We need to lock the item and set its status to complete and also generate data hash
	item, err := index.LoadAndLockItemByID(ctx, s.Mapper, tx, itemID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WrapError(sdk.ErrLocked, "item already locked")
		}
		return err
	}

	// Update index with final data
	item.Status = index.StatusItemCompleted

	var reader io.ReadCloser

	switch item.Type {
	case index.TypeItemStepLog, index.TypeItemServiceLog:
		// Get all data from buffer and add manually last line
		reader, err = s.Units.Buffer.NewReader(*item)
		if err != nil {
			return err
		}
	default:
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to complete item: unknown item type %s", item.Type)
	}

	h, err := convergent.NewHash(reader)
	if err != nil {
		return err
	}
	item.Hash = hex.EncodeToString(h.Sum(nil))

	if err := index.UpdateItem(ctx, s.Mapper, tx, item); err != nil {
		return err
	}

	if _, err := storage.InsertItemUnit(ctx, s.Mapper, tx, s.Units.Buffer.ID(), item.ID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
