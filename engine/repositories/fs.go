package repositories

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) checkOrCreateRootFS() error {
	fi, err := os.Stat(s.Cfg.Basedir)
	if os.IsNotExist(err) {
		return sdk.WrapError(os.MkdirAll(s.Cfg.Basedir, os.FileMode(0700)), "unable to create directory %q", s.Cfg.Basedir)
	}
	if fi.IsDir() {
		return nil
	}
	return fmt.Errorf("bad configuration: %s is not a directory", s.Cfg.Basedir)
}

func (s *Service) checkOrCreateFS(r *sdk.OperationRepo) error {
	if err := s.checkOrCreateRootFS(); err != nil {
		return sdk.WithStack(err)
	}
	path := filepath.Join(s.Cfg.Basedir, r.ID())
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return sdk.WrapError(os.MkdirAll(path, os.FileMode(0700)), "unable to create directory %q", path)
	}
	if fi.IsDir() {
		return nil
	}
	r.Basedir = path
	return nil
}

func (s *Service) cleanFS(ctx context.Context, r *sdk.OperationRepo) error {
	log.Info(ctx, "cleaning operation basedir: %v", r.Basedir)
	return sdk.WithStack(os.RemoveAll(r.Basedir))
}

func (s *Service) computeCacheSize(ctx context.Context) error {
	tick := time.NewTicker(5 * time.Minute)

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			var size int64
			err := filepath.Walk(s.Cfg.Basedir, func(_ string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					size += info.Size()
				}
				return err
			})
			if err != nil {
				log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to compute size"))
				continue
			}
			s.cacheSize = size
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
