package api

import (
	"github.com/yesnault/go-toml"
	"testing"
)

func TestToto(t *testing.T) {
	c := Configuration{}
	c.VCS.GPGKeys = map[string][]GPGKey{}
	c.VCS.GPGKeys["github"] = []GPGKey{
		{
			ID:        "AAAAA",
			PublicKey: "pub",
		},
		{
			ID:        "AAAAA",
			PublicKey: "pub",
		},
	}
	c.VCS.GPGKeys["stash"] = []GPGKey{
		{
			ID:        "AAAAA",
			PublicKey: "pub",
		},
		{
			ID:        "AAAAA",
			PublicKey: "pub",
		},
	}

	bts, _ := toml.Marshal(c)
	t.Logf("%s", string(bts))
}
