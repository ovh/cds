package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/sirupsen/logrus"

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

	if config.Buffer.Name == "" {
		return nil, fmt.Errorf("invalid CDN configuration. Missing buffer name")
	}

	if len(config.Storages) == 0 {
		return nil, fmt.Errorf("invalid CDN configuration. Missing storage unit")
	}

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

	if err := bd.Init(ctx, config.Buffer.Redis); err != nil {
		return nil, err
	}
	result.Buffer = bd

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // nolint

	u, err := LoadUnitByName(ctx, m, tx, config.Buffer.Name)
	if sdk.ErrorIs(err, sdk.ErrNotFound) {
		var srvConfig sdk.ServiceConfig
		b, _ := json.Marshal(config.Buffer.Redis)
		_ = json.Unmarshal(b, &srvConfig) // nolint
		u = &sdk.CDNUnit{
			ID:      sdk.UUID(),
			Created: time.Now(),
			Name:    config.Buffer.Name,
			Config:  srvConfig,
		}
		if err := InsertUnit(ctx, m, tx, u); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	bd.Set(*u)

	if err := tx.Commit(); err != nil {
		return nil, err
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
		defer tx.Rollback() // nolint

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
			return nil, err
		}
		storageUnit.Set(*u)

		result.Storages = append(result.Storages, storageUnit)
		if err := tx.Commit(); err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	return &result, nil
}

func (r *RunningStorageUnits) Start(ctx context.Context, gorts *sdk.GoRoutines) {
	// Start the sync processes
	for i := range r.Storages {
		s := r.Storages[i]
		for x := 0; x < cap(s.SyncItemChannel()); x++ {
			gorts.Run(ctx, fmt.Sprintf("RunningStorageUnits.Start.%s.%d", s.Name(), x),
				func(ctx context.Context) {
					for id := range s.SyncItemChannel() {
						tx, err := r.db.Begin()
						if err != nil {
							err = sdk.WrapError(err, "unable to begin tx")
							log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
							continue
						}

						if err := r.processItem(ctx, r.m, tx, s, id); err != nil {
							log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
							continue
						}

						if err := tx.Commit(); err != nil {
							err = sdk.WrapError(err, "unable to commit tx")
							log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
							_ = tx.Rollback()
							continue
						}
					}
				},
			)
		}
	}

	// 	Feed the sync processes with a ticker
	gorts.Run(ctx, "RunningStorageUnits.Start", func(ctx context.Context) {
		tickr := time.NewTicker(time.Second)
		defer tickr.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tickr.C:
				for i := range r.Storages {
					s := r.Storages[i]
					gorts.Exec(ctx, "RunningStorageUnits.Start."+s.Name(),
						func(ctx context.Context) {
							if err := r.Run(ctx, s, 100); err != nil {
								log.ErrorWithFields(ctx, logrus.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "RunningStorageUnits.Start> error: %v", err)
							}
						},
					)
				}
			}
		}

	})
}
