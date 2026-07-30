package repositories

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func Test_vacuumFilesystemCleanerRun(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)
	s.Cfg.Basedir = t.TempDir()
	ctx := context.TODO()

	protect := func(lastAccessID string) {
		s.dao.saveLastAccess(lastAccessID, time.Now().Add(time.Minute), 60)
		t.Cleanup(func() { _ = s.Cache.Delete(s.dao.lastAccessKey(lastAccessID)) })
	}
	mkClone := func(elem ...string) string {
		path := filepath.Join(append([]string{s.Cfg.Basedir}, elem...)...)
		require.NoError(t, os.MkdirAll(path, os.FileMode(0700)))
		return path
	}

	protectedFull := sdk.UUID()
	expiredFull := sdk.UUID()
	protectedBare := sdk.UUID()
	expiredBare := sdk.UUID()
	crossProtected := sdk.UUID()

	protectedFullPath := mkClone(protectedFull)
	expiredFullPath := mkClone(expiredFull)
	protectedBarePath := mkClone(bareCacheDir, protectedBare)
	expiredBarePath := mkClone(bareCacheDir, expiredBare)
	// Protected as a full clone only: its bare copy must still expire
	crossProtectedPath := mkClone(bareCacheDir, crossProtected)

	protect(protectedFull)
	protect(bareLastAccessID(protectedBare))
	protect(crossProtected)

	require.NoError(t, s.vacuumFilesystemCleanerRun(ctx))

	assert.DirExists(t, protectedFullPath, "protected full clone must be kept")
	assert.NoDirExists(t, expiredFullPath, "expired full clone must be removed")
	assert.DirExists(t, filepath.Join(s.Cfg.Basedir, bareCacheDir), "the bare namespace itself must never be removed")
	assert.DirExists(t, protectedBarePath, "protected bare clone must be kept")
	assert.NoDirExists(t, expiredBarePath, "expired bare clone must be removed")
	assert.NoDirExists(t, crossProtectedPath, "a full clone last access must not protect the bare copy")
}

func Test_vacuumFilesystemCleanerRunWithoutBareCache(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)
	s.Cfg.Basedir = t.TempDir()

	require.NoError(t, s.vacuumFilesystemCleanerRun(context.TODO()))
}

func Test_vacuumFileSystemCleanerFunc(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)
	s.Cfg.Basedir = t.TempDir()

	otherDao := dao{store: s.Cache, hostname: "other-instance"}

	tests := []struct {
		name       string
		protect    func(repoID string)
		expectKept bool
	}{
		{
			name:       "clone without last access is removed",
			protect:    nil,
			expectKept: false,
		},
		{
			name:       "clone recently used by this instance is kept",
			protect:    func(repoID string) { s.dao.saveLastAccess(repoID, time.Now().Add(time.Minute), 60) },
			expectKept: true,
		},
		{
			name:       "last access from another instance does not protect the clone",
			protect:    func(repoID string) { otherDao.saveLastAccess(repoID, time.Now().Add(time.Minute), 60) },
			expectKept: false,
		},
		{
			name:       "clone with an operation in progress is kept even without last access",
			protect:    func(repoID string) { s.localCache.Set(repoID, true, time.Minute) },
			expectKept: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoID := sdk.UUID()
			path := filepath.Join(s.Cfg.Basedir, repoID)
			require.NoError(t, os.MkdirAll(path, os.FileMode(0700)))
			t.Cleanup(func() {
				_ = s.Cache.Delete(s.dao.lastAccessKey(repoID))
				_ = s.Cache.Delete(otherDao.lastAccessKey(repoID))
				s.localCache.Delete(repoID)
			})

			if tt.protect != nil {
				tt.protect(repoID)
			}

			_, err := s.vacuumFileSystemCleanerFunc(context.TODO(), path, repoID, repoID)
			require.NoError(t, err)
			if tt.name != "clone with an operation in progress is kept even without last access" {
				_, held := s.localCache.Get(repoID)
				require.False(t, held, "the cleaner must release its marker once done")
			}

			_, err = os.Stat(path)
			if tt.expectKept {
				require.NoError(t, err, "clone directory must still exist")
			} else {
				require.True(t, os.IsNotExist(err), "clone directory must have been removed")
			}
		})
	}
}

func Test_repoLabel(t *testing.T) {
	id := sdk.OperationRepo{URL: "git@stash.example.net:cds/cds.git"}.ID()
	require.Equal(t, "git@stash.example.net:cds/cds.git ("+id+")", repoLabel(id))
	require.Equal(t, "not-base64!", repoLabel("not-base64!"), "invalid IDs are returned unchanged")
}

func Test_repoSizeLabel(t *testing.T) {
	s := &Service{}
	require.Equal(t, "size unknown", s.repoSizeLabel("repoA"), "no snapshot yet")

	s.repoSizes.Store(&repoSizesSnapshot{sizes: map[string]int64{"repoA": 1536 * 1024}})
	require.Equal(t, "1.5 MiB", s.repoSizeLabel("repoA"))
	require.Equal(t, "size unknown", s.repoSizeLabel("repoB"), "not present at the last walk")
}
