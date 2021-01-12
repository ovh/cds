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

type AbstractLocal struct {
	storage.AbstractUnit
	size     int64
	path     string
	isBuffer bool
}

type Local struct {
	AbstractLocal
	config storage.LocalStorageConfiguration
	encryption.ConvergentEncryption
}

func init() {
	storage.RegisterDriver("local", new(Local))
}

func (s *Local) Init(ctx context.Context, cfg interface{}) error {
	config, is := cfg.(*storage.LocalStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.path = config.Path
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

func (s *AbstractLocal) filename(i sdk.CDNItemUnit) (string, error) {
	if !s.isBuffer {
		loc := i.Locator
		if err := os.MkdirAll(filepath.Join(s.path, loc[:3]), os.FileMode(0700)); err != nil {
			return "", sdk.WithStack(err)
		}
		return filepath.Join(s.path, loc[:3], loc), nil
	}
	if err := os.MkdirAll(filepath.Join(s.path, string(i.Type)), os.FileMode(0700)); err != nil {
		return "", sdk.WithStack(err)
	}
	return filepath.Join(s.path, string(i.Type), i.Item.APIRefHash), nil
}

func (s *AbstractLocal) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
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

func (s *AbstractLocal) NewWriter(_ context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error) {
	// Open the file from the filesystem according to the locator
	path, err := s.filename(i)
	if err != nil {
		return nil, err
	}
	log.Debug("[%T] writing to %s", s, path)
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(0640))
}

func (s *AbstractLocal) NewReader(_ context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	// Open the file from the filesystem according to the locator
	path, err := s.filename(i)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	log.Debug("[%T] reading from %s", s, path)
	f, err := os.Open(path)
	return f, sdk.WithStack(err)
}

func (s *AbstractLocal) Status(_ context.Context) []sdk.MonitoringStatusLine {
	var lines []sdk.MonitoringStatusLine
	if finfo, err := os.Stat(s.path); os.IsNotExist(err) {
		lines = append(lines, sdk.MonitoringStatusLine{
			Component: "backend/" + s.Name(),
			Value:     fmt.Sprintf("directory: %v does not exist", s.path),
			Status:    sdk.MonitoringStatusAlert,
		})
	} else if !finfo.IsDir() {
		lines = append(lines, sdk.MonitoringStatusLine{
			Component: "backend/" + s.Name(),
			Value:     fmt.Sprintf("%v is not a directory", s.path),
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

func (s *AbstractLocal) computeSize(ctx context.Context) {
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
			s.size, err = s.dirSize(s.path)
			if err != nil {
				log.Error(ctx, "cdn:backend:local:computeSize:dirSize: %v", ctx.Err())
				continue
			}
		}
	}
}

func (s *AbstractLocal) dirSize(path string) (int64, error) {
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

func (s *AbstractLocal) Remove(_ context.Context, i sdk.CDNItemUnit) error {
	path, err := s.filename(i)
	if err != nil {
		return err
	}
	log.Debug("[%T] remove %s", s, path)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return sdk.ErrNotFound
		}
		return sdk.WithStack(err)
	}
	return nil
}
