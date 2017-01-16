package keys

import (
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := Generatekeypair("foo")
	if err != nil {
		t.Fatalf("cannot generate keypair: %s\n", err)
	}

	t.Logf("Pub key:\n%s\n", pub)
	t.Logf("Priv key:\n%s\n", priv)
}
