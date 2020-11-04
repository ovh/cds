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
	"net/http"
	"os"
	"time"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	rs  = rand.NewSource(time.Now().Unix())
	rnd = rand.New(rs)
)

type downloadOpts struct {
	Log struct {
		Sort    int64
		Refresh int64
	}
}

func (s *Service) downloadItem(ctx context.Context, t sdk.CDNItemType, apiRefHash string, w http.ResponseWriter, opts downloadOpts) error {
	t0 := time.Now()

	if !t.IsLog() {
		return sdk.NewErrorFrom(sdk.ErrNotImplemented, "only log item can be download for now")
	}

	it, _, rc, filename, err := s.getItemLogValue(ctx, t, apiRefHash, sdk.CDNReaderFormatText, 0, 0, opts.Log.Sort)
	if err != nil {
		return err
	}
	if rc == nil {
		return sdk.WrapError(sdk.ErrNotFound, "no storage found that contains given item %s", apiRefHash)
	}
	w.Header().Add("Content-Type", "text/plain")
	if it.Status != sdk.CDNStatusItemCompleted && opts.Log.Refresh > 0 {
		// This will allows to refresh the browser when opening the logs int a new tab
		w.Header().Add("Refresh", fmt.Sprintf("%d", opts.Log.Refresh))
	}
	w.Header().Add("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

	if _, err := io.Copy(w, rc); err != nil {
		return sdk.WithStack(err)
	}
	t1 := time.Now()

	log.InfoWithFields(ctx, log.Fields{
		"item_apiref":               it.APIRefHash,
		"duration_milliseconds_num": t1.Sub(t0).Milliseconds(),
	}, "downloadItem> item %s has been downloaded", it.ID)

	return nil
}

func (s *Service) getItemLogValue(ctx context.Context, t sdk.CDNItemType, apiRefHash string, format sdk.CDNReaderFormat, from int64, size uint, sort int64) (*sdk.CDNItem, int64, io.ReadCloser, string, error) {
	it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRefHash, t)
	if err != nil {
		return nil, 0, nil, "", err
	}

	filename := it.APIRef.ToFilename()

	itemUnit, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.Buffer.ID(), it.ID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil, 0, nil, "", err
	}

	// If item is in Buffer, get from it
	if itemUnit != nil {
		log.Debug("getItemLogValue> Getting logs from buffer")
		linesCount, err := s.Units.Buffer.Card(*itemUnit)
		if err != nil {
			return nil, 0, nil, "", err
		}

		rc, err := s.Units.Buffer.NewAdvancedReader(ctx, *itemUnit, format, from, size, sort)
		if err != nil {
			return nil, 0, nil, "", err
		}

		return it, int64(linesCount), rc, filename, nil
	}

	// Get from cache
	if ok, _ := s.LogCache.Exist(it.ID); !ok {
		log.Debug("getItemLogValue> Getting logs from storage")
		// Retrieve item and push it into the cache
		if err := s.pushItemLogIntoCache(ctx, *it); err != nil {
			return nil, 0, nil, "", err
		}
	}

	linesCount, err := s.LogCache.Card(it.ID)
	if err != nil {
		return nil, 0, nil, "", err
	}

	log.Debug("getItemLogValue> Getting logs from cache")
	return it, int64(linesCount), s.LogCache.NewReader(it.ID, format, from, size, sort), filename, nil
}

func (s *Service) pushItemLogIntoCache(ctx context.Context, it sdk.CDNItem) error {
	t0 := time.Now()
	// Search item in a storage unit
	mapItemUnits, err := storage.LoadAllItemUnitsByItemIDs(ctx, s.Mapper, s.mustDBWithCtx(ctx), []string{it.ID})
	if err != nil {
		return err
	}
	itemUnits, has := mapItemUnits[it.ID]
	if !has {
		return sdk.WithStack(fmt.Errorf("unable to find item units"))
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

	t1 := time.Now()

	// Create a reader
	storageReader, err := unitStorage.NewReader(ctx, *refItemUnit)
	if err != nil {
		return err
	}
	defer storageReader.Close()

	// Create a writer for the cache
	cacheWriter := s.LogCache.NewWriter(it.ID)
	defer cacheWriter.Close()

	// Write data in cache
	if err := unitStorage.Read(*refItemUnit, storageReader, cacheWriter); err != nil {
		return err
	}

	log.InfoWithFields(ctx, log.Fields{
		"item_apiref":               it.APIRefHash,
		"duration_milliseconds_num": t1.Sub(t0).Milliseconds(),
	}, "item %s has been pushed to cache", it.ID)

	return nil
}

func (s *Service) completeItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, itemUnit sdk.CDNItemUnit) error {
	t0 := time.Now()
	// We need to lock the item and set its status to complete and also generate data hash
	it, err := item.LoadAndLockByID(ctx, s.Mapper, tx, itemUnit.ItemID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WrapError(sdk.ErrLocked, "item already locked")
		}
		return err
	}

	// Update item with final data
	it.Status = sdk.CDNStatusItemCompleted

	var reader io.ReadCloser
	switch itemUnit.UnitID {
	case s.Units.Buffer.ID():
		// Get all data from buffer and add manually last line
		reader, err = s.Units.Buffer.NewReader(ctx, itemUnit)
		if err != nil {
			return err
		}
	default:
		for _, unit := range s.Units.Storages {
			if unit.ID() == itemUnit.UnitID {
				reader, err = unit.NewReader(ctx, itemUnit)
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
		_ = reader.Close()
		return sdk.WithStack(err)
	}
	if err := reader.Close(); err != nil {
		return sdk.WithStack(err)
	}

	sha512S := hex.EncodeToString(sha512Hash.Sum(nil))
	md5S := hex.EncodeToString(md5Hash.Sum(nil))

	it.Hash = sha512S
	it.MD5 = md5S
	it.Size = size

	if err := item.Update(ctx, s.Mapper, tx, it); err != nil {
		return err
	}

	t1 := time.Now()

	log.InfoWithFields(ctx, log.Fields{
		"item_apiref":               it.APIRefHash,
		"duration_milliseconds_num": t1.Sub(t0).Milliseconds(),
		"item_size_num":             it.Size,
	}, "completeItem> item %s has been completed", it.ID)

	return nil
}
