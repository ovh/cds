package osxkeychain

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"testing"
)

func TestOSXKeychainHelper(t *testing.T) {
	creds := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v1",
		Username:  "foobar",
		Secret:    "foobarbaz",
	}
	creds1 := &credentials.Credentials{
		ServerURL: "https://foobar.docker.io:2376/v2",
		Username:  "foobarbaz",
		Secret:    "foobar",
	}
	helper := Osxkeychain{}
	if err := helper.Add(creds); err != nil {
		t.Fatal(err)
	}

	username, secret, err := helper.Get(creds.ServerURL)
	if err != nil {
		t.Fatal(err)
	}

	if username != "foobar" {
		t.Fatalf("expected %s, got %s\n", "foobar", username)
	}

	if secret != "foobarbaz" {
		t.Fatalf("expected %s, got %s\n", "foobarbaz", secret)
	}

	auths, err := helper.List()
	if err != nil || len(auths) == 0 {
		t.Fatal(err)
	}

	helper.Add(creds1)
	defer helper.Delete(creds1.ServerURL)
	newauths, err := helper.List()
	if len(newauths)-len(auths) != 1 {
		if err == nil {
			t.Fatalf("Error: len(newauths): %d, len(auths): %d", len(newauths), len(auths))
		}
		t.Fatalf("Error: len(newauths): %d, len(auths): %d\n Error= %v", len(newauths), len(auths), err)
	}

	if err := helper.Delete(creds.ServerURL); err != nil {
		t.Fatal(err)
	}
}

func TestMissingCredentials(t *testing.T) {
	helper := Osxkeychain{}
	_, _, err := helper.Get("https://adsfasdf.wrewerwer.com/asdfsdddd")
	if !credentials.IsErrCredentialsNotFound(err) {
		t.Fatalf("expected ErrCredentialsNotFound, got %v", err)
	}
}
