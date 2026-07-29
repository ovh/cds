package repositories

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func TestBareRepoNamespace(t *testing.T) {
	s := Service{}
	s.Cfg.Basedir = "/var/lib/cds-engine/repositories"
	op := sdk.Operation{URL: "ssh://git@stash.local:7999/ovh/forgejo.git"}

	full := s.Repo(op)
	bare := s.BareRepo(op)

	assert.Equal(t, full.ID(), bare.ID(), "both caches must share the same repo ID")
	assert.Equal(t, filepath.Join(s.Cfg.Basedir, full.ID()), full.Basedir)
	assert.Equal(t, filepath.Join(s.Cfg.Basedir, "bare", bare.ID()), bare.Basedir)

	assert.NotEqual(t, full.ID(), bareLastAccessID(full.ID()), "bare retention must be scoped apart")
	assert.Equal(t, "bare/"+full.ID(), bareLastAccessID(full.ID()))
}
