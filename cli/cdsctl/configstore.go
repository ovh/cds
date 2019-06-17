// +build freebsd openbsd,386 linux,arm windows,386 windows,386 nokeychain,amd64 nokeychain,arm nokeychain

package main

import (
	"io"

	toml "github.com/pelletier/go-toml"
)

var keychainEnabled = false

func storeSecret(configFile io.Writer, c *config) error {
	enc := toml.NewEncoder(configFile)
	return enc.Encode(*c)
}

func loadSecret(configFile io.Reader, c *config) error {
	dec := toml.NewDecoder(configFile)
	return dec.Decode(c)
}
