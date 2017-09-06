package keys

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/openpgp"
)

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := Generatekeypair("foo")
	if err != nil {
		t.Fatalf("cannot generate keypair: %s\n", err)
	}

	t.Logf("Pub key:\n%s\n", pub)
	t.Logf("Priv key:\n%s\n", priv)
}

func TestGenerateGPGKeyPair(t *testing.T) {
	kid, pub, priv, err := GeneratePGPKeyPair("mykey")
	if err != nil {
		t.Fatalf("cannot generate keypair: %s\n", err)
	}
	t.Logf("Pub key:\n%s\n", pub)
	t.Logf("Priv key:\n%s\n", priv)
	t.Logf("ID key:\n%s\n", kid)

	stringToEncode := "I am a secret"
	entityList, err := openpgp.ReadArmoredKeyRing(strings.NewReader(pub))

	// encrypt string

	buf := new(bytes.Buffer)
	w, err := openpgp.Encrypt(buf, entityList, nil, nil, nil)
	if err != nil {
		t.Fatalf("cannot encrypt string: %s", err)
	}
	if _, err := w.Write([]byte(stringToEncode)); err != nil {
		t.Fatalf("cannot encrypt string: %s", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("cannot close %s", err)
	}
	bb, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Fatalf("cannot close %s", err)
	}
	encStr := base64.StdEncoding.EncodeToString(bb)
	t.Logf("Encrypted string:\n%s\n", encStr)

	//Decrypt string

	entityPrivate, errE := openpgp.ReadArmoredKeyRing(strings.NewReader(priv))
	if errE != nil {
		t.Fatalf("Cannot read private key: %s\n", errE)
	}
	dec, err := base64.StdEncoding.DecodeString(encStr)
	if err != nil {
		t.Fatalf("Decode 64 %s\n", err)
	}

	md, err := openpgp.ReadMessage(bytes.NewBuffer([]byte(dec)), entityPrivate, nil, nil)
	if err != nil {
		t.Fatalf("Cannot read message %s\n", err)
	}
	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		t.Fatalf("Cannot readall %s\n", err)
	}
	decStr := string(bytes)
	t.Logf("Decrypted string:\n%s\n", decStr)

	assert.Equal(t, stringToEncode, decStr)

}
