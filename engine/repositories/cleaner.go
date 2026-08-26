package repositories

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
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
			log.Info(ctx, "vacuumCleaner> Run: removing git repositories unused for %s on instance %q",
				s.repositoriesRetention(ctx), s.dao.hostname)
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
			if err := s.dao.deleteOperation(o); err != nil {
				log.Error(ctx, "vacuumStoreCleanerRun> unable to delete operation %s: %v", o.UUID, err)
			}
		}
	}
	return nil
}

// cleanerOutcome tells what the cleaner decided for one git repository directory.
type cleanerOutcome int

const (
	cleanerKeptInUse cleanerOutcome = iota
	cleanerKeptProtected
	cleanerRemoved
)

func (s *Service) vacuumFilesystemCleanerRun(ctx context.Context) error {
	start := time.Now()
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

	var inUse, protected, removed, failed int
	var freedBytes, protectedBytes uint64
	for _, n := range names {
		size, _ := s.repoSize(n)
		outcome, err := s.vacuumFileSystemCleanerFunc(ctx, n)
		switch {
		case err != nil:
			failed++
			log.Error(ctx, "vacuumFilesystemCleanerRun> %s: %v", n, err)
		case outcome == cleanerKeptInUse:
			inUse++
		case outcome == cleanerKeptProtected:
			protected++
			protectedBytes += uint64(size)
		case outcome == cleanerRemoved:
			removed++
			freedBytes += uint64(size)
		}
	}

	log.Info(ctx, "vacuumFilesystemCleanerRun> done in %s on instance %q: %d git repositories checked, %d removed (%s freed), %d kept (protected, %s), %d kept (in use), %d failed",
		time.Since(start).Round(time.Millisecond), s.dao.hostname, len(names), removed, humanize.IBytes(freedBytes), protected, humanize.IBytes(protectedBytes), inUse, failed)
	return nil
}

// repoLabel renders a Basedir entry for logs: the decoded repository URL
// followed by the raw directory name (base64 of the URL, also used in the
// Redis keys); names that are not a valid ID are returned as is.
func repoLabel(repoID string) string {
	url, err := base64.StdEncoding.DecodeString(repoID)
	if err != nil {
		return repoID
	}
	return fmt.Sprintf("%s (%s)", url, repoID)
}

// repoSizeLabel renders the last measured size of a repository for logs.
func (s *Service) repoSizeLabel(repoID string) string {
	size, ok := s.repoSize(repoID)
	if !ok {
		return "size unknown"
	}
	return humanize.IBytes(uint64(size))
}

func (s *Service) vacuumFileSystemCleanerFunc(ctx context.Context, repoID string) (cleanerOutcome, error) {
	label := repoLabel(repoID)
	sizeLabel := s.repoSizeLabel(repoID)
	// The processor marks repositories it is working on in the local cache
	if _, busy := s.localCache.Get(repoID); busy {
		log.Info(ctx, "vacuumFileSystemCleanerFunc> %s kept: an operation is running on it [%s]", label, sizeLabel)
		return cleanerKeptInUse, nil
	}

	if v, expired := s.dao.isExpired(ctx, repoID); !expired {
		log.Info(ctx, "vacuumFileSystemCleanerFunc> %s kept: protected until %s (%s left) [%s]", label, v.Format(time.RFC3339), time.Until(v).Round(time.Second), sizeLabel)
		return cleanerKeptProtected, nil
	}

	path := filepath.Join(s.Cfg.Basedir, repoID)
	log.Info(ctx, "vacuumFileSystemCleanerFunc> %s removed: no last access recorded for instance %q (%s) [%s]", label, s.dao.hostname, path, sizeLabel)
	if err := os.RemoveAll(path); err != nil {
		return cleanerRemoved, err
	}
	return cleanerRemoved, nil
}
