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
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func (s *Service) completeItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, itemUnit storage.ItemUnit) error {
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
	md5Hash := md5.New()
	sha512Hash := sha512.New()
	// For optimum speed, Getpagesize returns the underlying system's memory page size.
	pagesize := os.Getpagesize()
	// wraps the Reader object into a new buffered reader to read the files in chunks
	// and buffering them for performance.
	mreader := bufio.NewReaderSize(reader, pagesize)
	multiWriter := io.MultiWriter(md5Hash, sha512Hash)
	size, err := io.Copy(multiWriter, mreader)
	if err != nil {
		return sdk.WithStack(err)
	}

	sha512S := hex.EncodeToString(sha512Hash.Sum(nil))
	md5S := hex.EncodeToString(md5Hash.Sum(nil))

	item.Hash = sha512S
	item.MD5 = md5S
	item.Size = size

	if err := index.UpdateItem(ctx, s.Mapper, tx, item); err != nil {
		return err
	}

	return nil
}
