package main

import (
	"io"

	toml "github.com/pelletier/go-toml"

	"github.com/ovh/cds/sdk/keychain"
)

func storeSecret(configFile io.Writer, c *config) error {
	storedConf := *c
	storedConf.Token = "xxxxxxxx"
	enc := toml.NewEncoder(configFile)
	if err := enc.Encode(storedConf); err != nil {
		return err
	}
	return keychain.StoreSecret(c.Host, c.User, c.Token)
}

func loadSecret(configFile io.Reader, c *config) error {
	dec := toml.NewDecoder(configFile)
	if err := dec.Decode(c); err != nil {
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
