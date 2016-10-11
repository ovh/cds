package stash

import (
	"crypto/rand"
	"crypto/rsa"
	"log"
	"testing"

	"golang.org/x/crypto/ssh"
)

func GenerateNewPublicKey() string {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal("Failed to generate new RSA key: ", err)
	}
	pk, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		log.Fatal("Failed to generate public key: ", err)
	}
	return string(ssh.MarshalAuthorizedKey(pk))
}

func TestKeyCreateUpdate(t *testing.T) {
	k := GenerateNewPublicKey()
	key, err := client.Keys.CreateUpdate(k)
	if err != nil {
		t.Errorf("Unexpected error on `client.Keys.Create()`, got %v", err)
	}
	if key.Text == "" {
		t.Error("Expected key text, got nothing")
	}
}

func TestKeyCreateUpdateInvalidKey(t *testing.T) {
	key, err := client.Keys.CreateUpdate("key")
	if err == nil {
		t.Error("Expected error on `client.Keys.CreateUpdate()`")
	}
	if key != nil {
		t.Errorf("Did not expect key, got %v", key)
	}
}

func TestKeyFindNotFound(t *testing.T) {
	key, err := client.Keys.Find("key")
	if err != ErrNotFound {
		t.Error("Expected not found error on `client.Keys.Find()`")
	}
	if key != nil {
		t.Errorf("Did not expect key, got %v", key)
	}
}

func TestKeyCreateInvalidKey(t *testing.T) {
	key, err := client.Keys.Create("key")
	if err == nil {
		t.Error("Expected error on `client.Keys.Create()`")
	}
	if key != nil {
		t.Errorf("Did not expect key, got %v", key)
	}
}

func TestKeyCreate(t *testing.T) {
	k := GenerateNewPublicKey()
	key, err := client.Keys.Create(k)
	if err != nil {
		t.Errorf("Unexpected error on `client.Keys.Create()`, got %v", err)
	}
	if key.Text == "" {
		t.Errorf("Expect key text, got nothin")
	}
}

func TestKeyFind(t *testing.T) {
	k := GenerateNewPublicKey()
	_, err := client.Keys.Create(k)
	if err != nil {
		t.Errorf("Unexpected error on `client.Keys.Create()`, got %v", err)
	}
	key, err := client.Keys.Find(k)
	if err != nil {
		t.Errorf("Unexpected error on `client.Keys.Find()`, got %v", err)
	}
	if key.Text == "" {
		t.Error("Expected key text, got nothing")
	}
}
