// +build linux

package main

import "github.com/ovh/cds/sdk/keychain"

func storeSecret(configFile, url, username, token string) error {
	if err := keychain.StoreSecret(url, username, token); err != nil {
		return err
	}
}

func loadSecret(configFile, url string) (username, token string, err error) {
	return keychain.GetSecret(url)
}
