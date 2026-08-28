package repositories

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func Test_computeRepoSizes(t *testing.T) {
	s := &Service{}
	s.Cfg.Basedir = t.TempDir()
	basedir := s.Cfg.Basedir
	write := func(rel string, n int) {
		p := filepath.Join(basedir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
		require.NoError(t, os.WriteFile(p, make([]byte, n), 0o600))
	}
	write("full/repoA/.git/HEAD", 10)
	write("full/repoA/src/main.go", 100)
	write("full/repoB/README", 7)
	write("bare/repoA/HEAD", 20) // same repository, bare copy: measured apart
	require.NoError(t, os.MkdirAll(filepath.Join(basedir, "bare", "emptyRepo"), 0o700))
	write("full/stray-file", 3)    // files directly under a root are not repositories
	write("not-a-root/repoC/x", 5) // only cache roots are measured

	snap, err := s.computeRepoSizes()
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"full/repoA": 110, "full/repoB": 7, "bare/repoA": 20, "bare/emptyRepo": 0}, snap.sizes)
	require.Equal(t, int64(137), snap.total)
	require.False(t, snap.computedAt.IsZero())
}

func Test_computeRepoSizes_missingRoot(t *testing.T) {
	s := &Service{}
	s.Cfg.Basedir = t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(s.Cfg.Basedir, "full", "repoA"), 0o700))

	snap, err := s.computeRepoSizes()
	require.NoError(t, err, "a cache root that does not exist yet holds no clone")
	require.Equal(t, map[string]int64{"full/repoA": 0}, snap.sizes)
}

func Test_dirSize_missingDir(t *testing.T) {
	size, err := dirSize(filepath.Join(t.TempDir(), "vanished"))
	require.NoError(t, err, "a directory removed by the cleaner is not an error")
	require.Zero(t, size)
}

func Test_computeRepoSizes_missingBasedir(t *testing.T) {
	s := &Service{}
	s.Cfg.Basedir = filepath.Join(t.TempDir(), "missing")
	snap, err := s.computeRepoSizes()
	require.NoError(t, err, "no cache root means no clone, not an error")
	require.Empty(t, snap.sizes)
	require.Zero(t, snap.total)
}

func Test_repoSize(t *testing.T) {
	s := &Service{}
	_, ok := s.repoSize("full/repoA")
	require.False(t, ok, "no snapshot yet")

	s.repoSizes.Store(&repoSizesSnapshot{sizes: map[string]int64{"full/repoA": 42}, total: 42})
	size, ok := s.repoSize("full/repoA")
	require.True(t, ok)
	require.Equal(t, int64(42), size)
	_, ok = s.repoSize("unknown")
	require.False(t, ok)
}

func Test_checkOrCreateFS_statError(t *testing.T) {
	s := &Service{}
	s.Cfg.Basedir = t.TempDir()
	require.NoError(t, s.checkOrCreateRootFS())
	// A regular file where a directory is expected makes Stat fail with ENOTDIR
	require.NoError(t, os.WriteFile(filepath.Join(s.rootDir(cacheRootFull), "file"), nil, 0o600))

	r := &sdk.OperationRepo{Basedir: filepath.Join(s.rootDir(cacheRootFull), "file", "repo")}
	err := s.checkOrCreateFS(r)
	require.Error(t, err, "a stat error must be reported, not dereferenced")
}
