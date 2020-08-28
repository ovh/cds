package cdn

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
)

func (s *Service) completeItem(ctx context.Context, itemUnit storage.ItemUnit) error {
	tx, err := s.mustDBWithCtx(ctx).Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// We need to lock the item and set its status to complete and also generate data hash
	item, err := index.LoadAndLockItemByID(ctx, s.Mapper, tx, itemUnit.ItemID)
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
		reader, err = s.Units.Buffer.NewReader(itemUnit)
		if err != nil {
			return err
		}
	default:
		return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to complete item: unknown item type %s", item.Type)

	}

	// Compte md5 and sha512
	md5 := md5.New()
	sha512 := sha512.New()
	// For optimum speed, Getpagesize returns the underlying system's memory page size.
	pagesize := os.Getpagesize()
	// wraps the Reader object into a new buffered reader to read the files in chunks
	// and buffering them for performance.
	mreader := bufio.NewReaderSize(reader, pagesize)
	multiWriter := io.MultiWriter(md5, sha512)
	size, err := io.Copy(multiWriter, mreader)
	if err != nil {
		return sdk.WithStack(err)
	}

	sha512Hash := hex.EncodeToString(sha512.Sum(nil))
	md5Hash := hex.EncodeToString(md5.Sum(nil))

	item.Hash = sha512Hash
	item.MD5 = md5Hash
	item.Size = size

	if err := index.UpdateItem(ctx, s.Mapper, tx, item); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
