package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type Local struct {
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.LocalStorageConfiguration
	size   int64
}

func init() {
	storage.RegisterDriver("local", new(Local))
}

func (s *Local) Init(ctx context.Context, cfg interface{}) error {
	config, is := cfg.(*storage.LocalStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	s.ConvergentEncryption = encryption.New(config.Encryption)

	if err := os.MkdirAll(s.config.Path, os.FileMode(0700)); err != nil {
		return sdk.WithStack(err)
	}

	s.GoRoutines.Run(ctx, "cdn-local-compute-size", func(ctx context.Context) {
		s.computeSize(ctx)
	})

	return nil
}

func (s *Local) filename(i sdk.CDNItemUnit) (string, error) {
	loc := i.Locator
	if err := os.MkdirAll(filepath.Join(s.config.Path, loc[:3]), os.FileMode(0700)); err != nil {
		return "", nil
	}
	return filepath.Join(s.config.Path, loc[:3], loc), nil
}

func (s *Local) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	iu, err := s.ExistsInDatabase(ctx, m, db, i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	// Lookup on the filesystem according to the locator
	path, err := s.filename(*iu)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)

	return !os.IsNotExist(err), nil
}

func (s *Local) NewWriter(_ context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error) {
	// Open the file from the filesystem according to the locator
	path, err := s.filename(i)
	if err != nil {
		return nil, err
	}
	log.Debug("[%T] writing to %s", s, path)
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(0640))
}

func (s *Local) NewReader(_ context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	// Open the file from the filesystem according to the locator
	path, err := s.filename(i)
	if err != nil {
		return nil, err
	}
	log.Debug("[%T] reading from %s", s, path)
	return os.Open(path)
}

func (s *Local) Status(_ context.Context) []sdk.MonitoringStatusLine {
	var lines []sdk.MonitoringStatusLine
	if finfo, err := os.Stat(s.config.Path); os.IsNotExist(err) {
		lines = append(lines, sdk.MonitoringStatusLine{
			Component: "backend/" + s.Name(),
			Value:     fmt.Sprintf("directory: %v does not exist", s.config.Path),
			Status:    sdk.MonitoringStatusAlert,
		})
	} else if !finfo.IsDir() {
		lines = append(lines, sdk.MonitoringStatusLine{
			Component: "backend/" + s.Name(),
			Value:     fmt.Sprintf("%v is not a directory", s.config.Path),
			Status:    sdk.MonitoringStatusAlert,
		})
	}

	status := sdk.MonitoringStatusOK
	for _, v := range lines {
		if v.Status != sdk.MonitoringStatusOK {
			status = v.Status
		}
	}
	lines = append(lines, sdk.MonitoringStatusLine{
		Component: "backend/" + s.Name(),
		Value:     fmt.Sprintf("size: %d bytes", s.size),
		Status:    status,
	})
	return lines
}

func (s *Local) computeSize(ctx context.Context) {
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "cdn:backend:local:computeSize: %v", ctx.Err())
			}
			return
		case <-tick.C:
			var err error
			s.size, err = s.dirSize(s.config.Path)
			if err != nil {
				log.Error(ctx, "cdn:backend:local:computeSize:dirSize: %v", ctx.Err())
				continue
			}
		}
	}
}

func (s *Local) dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return sdk.WithStack(err)
	})
	return size, err
}

func (s *Local) Remove(_ context.Context, i sdk.CDNItemUnit) error {
	path, err := s.filename(i)
	if err != nil {
		return err
	}
	log.Debug("[%T] remove %s", s, path)
	return sdk.WithStack(os.Remove(path))
}
