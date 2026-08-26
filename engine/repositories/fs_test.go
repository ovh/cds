package repositories

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_computeRepoSizes(t *testing.T) {
	basedir := t.TempDir()
	write := func(rel string, n int) {
		p := filepath.Join(basedir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o700))
		require.NoError(t, os.WriteFile(p, make([]byte, n), 0o600))
	}
	write("repoA/.git/HEAD", 10)
	write("repoA/src/main.go", 100)
	write("repoB/README", 7)
	require.NoError(t, os.MkdirAll(filepath.Join(basedir, "emptyRepo"), 0o700))
	write("stray-file", 3) // files directly under basedir are not repositories

	snap, err := computeRepoSizes(basedir)
	require.NoError(t, err)
	require.Equal(t, map[string]int64{"repoA": 110, "repoB": 7, "emptyRepo": 0}, snap.sizes)
	require.Equal(t, int64(117), snap.total)
	require.False(t, snap.computedAt.IsZero())
}

func Test_dirSize_missingDir(t *testing.T) {
	size, err := dirSize(filepath.Join(t.TempDir(), "vanished"))
	require.NoError(t, err, "a directory removed by the cleaner is not an error")
	require.Zero(t, size)
}

func Test_computeRepoSizes_missingBasedir(t *testing.T) {
	_, err := computeRepoSizes(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func Test_repoSize(t *testing.T) {
	s := &Service{}
	_, ok := s.repoSize("repoA")
	require.False(t, ok, "no snapshot yet")

	s.repoSizes.Store(&repoSizesSnapshot{sizes: map[string]int64{"repoA": 42}, total: 42})
	size, ok := s.repoSize("repoA")
	require.True(t, ok)
	require.Equal(t, int64(42), size)
	_, ok = s.repoSize("unknown")
	require.False(t, ok)
}
