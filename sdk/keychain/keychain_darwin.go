package keychain

import (
	"fmt"

	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/osxkeychain"
)

//StoreSecret store a credential through libsecret
func StoreSecret(url, username, secret string) error {
	var nativeStore = osxkeychain.Osxkeychain{}

	c := &credentials.Credentials{
		ServerURL: getServerURL(url, username),
		Username:  username,
		Secret:    secret,
	}

	return nativeStore.Add(c)
}

//GetSecret rerieves a credential through libsecret
func GetSecret(url, username string) (string, error) {
	var nativeStore = osxkeychain.Osxkeychain{}
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
