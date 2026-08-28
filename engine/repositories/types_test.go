package repositories

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestCacheRoots(t *testing.T) {
	s := Service{}
	s.Cfg.Basedir = "/var/lib/cds-engine/repositories"
	op := sdk.Operation{URL: "ssh://git@stash.local:7999/ovh/forgejo.git"}

	full := s.Repo(op)
	bare := s.BareRepo(op)

	assert.Equal(t, full.ID(), bare.ID(), "both caches must share the same repo ID")
	assert.Equal(t, filepath.Join(s.Cfg.Basedir, "full", full.ID()), full.Basedir)
	assert.Equal(t, filepath.Join(s.Cfg.Basedir, "bare", bare.ID()), bare.Basedir)

	assert.Equal(t, "full/"+full.ID(), cacheRootFull.lastAccessID(full.ID()))
	assert.Equal(t, "bare/"+full.ID(), cacheRootBare.lastAccessID(full.ID()))
	assert.NotEqual(t, cacheRootFull.lastAccessID(full.ID()), cacheRootBare.lastAccessID(full.ID()), "retention must be scoped per cache")
}
