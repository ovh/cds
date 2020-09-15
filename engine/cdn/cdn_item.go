package cdn

import (
	"bufio"
	"context"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/redis"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	rs  = rand.NewSource(time.Now().Unix())
	rnd = rand.New(rs)
)

func (s *Service) getItemLogValue(ctx context.Context, t sdk.CDNItemType, apiRefHash string, from uint, size int) (io.ReadCloser, error) {
	item, err := index.LoadItemByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRefHash, t)
	if err != nil {
		return nil, err
	}

	itemUnit, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Buffer.ID(), item.ID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil, err
	}

	// If item is in Buffer, get from it
	if itemUnit != nil {
		log.Debug("Getting logs from buffer")
		rc, err := s.Units.Buffer.NewReader(*itemUnit)
		if err != nil {
			return nil, err
		}
		redisReader, ok := rc.(*redis.Reader)
		if ok {
			redisReader.From = from
			redisReader.Size = size
		}
		return redisReader, nil
	}

	// Get from cache
	ok, _ := s.LogCache.Exist(item.ID)
	if ok {
		log.Debug("Getting logs from cache")
		return s.LogCache.NewReader(item.ID, from, size), nil
	}

	log.Debug("Getting logs from storage")
	// Retrieve item and push it into the cache
	if err := s.pushItemLogIntoCache(ctx, *item); err != nil {
		return nil, err
	}

	// Get from cache
	return s.LogCache.NewReader(item.ID, from, size), nil
}

func (s *Service) pushItemLogIntoCache(ctx context.Context, item index.Item) error {
	// Search item in a storage unit
	itemUnits, err := storage.LoadAllItemUnitsByItemID(ctx, s.Mapper, s.mustDBWithCtx(ctx), item.ID)
	if err != nil {
		return err
	}
	// Random pick a unit
	idx := 0
	if len(itemUnits) > 1 {
		idx = rnd.Intn(len(itemUnits))
	}
	refItemUnit, err := storage.LoadItemUnitByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), itemUnits[idx].ID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}

	// Load Unit
	unit, err := storage.LoadUnitByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), refItemUnit.UnitID)
	if err != nil {
		return err
	}

	// Get Storage unit
	unitStorage := s.Units.Storage(unit.Name)
	if unitStorage == nil {
		return sdk.WithStack(fmt.Errorf("unable to find unit %s", unit.Name))
	}

	// Create a reader
	reader, err := unitStorage.NewReader(*refItemUnit)
	if err != nil {
		return err
	}

	// Create a writer for the cache
	cacheWriter := s.LogCache.NewWriter(item.ID)

	// Write data in cache
	chanError := make(chan error)
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		if err := unitStorage.Read(*refItemUnit, reader, pw); err != nil {
			chanError <- err
		}
		close(chanError)
	}()
	if _, err := io.Copy(cacheWriter, pr); err != nil {
		return err
	}

	if err := reader.Close(); err != nil {
		return sdk.WithStack(err)
	}

	if err := cacheWriter.Close(); err != nil {
		return sdk.WithStack(err)
	}

	if err := pr.Close(); err != nil {
		return sdk.WithStack(err)
	}

	for err := range chanError {
		if err != nil {
			return sdk.WithStack(err)
		}
	}

	log.Info(ctx, "log %s has been pushed to cache", item.ID)
	return nil
}

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
	switch itemUnit.UnitID {
	case s.Units.Buffer.ID():
		// Get all data from buffer and add manually last line
		reader, err = s.Units.Buffer.NewReader(itemUnit)
		if err != nil {
			return err
		}
	default:
		for _, unit := range s.Units.Storages {
			if unit.ID() == itemUnit.UnitID {
				reader, err = unit.NewReader(itemUnit)
				if err != nil {
					return err
				}
				break
			}
		}
		if reader == nil {
			return sdk.WithStack(fmt.Errorf("unable to find unit storage %s", itemUnit.ID))
		}
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
