package repositories

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) vacuumCleaner(ctx context.Context) error {
	if err := s.checkOrCreateRootFS(); err != nil {
		return sdk.WithStack(err)
	}

	tick := time.NewTicker(1 * time.Hour)

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			log.Info(ctx, "vacuumCleaner> Run")
			if err := s.vacuumFilesystemCleanerRun(ctx); err != nil {
				log.Error(ctx, "vacuumCleaner> Error cleaning the filesystem: %v", err)
			}
			if err := s.vacuumStoreCleanerRun(ctx); err != nil {
				log.Error(ctx, "vacuumCleaner> Error cleaning the store: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Service) vacuumStoreCleanerRun(ctx context.Context) error {
	ops, err := s.dao.loadAllOperations(ctx)
	if err != nil {
		return err
	}
	for _, o := range ops {
		if o.Status == sdk.OperationStatusPending || o.Status == sdk.OperationStatusProcessing || o.Date == nil {
			continue
		}
		if time.Since(*o.Date) > 24*time.Hour*time.Duration(s.Cfg.OperationRetention) {
			if err := s.dao.deleteOperation(ctx, o); err != nil {
				log.Error(ctx, "vacuumStoreCleanerRun> unable to delete operation %s: %v", o.UUID, err)
			}
		}
	}
	return nil
}

func (s *Service) vacuumFilesystemCleanerRun(ctx context.Context) error {
	fi, err := os.Open(s.Cfg.Basedir)
	if err != nil {
		return err
	}
	defer fi.Close()

	names, err := fi.Readdirnames(-1)
	if err != nil {
		return err
	}

	sort.Strings(names)

	for _, n := range names {
		if err := s.vacuumFileSystemCleanerFunc(ctx, n); err != nil {
			log.Error(context.TODO(), "vacuumFilesystemCleanerRun> %v ", err)
		}
	}

	return nil
}

func (s *Service) vacuumFileSystemCleanerFunc(ctx context.Context, repoUUID string) error {
	log.Debug(ctx, "vacuumFileSystemCleanerFunc> Checking %s", repoUUID)

	if err := s.dao.lock(ctx, repoUUID); err == errLockUnavailable {
		log.Debug(ctx, "vacuumFileSystemCleanerFunc> %s is locked. skipping", repoUUID)
		return nil
	}

	if v, b := s.dao.isExpired(ctx, repoUUID); !b {
		log.Debug(ctx, "vacuumFileSystemCleanerFunc> %s is not expired: %s. skipping", repoUUID, v.String())
		_ = s.dao.unlock(ctx, repoUUID)
		return nil
	}

	log.Debug(ctx, "vacuumFileSystemCleanerFunc> Removing %s", repoUUID)

	path := filepath.Join(s.Cfg.Basedir, repoUUID)
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	if err := s.dao.deleteLock(ctx, repoUUID); err != nil {
		log.Error(ctx, "unable to deleteLock %v: %v", repoUUID, err)
	}

	return nil
}
