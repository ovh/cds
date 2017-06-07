package keychain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/secretservice"
)

// Errors
var (
	ErrLibSecretUnavailable = fmt.Errorf("Shared library libsecret not found. Please install libsecret-1 from your package manager")
	ErrLdd                  = fmt.Errorf("Unable to check shared object dependencies")
	ErrExecNotFound         = fmt.Errorf("Unable to get current binary file")
)

func checkLibSecretAvailable() (bool, error) {
	cliexec, err := filepath.Abs(os.Args[0])
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
