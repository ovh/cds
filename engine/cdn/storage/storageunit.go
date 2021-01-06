package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (r RunningStorageUnits) Storage(name string) StorageUnit {
	for _, x := range r.Storages {
		if x.Name() == name {
			return x
		}
	}
	return nil
}

func Init(ctx context.Context, m *gorpmapper.Mapper, db *gorp.DbMap, gorts *sdk.GoRoutines, config Configuration) (*RunningStorageUnits, error) {
	for i := range config.Storages {
		if config.Storages[i].SyncParallel <= 0 {
			config.Storages[i].SyncParallel = 1
		}
	}

	var result = RunningStorageUnits{
		m:      m,
		db:     db,
		config: config,
	}

	if len(config.HashLocatorSalt) < 8 {
		return nil, fmt.Errorf("invalid CDN configuration. HashLocatorSalt is too short")
	}

	countLogBuffer := 0
	for _, bu := range config.Buffers {
		if bu.BufferType == CDNBufferTypeLog {
			countLogBuffer++
		}
		if bu.Name == "" {
			return nil, fmt.Errorf("invalid CDN configuration. Missing buffer name")
		}
	}
	if countLogBuffer == 0 || countLogBuffer > 1 {
		return nil, fmt.Errorf("missing or too much CDN Buffer for log items")
	}

	if len(config.Storages) == 0 {
		return nil, fmt.Errorf("invalid CDN configuration. Missing storage unit")
	}

	if config.SyncNbElements < 0 || config.SyncNbElements > 1000 {
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
				return nil, fmt.Errorf("redis driver is not available")
			}
			bd, is := d.(BufferUnit)
			if !is {
				return nil, fmt.Errorf("redis driver is not a buffer unit driver")
			}
			bd.New(gorts, 1, math.MaxFloat64)
			if err := bd.Init(ctx, bu.Redis, bu.BufferType); err != nil {
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

func (r *RunningStorageUnits) Start(ctx context.Context, gorts *sdk.GoRoutines, store cache.Store) {
	for i := range r.Storages {
		s := r.Storages[i]
		// Start the sync processes
		for x := 0; x < cap(s.SyncItemChannel()); x++ {
			gorts.Run(ctx, fmt.Sprintf("RunningStorageUnits.process.%s.%d", s.Name(), x),
				func(ctx context.Context) {
					for id := range s.SyncItemChannel() {
						t0 := time.Now()
						tx, err := r.db.Begin()
						if err != nil {
							err = sdk.WrapError(err, "unable to begin tx")
							log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
							continue
						}

						if err := r.processItem(ctx, tx, s, id); err != nil {
							t1 := time.Now()
							log.ErrorWithFields(ctx, log.Fields{
								"stack_trace":               fmt.Sprintf("%+v", err),
								"duration_milliseconds_num": t1.Sub(t0).Milliseconds(),
							}, "error processing item id=%q: %v", id, err)
							_ = tx.Rollback()
							continue
						}

						if err := tx.Commit(); err != nil {
							err = sdk.WrapError(err, "unable to commit tx")
							log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
							_ = tx.Rollback()
							continue
						}

						// Remove from redis
						k := cache.Key(KeyBackendSync, s.Name())
						bts, _ := json.Marshal(id)
						if err := store.ScoredSetRem(ctx, k, string(bts)); err != nil {
							err = sdk.WrapError(err, "unable to remove sync item %s from redis %s", id, k)
							log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
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
							if err := r.FillSyncItemChannel(ctx, store, s, r.config.SyncNbElements); err != nil {
								log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "RunningStorageUnits.run> error: %v", err)
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
								log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "RunningStorageUnits.purge> error: %v", err)
							}
						},
					)
				}

				for i := range r.Storages {
					s := r.Storages[i]
					gorts.Exec(ctx, "RunningStorageUnits.purge."+s.Name(),
						func(ctx context.Context) {
							if err := r.Purge(ctx, s); err != nil {
								log.ErrorWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "RunningStorageUnits.purge> error: %v", err)
							}
						},
					)
				}
			}
		}

	})
}
