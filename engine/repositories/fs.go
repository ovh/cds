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
	fi, err := os.Stat(r.Basedir)
	if os.IsNotExist(err) {
		return sdk.WrapError(os.MkdirAll(r.Basedir, os.FileMode(0700)), "unable to create directory %q", r.Basedir)
	}
	if fi.IsDir() {
		return nil
	}
	return fmt.Errorf("bad repository basedir: %s is not a directory", r.Basedir)
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

// computeRepoSizes measures each first-level directory of basedir.
func computeRepoSizes(basedir string) (*repoSizesSnapshot, error) {
	entries, err := os.ReadDir(basedir)
	if err != nil {
		return nil, err
	}
	snap := &repoSizesSnapshot{sizes: make(map[string]int64, len(entries))}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		size, err := dirSize(filepath.Join(basedir, e.Name()))
		if err != nil {
			return nil, err
		}
		snap.sizes[e.Name()] = size
		snap.total += size
	}
	snap.computedAt = time.Now()
	return snap, nil
}

// repoSize returns the last known disk usage of a repository; ok is false when
// no snapshot exists yet or the repository was not present at the last walk.
func (s *Service) repoSize(repoID string) (size int64, ok bool) {
	snap := s.repoSizes.Load()
	if snap == nil {
		return 0, false
	}
	size, ok = snap.sizes[repoID]
	return size, ok
}

func (s *Service) computeCacheSize(ctx context.Context) error {
	tick := time.NewTicker(5 * time.Minute)

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			start := time.Now()
			snap, err := computeRepoSizes(s.Cfg.Basedir)
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
