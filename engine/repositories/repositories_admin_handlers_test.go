package repositories

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func Test_getAdminRepositoriesHandler(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)
	s.Cfg.Basedir = t.TempDir()
	otherDao := dao{store: s.Cache, hostname: "other-instance"}

	protected := sdk.OperationRepo{URL: "git@example.net:cds/protected.git"}.ID()
	unprotected := sdk.OperationRepo{URL: "git@example.net:cds/unprotected.git"}.ID()
	foreign := sdk.OperationRepo{URL: "git@example.net:cds/foreign.git"}.ID()
	for _, id := range []string{protected, unprotected, foreign} {
		require.NoError(t, os.MkdirAll(filepath.Join(s.Cfg.Basedir, id), 0o700))
	}
	require.NoError(t, os.WriteFile(filepath.Join(s.Cfg.Basedir, "not-a-repo"), nil, 0o600))

	until := time.Now().Add(time.Hour).Truncate(time.Second)
	s.dao.saveLastAccess(protected, until, 3600)
	otherDao.saveLastAccess(foreign, until, 3600)
	t.Cleanup(func() {
		_ = s.Cache.Delete(s.dao.lastAccessKey(protected))
		_ = s.Cache.Delete(otherDao.lastAccessKey(foreign))
	})

	call := func() sdk.RepositoriesAdminList {
		uri := s.Router.GetRoute("GET", s.getAdminRepositoriesHandler, nil)
		require.NotEmpty(t, uri)
		rec := httptest.NewRecorder()
		s.Router.Mux.ServeHTTP(rec, newRequest(t, s, "GET", uri, nil))
		require.Equal(t, 200, rec.Code, rec.Body.String())
		var res sdk.RepositoriesAdminList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
		return res
	}

	t.Run("before the first size measure", func(t *testing.T) {
		res := call()
		require.Equal(t, "test-instance", res.Instance)
		require.Nil(t, res.ComputedAt)
		require.Zero(t, res.TotalSize)
		require.Len(t, res.Repositories, 3, "plain files under basedir are not listed")
		for _, r := range res.Repositories {
			require.Zero(t, r.Size)
		}
	})

	t.Run("with a size snapshot", func(t *testing.T) {
		s.repoSizes.Store(&repoSizesSnapshot{
			sizes:      map[string]int64{protected: 10, unprotected: 300, foreign: 20},
			total:      330,
			computedAt: time.Now(),
		})
		res := call()
		require.NotNil(t, res.ComputedAt)
		require.Equal(t, int64(330), res.TotalSize)
		require.Len(t, res.Repositories, 3)

		byURL := map[string]sdk.RepositoriesAdminEntry{}
		var order []string
		for _, r := range res.Repositories {
			byURL[r.URL] = r
			order = append(order, r.URL)
		}
		require.Equal(t, []string{"git@example.net:cds/unprotected.git", "git@example.net:cds/foreign.git", "git@example.net:cds/protected.git"}, order, "sorted by decreasing size")

		p := byURL["git@example.net:cds/protected.git"]
		require.Equal(t, protected, p.ID)
		require.Equal(t, int64(10), p.Size)
		require.False(t, p.Expired)
		require.NotNil(t, p.ProtectedUntil)
		require.WithinDuration(t, until, *p.ProtectedUntil, time.Second)

		u := byURL["git@example.net:cds/unprotected.git"]
		require.True(t, u.Expired)
		require.Nil(t, u.ProtectedUntil)

		f := byURL["git@example.net:cds/foreign.git"]
		require.True(t, f.Expired, "a last access from another instance does not protect the repository here")
		require.Nil(t, f.ProtectedUntil)
	})
}
