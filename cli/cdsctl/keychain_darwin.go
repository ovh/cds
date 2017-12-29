// +build darwin

package main

import "github.com/ovh/cds/sdk/keychain"

func storeSecret(configFile, url, username, token string) error {
	return keychain.StoreSecret(url, username, token)
}

func loadSecret(configFile, url string) (username, token string, err error) {
	return keychain.GetSecret(url)
}
