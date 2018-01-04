// +build dragonfly freebsd netbsd openbsd linux,386

package main

import (
	"io"

	"github.com/BurntSushi/toml"
)

func storeSecret(configFile io.Writer, c *config) error {
	enc := toml.NewEncoder(configFile)
	return enc.Encode(c)
}

func loadSecret(configFile io.Reader, c *config) error {
	_, err := toml.DecodeReader(configFile, c)
	return err
}
