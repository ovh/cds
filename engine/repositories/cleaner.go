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

// cleanerStats accumulates the decisions of one cleaner run.
type cleanerStats struct {
	checked, inUse, protected, removed, failed int
	freedBytes, protectedBytes                 uint64
}

func (s *Service) vacuumFilesystemCleanerRun(ctx context.Context) error {
	start := time.Now()
	var st cleanerStats

	// Full clones live at the basedir root; the bare clones cache is a namespace, not a repository
	names, err := readCacheEntries(s.Cfg.Basedir)
	if err != nil {
		return err
	}
	for _, n := range names {
		if n == bareCacheDir {
			continue
		}
		s.cleanRepository(ctx, &st, filepath.Join(s.Cfg.Basedir, n), n, n)
	}

	// Bare clones cache, absent until the first analysis operation runs
	bareDir := filepath.Join(s.Cfg.Basedir, bareCacheDir)
	bareNames, err := readCacheEntries(bareDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, n := range bareNames {
		s.cleanRepository(ctx, &st, filepath.Join(bareDir, n), n, bareLastAccessID(n))
	}

	log.Info(ctx, "vacuumFilesystemCleanerRun> done in %s on instance %q: %d git repositories checked, %d removed (%s freed), %d kept (protected, %s), %d kept (in use), %d failed",
		time.Since(start).Round(time.Millisecond), s.dao.hostname, st.checked, st.removed, humanize.IBytes(st.freedBytes), st.protected, humanize.IBytes(st.protectedBytes), st.inUse, st.failed)
	return nil
}

// cleanRepository runs the cleaner on one directory and records the outcome.
func (s *Service) cleanRepository(ctx context.Context, st *cleanerStats, path, repoID, lastAccessID string) {
	st.checked++
	size, _ := s.repoSize(lastAccessID)
	outcome, err := s.vacuumFileSystemCleanerFunc(ctx, path, repoID, lastAccessID)
	switch {
	case err != nil:
		st.failed++
		log.Error(ctx, "vacuumFilesystemCleanerRun> %s: %v", lastAccessID, err)
	case outcome == cleanerKeptInUse:
		st.inUse++
	case outcome == cleanerKeptProtected:
		st.protected++
		st.protectedBytes += uint64(size)
	case outcome == cleanerRemoved:
		st.removed++
		st.freedBytes += uint64(size)
	}
}

func readCacheEntries(dir string) ([]string, error) {
	fi, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	names, err := fi.Readdirnames(-1)
	if err != nil {
		return nil, err
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

// repoSizeLabel renders the last measured size of a repository for logs.
func (s *Service) repoSizeLabel(repoID string) string {
	size, ok := s.repoSize(repoID)
	if !ok {
		return "size unknown"
	}
	return humanize.IBytes(uint64(size))
}

// vacuumFileSystemCleanerFunc decides the fate of one clone directory: repoID
// is the repository id shared by its full and bare copies (the processor's
// in-progress marker), lastAccessID the retention key of this copy.
func (s *Service) vacuumFileSystemCleanerFunc(ctx context.Context, path, repoID, lastAccessID string) (cleanerOutcome, error) {
	label := repoLabel(repoID)
	if lastAccessID != repoID {
		label = "[" + filepath.Dir(lastAccessID) + "] " + label
	}
	sizeLabel := s.repoSizeLabel(lastAccessID)
	// Same marker as the processor: whoever holds it owns the directory, the other steps back.
	if err := s.localCache.Add(repoID, true, 10*time.Minute); err != nil {
		log.Info(ctx, "vacuumFileSystemCleanerFunc> %s kept: an operation is running on it [%s]", label, sizeLabel)
		return cleanerKeptInUse, nil
	}
	defer s.localCache.Delete(repoID)

	if v, expired := s.dao.isExpired(ctx, lastAccessID); !expired {
		log.Info(ctx, "vacuumFileSystemCleanerFunc> %s kept: protected until %s (%s left) [%s]", label, v.Format(time.RFC3339), time.Until(v).Round(time.Second), sizeLabel)
		return cleanerKeptProtected, nil
	}

	log.Info(ctx, "vacuumFileSystemCleanerFunc> %s removed: no last access recorded for instance %q (%s) [%s]", label, s.dao.hostname, path, sizeLabel)
	if err := os.RemoveAll(path); err != nil {
		return cleanerRemoved, err
	}
	return cleanerRemoved, nil
}
