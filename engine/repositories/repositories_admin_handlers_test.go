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
	mkCloneDir(t, s, cacheRootFull, protected)
	mkCloneDir(t, s, cacheRootFull, unprotected)
	mkCloneDir(t, s, cacheRootFull, foreign)
	mkCloneDir(t, s, cacheRootBare, protected) // bare copy of the same repository, with its own retention
	require.NoError(t, os.WriteFile(filepath.Join(s.rootDir(cacheRootFull), "not-a-repo"), nil, 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(s.Cfg.Basedir, "not-a-root", "x"), 0o700))

	until := time.Now().Add(time.Hour).Truncate(time.Second)
	s.dao.saveLastAccess(cacheRootFull.cloneKey(protected), until, 3600)
	otherDao.saveLastAccess(cacheRootFull.cloneKey(foreign), until, 3600)
	t.Cleanup(func() {
		_ = s.Cache.Delete(s.dao.lastAccessKey(cacheRootFull.cloneKey(protected)))
		_ = s.Cache.Delete(otherDao.lastAccessKey(cacheRootFull.cloneKey(foreign)))
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
	find := func(res sdk.RepositoriesAdminList, kind, url string) sdk.RepositoriesAdminEntry {
		for _, r := range res.Repositories {
			if r.Kind == kind && r.URL == url {
				return r
			}
		}
		t.Fatalf("no %s entry for %s", kind, url)
		return sdk.RepositoriesAdminEntry{}
	}

	t.Run("before the first size measure", func(t *testing.T) {
		res := call()
		require.Equal(t, "test-instance", res.Instance)
		require.Nil(t, res.ComputedAt)
		require.Zero(t, res.TotalSize)
		require.Len(t, res.Repositories, 4, "plain files and directories outside the cache roots are not listed")
		for _, r := range res.Repositories {
			require.Zero(t, r.Size)
		}
	})

	t.Run("with a size snapshot", func(t *testing.T) {
		s.repoSizes.Store(&repoSizesSnapshot{
			sizes: map[string]int64{
				cacheRootFull.cloneKey(protected):   10,
				cacheRootFull.cloneKey(unprotected): 300,
				cacheRootFull.cloneKey(foreign):     20,
				cacheRootBare.cloneKey(protected):   5,
			},
			total:      335,
			computedAt: time.Now(),
		})
		res := call()
		require.NotNil(t, res.ComputedAt)
		require.Equal(t, int64(335), res.TotalSize)
		require.Len(t, res.Repositories, 4)

		var order []string
		for _, r := range res.Repositories {
			order = append(order, r.Kind+" "+r.URL)
		}
		require.Equal(t, []string{
			"full git@example.net:cds/unprotected.git",
			"full git@example.net:cds/foreign.git",
			"full git@example.net:cds/protected.git",
			"bare git@example.net:cds/protected.git",
		}, order, "sorted by decreasing size")

		p := find(res, "full", "git@example.net:cds/protected.git")
		require.Equal(t, protected, p.ID)
		require.Equal(t, int64(10), p.Size)
		require.False(t, p.Expired)
		require.NotNil(t, p.ProtectedUntil)
		require.WithinDuration(t, until, *p.ProtectedUntil, time.Second)

		pb := find(res, "bare", "git@example.net:cds/protected.git")
		require.Equal(t, protected, pb.ID, "both copies share the repository id")
		require.Equal(t, int64(5), pb.Size)
		require.True(t, pb.Expired, "the full copy retention does not protect the bare copy")
		require.Nil(t, pb.ProtectedUntil)

		u := find(res, "full", "git@example.net:cds/unprotected.git")
		require.True(t, u.Expired)
		require.Nil(t, u.ProtectedUntil)

		f := find(res, "full", "git@example.net:cds/foreign.git")
		require.True(t, f.Expired, "a last access from another instance does not protect the repository here")
		require.Nil(t, f.ProtectedUntil)
	})
}
