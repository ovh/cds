package repositories

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoID := sdk.UUID()
			path := filepath.Join(s.Cfg.Basedir, repoID)
			require.NoError(t, os.MkdirAll(path, os.FileMode(0700)))
			t.Cleanup(func() {
				_ = s.Cache.Delete(s.dao.lastAccessKey(repoID))
				_ = s.Cache.Delete(otherDao.lastAccessKey(repoID))
			})

			if tt.protect != nil {
				tt.protect(repoID)
			}

			require.NoError(t, s.vacuumFileSystemCleanerFunc(context.TODO(), repoID))

			_, err := os.Stat(path)
			if tt.expectKept {
				require.NoError(t, err, "clone directory must still exist")
			} else {
				require.True(t, os.IsNotExist(err), "clone directory must have been removed")
			}
		})
	}
}
