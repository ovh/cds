package repositories

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) vacuumCleaner(ctx context.Context) error {
	if err := s.checkOrCreateRootFS(); err != nil {
		return sdk.WrapError(err, "checkOrCreateFS> ")
	}

	tick := time.NewTicker(1 * time.Hour)

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			log.Info("vacuumCleaner> Run")
			if err := s.vacuumFilesystemCleanerRun(); err != nil {
				log.Error("vacuumCleaner> Error cleaning the filesystem: %v", err)
			}
			if err := s.vacuumStoreCleanerRun(); err != nil {
				log.Error("vacuumCleaner> Error cleaning the store: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Service) vacuumStoreCleanerRun() error {
	ops, err := s.dao.loadAllOperations()
	if err != nil {
		return err
	}
	for _, o := range ops {
		if o.Status == sdk.OperationStatusPending || o.Status == sdk.OperationStatusProcessing || o.Date == nil {
			continue
		}
		if time.Since(*o.Date) > 24*time.Hour*time.Duration(s.Cfg.OperationRetention) {
			if err := s.dao.deleteOperation(o); err != nil {
				log.Error("vacuumStoreCleanerRun> unable to delete operation %s", o.UUID)
			}
		}
	}
	return nil
}

func (s *Service) vacuumFilesystemCleanerRun() error {
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
		if err := s.vacuumFileSystemCleanerFunc(n); err != nil {
			log.Error("vacuumFilesystemCleanerRun> ", err)
		}
	}

	return nil
}

func (s *Service) vacuumFileSystemCleanerFunc(repoUUID string) error {
	log.Debug("vacuumFileSystemCleanerFunc> Checking %s", repoUUID)

	if err := s.dao.lock(repoUUID); err == errLockUnavailable {
		log.Debug("vacuumFileSystemCleanerFunc> %s is locked. skipping", repoUUID)
		return nil
	}

	if !s.dao.isExpired(repoUUID) {
		log.Debug("vacuumFileSystemCleanerFunc> %s is not expired. skipping", repoUUID)
		s.dao.unlock(repoUUID, 24*time.Hour*time.Duration(s.Cfg.RepositoriesRentention))
		return nil
	}

	log.Debug("vacuumFileSystemCleanerFunc> Removing %s", repoUUID)

	path := filepath.Join(s.Cfg.Basedir, repoUUID)
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	s.dao.deleteLock(repoUUID)

	return nil
}
