package keys

import (
	"testing"

	_ "github.com/ovh/cds/engine/api/test"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := Generatekeypair("foo")
	if err != nil {
		t.Fatalf("cannot generate keypair: %s\n", err)
	}

	t.Logf("Pub key:\n%s\n", pub)
	t.Logf("Priv key:\n%s\n", priv)
}
