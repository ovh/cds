package main

import (
	"io"

	"github.com/naoina/toml"
	"github.com/ovh/cds/sdk/keychain"
)

func storeSecret(configFile io.Writer, c *config) error {
	enc := toml.NewEncoder(configFile)
	storedConf := *c
	storedConf.Token = "xxxxxxxx"
	if err := enc.Encode(&storedConf); err != nil {
		return err
	}
	return keychain.StoreSecret(c.Host, c.User, c.Token)
}

func loadSecret(configFile io.Reader, c *config) error {
	_, err := toml.DecodeReader(configFile, c)
	if err != nil {
		return err
	}

	username, token, err := keychain.GetSecret(c.Host)
	if err != nil {
		return err
	}

	c.Token = token
	c.User = username
	return nil
}
