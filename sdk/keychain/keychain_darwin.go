package keychain

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/osxkeychain"
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
