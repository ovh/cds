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

	assert.Equal(t, "full/"+full.ID(), cacheRootFull.cloneKey(full.ID()))
	assert.Equal(t, "bare/"+full.ID(), cacheRootBare.cloneKey(full.ID()))
	assert.NotEqual(t, cacheRootFull.cloneKey(full.ID()), cacheRootBare.cloneKey(full.ID()), "retention must be scoped per cache")
}

func TestCloneKey(t *testing.T) {
	s := Service{}
	s.Cfg.Basedir = "/var/lib/cds-engine/repositories"
	s.Cfg.BareAnalysisCache = true
	op := sdk.Operation{URL: "ssh://git@stash.local:7999/ovh/forgejo.git"}
	id := s.Repo(op).ID()

	op.Setup.Checkout.Branch = "master"
	op.LoadFiles.Pattern = ".cds/**"
	assert.Equal(t, "full/"+id, s.cloneKey(op), "loading files needs the full clone")

	op.LoadFiles.Pattern = ""
	op.Setup.Checkout.CheckSignature = true
	assert.Equal(t, "bare/"+id, s.cloneKey(op), "an analysis-only operation uses the bare cache")
}
