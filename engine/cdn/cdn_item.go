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

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
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

func (s *Service) downloadItemFromUnit(ctx context.Context, t sdk.CDNItemType, apiRefHash string, unitName string, w http.ResponseWriter) error {
	// Load Item
	it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRefHash, t)
	if err != nil {
		return err
	}

	// Load Unit
	unit, err := storage.LoadUnitByName(ctx, s.Mapper, s.mustDBWithCtx(ctx), unitName)
	if err != nil {
		return err
	}

	// Load item unit
	itemUnit, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), unit.ID, it.ID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}

	// Get reader from unit
	source, err := s.Units.NewSource(ctx, *itemUnit)
	if err != nil {
		return err
	}

	reader, err := source.NewReader(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := reader.Close(); err != nil {
			log.Error(ctx, "downloadItemFromUnit> can't close reader: %+v", err)
		}
	}()

	if err := source.Read(reader, w); err != nil {
		return err
	}

	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", it.APIRef.ToFilename()))
	return nil
}

func (s *Service) downloadItem(ctx context.Context, t sdk.CDNItemType, apiRefHash string, w http.ResponseWriter, opts downloadOpts) error {
	ctx = context.WithValue(ctx, storage.FieldAPIRef, apiRefHash)

	if !t.IsLog() {
		return sdk.NewErrorFrom(sdk.ErrNotImplemented, "only log item can be download for now")
	}

	it, _, rc, filename, err := s.getItemLogValue(ctx, t, apiRefHash, getItemLogOptions{
		format: sdk.CDNReaderFormatText,
		from:   0,
		size:   0,
		sort:   opts.Log.Sort,
	})
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

	log.Info(ctx, "downloadItem> item %s has been downloaded", it.ID)

	return nil
}

type getItemLogOptions struct {
	format      sdk.CDNReaderFormat
	from        int64
	size        uint
	sort        int64
	cacheClean  bool
	cacheSource string
}

func (s *Service) getItemLogLinesCount(ctx context.Context, t sdk.CDNItemType, apiRefHash string) (int64, error) {
	ctx = context.WithValue(ctx, storage.FieldAPIRef, apiRefHash)

	it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRefHash, t)
	if err != nil {
		return 0, err
	}

	itemUnit, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.LogsBuffer().ID(), it.ID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return 0, err
	}

	// If item is in Buffer, get from it
	if itemUnit != nil {
		log.Debug(ctx, "getItemLogLines> Getting logs from buffer")
		lines, err := s.Units.LogsBuffer().Card(*itemUnit)
		return int64(lines), err
	}

	// Get from cache
	if ok, _ := s.LogCache.Exist(it.ID); !ok {
		log.Debug(ctx, "getItemLogLines> Getting logs from storage and push to cache")
		// Retrieve item and push it into the cache
		if err := s.pushItemLogIntoCache(ctx, *it, ""); err != nil {
			return 0, err
		}
	}

	linesCount, err := s.LogCache.Card(it.ID)
	return int64(linesCount), err
}

func (s *Service) getItemLogValue(ctx context.Context, t sdk.CDNItemType, apiRefHash string, opts getItemLogOptions) (*sdk.CDNItem, int64, io.ReadCloser, string, error) {
	ctx = context.WithValue(ctx, storage.FieldAPIRef, apiRefHash)

	it, err := item.LoadByAPIRefHashAndType(ctx, s.Mapper, s.mustDBWithCtx(ctx), apiRefHash, t)
	if err != nil {
		return nil, 0, nil, "", err
	}

	filename := it.APIRef.ToFilename()

	itemUnit, err := storage.LoadItemUnitByUnit(ctx, s.Mapper, s.mustDBWithCtx(ctx), s.Units.LogsBuffer().ID(), it.ID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil, 0, nil, "", err
	}

	// If item is in Buffer, get from it
	if itemUnit != nil {
		log.Debug(ctx, "getItemLogValue> Getting logs from buffer")
		linesCount, err := s.Units.LogsBuffer().Card(*itemUnit)
		if err != nil {
			return nil, 0, nil, "", err
		}

		rc, err := s.Units.LogsBuffer().NewAdvancedReader(ctx, *itemUnit, opts.format, opts.from, opts.size, opts.sort)
		if err != nil {
			return nil, 0, nil, "", err
		}

		return it, int64(linesCount), rc, filename, nil
	}

	if opts.cacheClean {
		if err := s.LogCache.Remove([]string{it.ID}); err != nil {
			return nil, 0, nil, "", err
		}
	}

	// Get from cache
	if ok, _ := s.LogCache.Exist(it.ID); !ok {
		log.Debug(ctx, "getItemLogValue> Getting logs from storage and push to cache")
		// Retrieve item and push it into the cache
		if err := s.pushItemLogIntoCache(ctx, *it, opts.cacheSource); err != nil {
			return nil, 0, nil, "", err
		}
	}

	linesCount, err := s.LogCache.Card(it.ID)
	if err != nil {
		return nil, 0, nil, "", err
	}

	log.Debug(ctx, "getItemLogValue> Getting logs from cache")
	return it, int64(linesCount), s.LogCache.NewReader(it.ID, opts.format, opts.from, opts.size, opts.sort), filename, nil
}

func (s *Service) pushItemLogIntoCache(ctx context.Context, it sdk.CDNItem, unitName string) error {
	ctx = context.WithValue(ctx, storage.FieldAPIRef, it.APIRefHash)

	// Search item in a storage unit
	itemUnits, err := storage.LoadAllItemUnitsByItemIDs(ctx, s.Mapper, s.mustDBWithCtx(ctx), it.ID)
	if err != nil {
		return err
	}

	if len(itemUnits) == 0 {
		return sdk.WithStack(fmt.Errorf("unable to find item units for item with id: %s", it.ID))
	}

	var unit *sdk.CDNUnit
	var selectedItemUnit *sdk.CDNItemUnit
	if unitName != "" {
		// Try to load the item from given unit
		unit, err = storage.LoadUnitByName(ctx, s.Mapper, s.mustDBWithCtx(ctx), unitName)
		if err != nil {
			return sdk.NewErrorFrom(err, "unit with name %s can't be loaded", unitName)
		}
		for i := range itemUnits {
			if itemUnits[i].UnitID == unit.ID {
				selectedItemUnit = &itemUnits[i]
				break
			}
		}
		if selectedItemUnit == nil {
			return sdk.NewErrorFrom(err, "cannot load item %s from given unit %s", it.ID, unitName)
		}
	} else {
		// Random pick a unit
		idx := 0
		if len(itemUnits) > 1 {
			idx = rnd.Intn(len(itemUnits))
		}
		selectedItemUnit = &itemUnits[idx]
		unit, err = storage.LoadUnitByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), selectedItemUnit.UnitID)
		if err != nil {
			return err
		}
	}

	// Get Storage unit
	unitStorage := s.Units.Storage(unit.Name)
	if unitStorage == nil {
		return sdk.WithStack(fmt.Errorf("unable to find unit %s", unit.Name))
	}

	selectedItemUnit, err = storage.LoadItemUnitByID(ctx, s.Mapper, s.mustDBWithCtx(ctx), selectedItemUnit.ID, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return err
	}

	// Create a reader
	storageReader, err := unitStorage.NewReader(ctx, *selectedItemUnit)
	if err != nil {
		return err
	}
	defer func() {
		if err := storageReader.Close(); err != nil {
			log.Error(ctx, "pushItemLogIntoCache> can't close reader: %+v", err)
		}
	}()

	// Create a writer for the cache
	cacheWriter := s.LogCache.NewWriter(it.ID)
	defer func() {
		if err := cacheWriter.Close(); err != nil {
			log.Error(ctx, "pushItemLogIntoCache> can't close writer: %+v", err)
		}
	}()

	// Write data in cache
	if err := unitStorage.Read(*selectedItemUnit, storageReader, cacheWriter); err != nil {
		return err
	}

	log.Info(ctx, "item %s has been pushed to cache", it.ID)

	return nil
}

func (s *Service) completeItem(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, itemUnit sdk.CDNItemUnit) error {
	// We need to lock the item and set its status to complete and also generate data hash
	it, err := item.LoadAndLockByID(ctx, s.Mapper, tx, itemUnit.ItemID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WrapError(sdk.ErrLocked, "item already locked")
		}
		return err
	}

	ctx = context.WithValue(ctx, storage.FieldAPIRef, it.APIRefHash)

	// Update item with final data
	it.Status = sdk.CDNStatusItemCompleted

	var reader io.ReadCloser
	for _, unit := range s.Units.Buffers {
		if unit.ID() == itemUnit.UnitID {
			reader, err = unit.NewReader(ctx, itemUnit)
			if err != nil {
				return err
			}
			break
		}
	}
	if reader == nil {
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

	log.Info(ctx, "completeItem> item %s has been completed", it.ID)

	return nil
}
