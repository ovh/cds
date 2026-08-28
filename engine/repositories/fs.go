package repositories

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// checkOrCreateRootFS makes sure Basedir and every cache root exist.
func (s *Service) checkOrCreateRootFS() error {
	dirs := []string{s.Cfg.Basedir}
	for _, c := range cacheRoots {
		dirs = append(dirs, s.rootDir(c))
	}
	for _, dir := range dirs {
		fi, err := os.Stat(dir)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.FileMode(0700)); err != nil {
				return sdk.WrapError(err, "unable to create directory %q", dir)
			}
			continue
		}
		if err != nil {
			return sdk.WithStack(err)
		}
		if !fi.IsDir() {
			return fmt.Errorf("bad configuration: %s is not a directory", dir)
		}
	}
	return nil
}

func (s *Service) checkOrCreateFS(r *sdk.OperationRepo) error {
	if err := s.checkOrCreateRootFS(); err != nil {
		return sdk.WithStack(err)
	}
	fi, err := os.Stat(r.Basedir)
	if os.IsNotExist(err) {
		return sdk.WrapError(os.MkdirAll(r.Basedir, os.FileMode(0700)), "unable to create directory %q", r.Basedir)
	}
	if err != nil {
		return sdk.WrapError(err, "unable to stat %q", r.Basedir)
	}
	if fi.IsDir() {
		return nil
	}
	return fmt.Errorf("bad repository basedir: %s is not a directory", r.Basedir)
}

// isEmptyDir reports whether dir exists and holds no entry.
func isEmptyDir(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func (s *Service) cleanFS(ctx context.Context, r *sdk.OperationRepo) error {
	log.Info(ctx, "cleaning operation basedir: %v", r.Basedir)
	return sdk.WithStack(os.RemoveAll(r.Basedir))
}

// repoSizesSnapshot is an immutable view of the disk usage under Basedir,
// aggregated per repository directory; it is replaced as a whole on each walk.
type repoSizesSnapshot struct {
	sizes      map[string]int64 // repoID -> bytes
	total      int64
	computedAt time.Time
}

// dirSize sums the size of all files under dir; entries removed while walking
// (the cleaner runs concurrently) are skipped, a vanished dir counts as 0.
func dirSize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// computeRepoSizes measures every clone of every cache root, keyed by clone key.
func (s *Service) computeRepoSizes() (*repoSizesSnapshot, error) {
	snap := &repoSizesSnapshot{sizes: make(map[string]int64)}
	for _, c := range cacheRoots {
		names, err := readCacheEntries(s.rootDir(c))
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			size, err := dirSize(s.clonePath(c, n))
			if err != nil {
				return nil, err
			}
			snap.sizes[c.cloneKey(n)] = size
			snap.total += size
		}
	}
	snap.computedAt = time.Now()
	return snap, nil
}

// repoSize returns the last known disk usage of a clone, keyed "<kind>/<id>";
// ok is false when no snapshot exists yet or the clone was absent at the last walk.
func (s *Service) repoSize(key string) (size int64, ok bool) {
	snap := s.repoSizes.Load()
	if snap == nil {
		return 0, false
	}
	size, ok = snap.sizes[key]
	return size, ok
}

func (s *Service) computeCacheSize(ctx context.Context) error {
	tick := time.NewTicker(5 * time.Minute)

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			start := time.Now()
			snap, err := s.computeRepoSizes()
			if err != nil {
				log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to compute size"))
				continue
			}
			s.repoSizes.Store(snap)
			log.Info(ctx, "computeCacheSize> measured %d git repositories in %s: %s total",
				len(snap.sizes), time.Since(start).Round(time.Millisecond), humanize.IBytes(uint64(snap.total)))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
