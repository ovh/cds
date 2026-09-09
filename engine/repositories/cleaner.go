package repositories

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
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

// cleanerStats accumulates the decisions of one cleaner run.
type cleanerStats struct {
	checked, inUse, protected, removed, failed int
	freedBytes, protectedBytes                 uint64
}

func (s *Service) vacuumFilesystemCleanerRun(ctx context.Context) error {
	start := time.Now()
	var st cleanerStats

	// Browse all roots ( full and bare )
	for _, c := range cacheRoots {
		names, err := readCacheEntries(s.rootDir(c))
		if err != nil {
			return err
		}
		for _, n := range names {
			s.cleanRepository(ctx, &st, c, n)
		}
	}

	log.Info(ctx, "vacuumFilesystemCleanerRun> done in %s on instance %q: %d git repositories checked, %d removed (%s freed), %d kept (protected, %s), %d kept (in use), %d failed",
		time.Since(start).Round(time.Millisecond), s.dao.hostname, st.checked, st.removed, humanize.IBytes(st.freedBytes), st.protected, humanize.IBytes(st.protectedBytes), st.inUse, st.failed)
	return nil
}

// cleanRepository runs the cleaner on one clone and records the outcome.
func (s *Service) cleanRepository(ctx context.Context, st *cleanerStats, c cacheRoot, repoID string) {
	st.checked++
	size, _ := s.repoSize(c.cloneKey(repoID))
	outcome, err := s.vacuumFileSystemCleanerFunc(ctx, c, repoID)
	switch {
	case err != nil:
		st.failed++
		log.Error(ctx, "vacuumFilesystemCleanerRun> %s: %v", c.cloneKey(repoID), err)
	case outcome == cleanerKeptInUse: // operation is still running
		st.inUse++
	case outcome == cleanerKeptProtected: // not yet expired
		st.protected++
		st.protectedBytes += uint64(size)
	case outcome == cleanerRemoved: // removed
		st.removed++
		st.freedBytes += uint64(size)
	}
}

// readCacheEntries lists the clone directories of a cache root, sorted; a root
// that does not exist yet holds no clone.
func readCacheEntries(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// repoLabel renders a cache entry for logs: the decoded repository URL
// followed by the raw directory name (base64 of the URL, also used in the
// Redis keys); names that are not a valid ID are returned as is.
func repoLabel(repoID string) string {
	url, err := base64.StdEncoding.DecodeString(repoID)
	if err != nil {
		return repoID
	}
	return fmt.Sprintf("%s (%s)", url, repoID)
}

// repoSizeLabel renders the last measured size of a clone ("<kind>/<id>") for logs.
func (s *Service) repoSizeLabel(key string) string {
	size, ok := s.repoSize(key)
	if !ok {
		return "size unknown"
	}
	return humanize.IBytes(uint64(size))
}

// vacuumFileSystemCleanerFunc decides the fate of one clone.
func (s *Service) vacuumFileSystemCleanerFunc(ctx context.Context, c cacheRoot, repoID string) (cleanerOutcome, error) {
	key := c.cloneKey(repoID)
	path := s.clonePath(c, repoID)
	label := "[" + c.kind + "] " + repoLabel(repoID)
	sizeLabel := s.repoSizeLabel(key)
	if !s.tryLockRepository(key) {
		log.Info(ctx, "vacuumFileSystemCleanerFunc> %s kept: an operation is running on it [%s]", label, sizeLabel)
		return cleanerKeptInUse, nil
	}
	defer s.unlockRepository(key)

	if v, expired := s.dao.isExpired(ctx, key); !expired {
		log.Info(ctx, "vacuumFileSystemCleanerFunc> %s kept: protected until %s (%s left) [%s]", label, v.Format(time.RFC3339), time.Until(v).Round(time.Second), sizeLabel)
		return cleanerKeptProtected, nil
	}

	log.Info(ctx, "vacuumFileSystemCleanerFunc> %s removed: no last access recorded for instance %q (%s) [%s]", label, s.dao.hostname, path, sizeLabel)
	if err := os.RemoveAll(path); err != nil {
		return cleanerRemoved, err
	}
	return cleanerRemoved, nil
}
