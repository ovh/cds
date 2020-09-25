package webdav

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-gorp/gorp"
	"github.com/studio-b12/gowebdav"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type Webdav struct {
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.WebdavStorageConfiguration
	client *gowebdav.Client
}

var (
	_              storage.StorageUnit = new(Webdav)
	metricsReaders                     = stats.Int64("cdn/storage/webdav/readers", "nb readers", stats.UnitDimensionless)
	metricsWriters                     = stats.Int64("cdn/storage/webdav/writers", "nb writers", stats.UnitDimensionless)
)

func init() {
	storage.RegisterDriver("webdav", new(Webdav))
}

func (s *Webdav) Init(ctx context.Context, _ *sdk.GoRoutines, cfg interface{}) error {
	config, is := cfg.(*storage.WebdavStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	s.ConvergentEncryption = encryption.New(config.Encryption)
	s.client = gowebdav.NewClient(config.Address, config.Username, config.Password)
	if err := s.client.Connect(); err != nil {
		return err
	}

	if err := telemetry.InitMetricsInt64(ctx, metricsReaders, metricsWriters); err != nil {
		return err
	}

	return s.client.MkdirAll(config.Path, os.FileMode(0600))
}

func (s *Webdav) filename(i sdk.CDNItemUnit) (string, error) {
	loc := i.Locator
	if err := s.client.MkdirAll(filepath.Join(s.config.Path, loc[:3]), os.FileMode(0700)); err != nil {
		return "", nil
	}
	return filepath.Join(s.config.Path, loc[:3], loc), nil
}

func (s *Webdav) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	iu, err := s.ExistsInDatabase(ctx, m, db, i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	path, err := s.filename(*iu)
	if err != nil {
		return false, err
	}
	_, err = s.client.Stat(path)
	return !os.IsNotExist(err), nil
}

func (s *Webdav) NewWriter(_ context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error) {
	f, err := s.filename(i)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	go func() {
		if err := s.client.WriteStream(f, pr, os.FileMode(0600)); err != nil {
			log.Error(context.Background(), "unable to write stream %s: %v", f, err)
			return
		}
	}()
	return pw, nil
}

func (s *Webdav) NewReader(_ context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	f, err := s.filename(i)
	if err != nil {
		return nil, err
	}
	return s.client.ReadStream(f)
}

func (s *Webdav) Status(_ context.Context) []sdk.MonitoringStatusLine {
	if err := s.client.Connect(); err != nil {
		return []sdk.MonitoringStatusLine{{Component: "backend/" + s.Name(), Value: "webdav KO" + err.Error(), Status: sdk.MonitoringStatusAlert}}
	}

	return []sdk.MonitoringStatusLine{{
		Component: "backend/" + s.Name(),
		Value:     "connect OK",
		Status:    sdk.MonitoringStatusOK,
	}}
}
