package keychain

import (
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
		ServerURL: url,
		Username:  username,
		Secret:    secret,
	}

	return nativeStore.Add(c)
}

//GetSecret rerieves a credential through libsecret
func GetSecret(url string) (username string, secret string, err error) {
	ok, err := checkLibSecretAvailable()
	if err != nil || !ok {
		return
	}
	var nativeStore = secretservice.Secretservice{}
	username, secret, err = nativeStore.Get(url)
	return
}
