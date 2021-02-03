package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (r RunningStorageUnits) Storage(name string) StorageUnit {
	for _, x := range r.Storages {
		if x.Name() == name {
			return x
		}
	}
	return nil
}

func Init(ctx context.Context, m *gorpmapper.Mapper, store cache.Store, db *gorp.DbMap, gorts *sdk.GoRoutines, config Configuration) (*RunningStorageUnits, error) {
	for i := range config.Storages {
		if config.Storages[i].SyncParallel <= 0 {
			config.Storages[i].SyncParallel = 1
		}
	}

	var result = RunningStorageUnits{
		m:      m,
		db:     db,
		cache:  store,
		config: config,
	}

	if len(config.HashLocatorSalt) < 8 {
		return nil, sdk.WithStack(fmt.Errorf("invalid CDN configuration. HashLocatorSalt is too short"))
	}

	countLogBuffer := 0
	countFileBuffer := 0
	for _, bu := range config.Buffers {
		switch bu.BufferType {
		case CDNBufferTypeLog:
			countLogBuffer++
		case CDNBufferTypeFile:
			countFileBuffer++
		}
		if bu.Name == "" {
			return nil, sdk.WithStack(fmt.Errorf("invalid CDN configuration. Missing buffer name"))
		}
	}
	if countLogBuffer == 0 || countLogBuffer > 1 {
		return nil, sdk.WithStack(fmt.Errorf("missing or too much CDN Buffer for log items"))
	}
	if countFileBuffer > 1 {
		return nil, sdk.WithStack(fmt.Errorf("too much CDN Buffer for file items"))
	}

	if len(config.Storages) == 0 {
		return nil, sdk.WithStack(fmt.Errorf("invalid CDN configuration. Missing storage unit"))
	}

	if config.SyncNbElements <= 0 || config.SyncNbElements > 1000 {
		config.SyncNbElements = 100
	}

	if config.SyncSeconds <= 0 {
		config.SyncSeconds = 30
	}

	for _, bu := range config.Buffers {
		var bufferUnit BufferUnit
		switch {
		case bu.Redis != nil:
			// Start by initializing the buffer unit
			d := GetDriver("redis")
			if d == nil {
				return nil, sdk.WithStack(fmt.Errorf("redis driver is not available"))
			}
			bd, is := d.(BufferUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("redis driver is not a buffer unit driver"))
			}
			bd.New(gorts, 1, math.MaxFloat64)
			if err := bd.Init(ctx, bu.Redis, bu.BufferType); err != nil {
				return nil, err
			}
			bufferUnit = bd
		case bu.Local != nil:
			d := GetDriver("local-buffer")
			if d == nil {
				return nil, sdk.WithStack(fmt.Errorf("local driver is not available"))
			}
			bd, is := d.(BufferUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("local driver is not a buffer unit driver"))
			}
			bd.New(gorts, 1, math.MaxFloat64)
			if err := bd.Init(ctx, bu.Local, bu.BufferType); err != nil {
				return nil, err
			}
			bufferUnit = bd
		default:
			return nil, sdk.WithStack(errors.New("unsupported buffer units"))
		}

		result.Buffers = append(result.Buffers, bufferUnit)
		tx, err := db.Begin()
		if err != nil {
			return nil, err
		}
		defer tx.Rollback() // nolint

		u, err := LoadUnitByName(ctx, m, tx, bu.Name)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			var srvConfig sdk.ServiceConfig
			b, _ := json.Marshal(bu)
			_ = json.Unmarshal(b, &srvConfig) // nolint
			u = &sdk.CDNUnit{
				ID:      sdk.UUID(),
				Created: time.Now(),
				Name:    bu.Name,
				Config:  srvConfig,
			}
			if err := InsertUnit(ctx, m, tx, u); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		bufferUnit.Set(*u)

		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}

	// Then initialize the storages unit
	for _, cfg := range config.Storages {
		if cfg.Name == "" {
			return nil, sdk.WithStack(fmt.Errorf("invalid CDN configuration. Missing storage name"))
		}

		var storageUnit StorageUnit
		switch {
		case cfg.CDS != nil:
			d := GetDriver("cds")
			if d == nil {
				return nil, sdk.WithStack(fmt.Errorf("cds driver is not available"))
			}
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("cds driver is not a storage unit driver"))
			}
			sd.New(gorts, cfg.SyncParallel, float64(cfg.SyncBandwidth)*1024*1024) // convert from MBytes to Bytes

			if err := sd.Init(ctx, cfg.CDS); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Local != nil:
			d := GetDriver("local")
			if d == nil {
				return nil, sdk.WithStack(fmt.Errorf("local driver is not available"))
			}
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("local driver is not a storage unit driver"))
			}
			sd.New(gorts, cfg.SyncParallel, float64(cfg.SyncBandwidth)*1024*1024) // convert from MBytes to Bytes

			if err := sd.Init(ctx, cfg.Local); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Swift != nil:
			d := GetDriver("swift")
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("swift driver is not a storage unit driver"))
			}
			sd.New(gorts, cfg.SyncParallel, float64(cfg.SyncBandwidth)*1024*1024) // convert from MBytes to Bytes

			if err := sd.Init(ctx, cfg.Swift); err != nil {
				return nil, err
			}
			storageUnit = sd
		case cfg.Webdav != nil:
			d := GetDriver("webdav")
			sd, is := d.(StorageUnit)
			if !is {
				return nil, sdk.WithStack(fmt.Errorf("webdav driver is not a storage unit driver"))
			}
			sd.New(gorts, cfg.SyncParallel, float64(cfg.SyncBandwidth)*1024*1024) // convert from MBytes to Bytes

			if err := sd.Init(ctx, cfg.Webdav); err != nil {
				return nil, err
			}
			storageUnit = sd
		default:
			return nil, sdk.WithStack(errors.New("unsupported storage unit"))
		}

		tx, err := db.Begin()
		if err != nil {
			return nil, sdk.WithStack(err)
		}

		u, err := LoadUnitByName(ctx, m, tx, cfg.Name)
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			var srvConfig sdk.ServiceConfig
			b, _ := json.Marshal(cfg)
			_ = json.Unmarshal(b, &srvConfig) // nolint

			u = &sdk.CDNUnit{
				ID:      sdk.UUID(),
				Created: time.Now(),
				Name:    cfg.Name,
				Config:  srvConfig,
			}
			err = InsertUnit(ctx, m, tx, u)
		}
		if err != nil {
			_ = tx.Rollback() // nolint
			return nil, err
		}
		storageUnit.Set(*u)

		result.Storages = append(result.Storages, storageUnit)
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback() // nolint
			return nil, sdk.WithStack(err)
		}
	}

	return &result, nil
}

func (r *RunningStorageUnits) PushInSyncQueue(ctx context.Context, itemID string, created time.Time) {
	for _, sto := range r.Storages {
		if err := r.cache.ScoredSetAdd(ctx, cache.Key(KeyBackendSync, sto.Name()), itemID, float64(created.Unix())); err != nil {
			log.Info(ctx, "storeLogs> cannot push item %s into scoredset for unit %s", itemID, sto.Name())
			continue
		}
	}
}

func (r *RunningStorageUnits) Start(ctx context.Context, gorts *sdk.GoRoutines) {
	// Get Unknown items
	for _, s := range r.Storages {
		if err := r.FillWithUnknownItems(ctx, s, r.config.SyncNbElements); err != nil {
			log.Error(ctx, "Start> unable to get unknown items: %v", err)
		}
	}

	// Start the sync processes
	for i := range r.Storages {
		s := r.Storages[i]
		for x := 0; x < cap(s.SyncItemChannel()); x++ {
			gorts.Run(ctx, fmt.Sprintf("RunningStorageUnits.process.%s.%d", s.Name(), x),
				func(ctx context.Context) {
					for id := range s.SyncItemChannel() {
						log.Debug(ctx, "processItem: %s", id)
						for {
							lockKey := cache.Key("cdn", "backend", "lock", "sync", s.Name())
							if b, err := r.cache.Exist(lockKey); err != nil || b {
								log.Info(ctx, "RunningStorageUnits.Start.%s > waiting for processItem %s: %v", s.Name(), id, err)
								time.Sleep(30 * time.Second)
								continue
							}
							break
						}
						t0 := time.Now()
						tx, err := r.db.Begin()
						if err != nil {
							err = sdk.WrapError(err, "unable to begin tx")
							ctx = sdk.ContextWithStacktrace(ctx, err)
							log.Error(ctx, "%v", err)
							continue
						}

						if err := r.processItem(ctx, tx, s, id); err != nil {
							if !sdk.ErrorIs(err, sdk.ErrNotFound) {
								t1 := time.Now()
								ctx = sdk.ContextWithStacktrace(ctx, err)
								ctx = context.WithValue(ctx, cdslog.Duration, t1.Sub(t0).Milliseconds())
								log.Error(ctx, "error processing item id=%q: %v", id, err)
							} else {
								log.Info(ctx, "item id=%q is locked", id)
							}
							_ = tx.Rollback()
							continue
						}

						if err := tx.Commit(); err != nil {
							err = sdk.WrapError(err, "unable to commit tx")
							ctx = sdk.ContextWithStacktrace(ctx, err)
							log.Error(ctx, "%v", err)
							_ = tx.Rollback()
							continue
						}

						// Remove from redis
						k := cache.Key(KeyBackendSync, s.Name())
						bts, _ := json.Marshal(id)
						if err := r.cache.ScoredSetRem(ctx, k, string(bts)); err != nil {
							err = sdk.WrapError(err, "unable to remove sync item %s from redis %s", id, k)
							ctx = sdk.ContextWithStacktrace(ctx, err)
							log.Error(ctx, "%v", err)
						}
					}
				},
			)
		}
	}

	// 	Feed the sync processes with a ticker
	gorts.Run(ctx, "RunningStorageUnits.Start", func(ctx context.Context) {
		tickr := time.NewTicker(time.Duration(r.config.SyncSeconds) * time.Second)
		tickrPurge := time.NewTicker(30 * time.Second)

		defer tickr.Stop()
		defer tickrPurge.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tickr.C:
				wg := sync.WaitGroup{}
				for i := range r.Storages {
					s := r.Storages[i]
					gorts.Exec(ctx, "RunningStorageUnits.run."+s.Name(),
						func(ctx context.Context) {
							wg.Add(1)
							if err := r.FillSyncItemChannel(ctx, s, r.config.SyncNbElements); err != nil {
								ctx = sdk.ContextWithStacktrace(ctx, err)
								log.Error(ctx, "RunningStorageUnits.run> error: %v", err)
							}
							wg.Done()
						},
					)
				}
				wg.Wait()
			case <-tickrPurge.C:
				for i := range r.Buffers {
					b := r.Buffers[i]
					gorts.Exec(ctx, "RunningStorageUnits.purge."+b.Name(),
						func(ctx context.Context) {
							if err := r.Purge(ctx, b); err != nil {
								ctx = sdk.ContextWithStacktrace(ctx, err)
								log.Error(ctx, "RunningStorageUnits.purge> error: %v", err)
							}
						},
					)
				}

				for i := range r.Storages {
					s := r.Storages[i]
					gorts.Exec(ctx, "RunningStorageUnits.purge."+s.Name(),
						func(ctx context.Context) {
							if err := r.Purge(ctx, s); err != nil {
								ctx = sdk.ContextWithStacktrace(ctx, err)
								log.Error(ctx, "RunningStorageUnits.purge> error: %v", err)
							}
						},
					)
				}
			}
		}

	})
}

func (r *RunningStorageUnits) SyncBuffer(ctx context.Context) {
	log.Info(ctx, "[SyncBuffer] Start")
	keysDeleted := 0
	bu := r.LogsBuffer()

	keys, err := bu.Keys()
	if err != nil {
		log.Error(ctx, "[SyncBuffer] unable to list keys: %v", err)
		return
	}
	log.Info(ctx, "[SyncBuffer] Found %d keys", len(keys))

	for _, k := range keys {
		keySplitted := strings.Split(k, ":")
		if len(keySplitted) != 3 {
			continue
		}
		itemID := keySplitted[2]
		_, err := LoadItemUnitByUnit(ctx, r.m, r.db, bu.ID(), itemID)
		if err == nil {
			log.Info(ctx, "[SyncBuffer] Item %s exists in database ", itemID)
			continue
		}
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			if err := bu.Remove(ctx, sdk.CDNItemUnit{ItemID: itemID}); err != nil {
				log.Error(ctx, "[SyncBuffer] unable to remove item %s from buffer: %v", itemID, err)
				continue
			}
			keysDeleted++
			log.Info(ctx, "[SyncBuffer] item %s remove from redis", itemID)
		} else {
			log.Error(ctx, "[SyncBuffer] unable to load item %s: %v", itemID, err)
		}
	}
	log.Info(ctx, "[SyncBuffer] Done - %d keys deleted", keysDeleted)

}
