package secret

import (
	"bytes"
	"database/sql"
	"testing"

	_ "github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"
)

func TestInvalidKey(t *testing.T) {
	key = []byte("78eKVxLm6gwoH9LAQ15ZD5AOABo1Xb239fj209uf23hwefw34")
	data := []byte("Hello world !")

	_, err := Encrypt(data)
	if err == nil {
		t.Fatalf("Encrypt should have failed: %s", err)
	}
}

func TestEncrypt(t *testing.T) {
	key = []byte("78eKVxCGLm6gwoH9LAQ15ZD5AOABo1Xf")
	data := []byte("Hello world !")

	ct, err := Encrypt(data)
	if err != nil {
		t.Fatalf("Encrypt failed: %s", err)
	}

	clear, err := Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt failed: %s", err)
	}

	if bytes.Compare(clear, data) != 0 {
		t.Fatalf("Fail: Expected '%s', got '%s'", data, clear)
	}
}

func TestEncryptEmpty(t *testing.T) {
	key = []byte("78eKVxCGLm6gwoH9LAQ15ZD5AOABo1Xf")
	data := []byte("")

	ct, err := Encrypt(data)
	if err != nil {
		t.Fatalf("Encrypt failed: %s", err)
	}

	clear, err := Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt failed: %s", err)
	}

	if bytes.Compare(clear, data) != 0 {
		t.Fatalf("Fail: Expected '%s', got '%s'", data, clear)
	}
}

func TestClear(t *testing.T) {
	key = []byte("78eKVxCGLm6gwoH9LAQ15ZD5AOABo1Xb")
	data := []byte("Hello world !")

	clear, err := Decrypt(data)
	if err != nil {
		t.Fatalf("Decrypt failed: %s", err)
	}

	if bytes.Compare(clear, data) != 0 {
		t.Fatalf("Fail: Expected '%s', got '%s'", data, clear)
	}
}

func TestDecryptS(t *testing.T) {

	key = []byte("78eKVxCGLm6gwoH9LAQ15ZD5AOABo1Xb")
	data := []byte("Hello world !")

	s, err := DecryptS(sdk.SecretVariable, sql.NullString{}, data, false)
	if err != nil {
		t.Fatalf("DecryptS failed: %s\n", err)
	}

	if s != sdk.PasswordPlaceholder {
		t.Fatalf("DecryptS should have return password placeholder, got '%s'\n", s)
	}

	ct, err := Encrypt(data)
	if err != nil {
		t.Fatalf("Encrypt failed: %s", err)
	}

	s, err = DecryptS(sdk.SecretVariable, sql.NullString{}, ct, true)
	if err != nil {
		t.Fatalf("DecryptS failed: %s\n", err)
	}

	if s != string(data) {
		t.Fatalf("Fail: Expected '%s', got '%s'", data, s)
	}

}
