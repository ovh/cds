package keychain

import (
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/docker/docker-credential-helpers/wincred"
)

//StoreSecret store a credential through wincred
func StoreSecret(url, username, secret string) error {
	var nativeStore = wincred.Wincred{}

	c := &credentials.Credentials{
		ServerURL: url,
		Username:  username,
		Secret:    secret,
	}

	return nativeStore.Add(c)
}

//GetSecret rerieves a credential through wincred
func GetSecret(url string) (username string, secret string, err error) {
	var nativeStore = secretservice.Wincred{}
	username, secret, err = nativeStore.Get(url)
	return
}
