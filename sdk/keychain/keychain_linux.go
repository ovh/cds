package keychain

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/secretservice"
)

func checkLibSecretAvailable() (bool, error) {
	cliexec, err := os.Executable()
	if err != nil {
		return false, ErrExecNotFound
	}

	out, err := exec.Command("ldd", cliexec).CombinedOutput()
	if err != nil {
		return false, ErrLdd
	}

	if strings.Contains(string(out), "libsecret-1.so.0 => not found") {
		return false, nil
	}

	return true, nil
}

//StoreSecret store a credential through libsecret
func StoreSecret(url, username, secret string) error {
	ok, err := checkLibSecretAvailable()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLibSecretUnavailable
	}

	var nativeStore = secretservice.Secretservice{}

	c := &credentials.Credentials{
		ServerURL: getServerURL(url, username),
		Username:  username,
		Secret:    secret,
	}

	return nativeStore.Add(c)
}

//GetSecret rerieves a credential through libsecret
func GetSecret(url, username string) (string, error) {
	ok, err := checkLibSecretAvailable()
	if err != nil || !ok {
		return "", fmt.Errorf("libsecret is not available: %v", err)
	}
	var nativeStore = secretservice.Secretservice{}

	var err error
	var usernameFind, secret string
	usernameFind, secret, err = nativeStore.Get(getServerURL(url, username))

	// if http://url#username not found, try http://url
	if usernameFind != username {
		if usernameFind, secret, err = nativeStore.Get(url); err != nil {
			return "", err
		} else if usernameFind != username {
			return "", fmt.Errorf("username %s not found in your keychain", username)
		}
	}
	return secret, err
}
