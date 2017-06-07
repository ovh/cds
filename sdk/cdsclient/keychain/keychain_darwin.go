package keychain

import (
	"fmt"

	"github.com/docker/docker-credential-helpers/credentials"
)

// Errors
var (
	ErrLibSecretUnavailable = fmt.Errorf("Shared library libsecret not found. Please install libsecret-1 from your package manager")
	ErrLdd                  = fmt.Errorf("Unable to check shared object dependencies")
	ErrExecNotFound         = fmt.Errorf("Unable to get current binary file")
)

//StoreSecret store a credential through libsecret
func StoreSecret(url, username, secret string) error {
	var nativeStore = osxkeychain.Osxkeychain{}

	c := &credentials.Credentials{
		ServerURL: url,
		Username:  username,
		Secret:    secret,
	}

	return nativeStore.Add(c)
}

//GetSecret rerieves a credential through libsecret
func GetSecret(url string) (username string, secret string, err error) {
	var nativeStore = osxkeychain.Osxkeychain{}
	username, secret, err = nativeStore.Get(url)
	return
}
