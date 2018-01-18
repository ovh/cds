// +build dragonfly freebsd netbsd openbsd linux,386 windows,386

package main

import (
	"io"

	toml "github.com/pelletier/go-toml"
)

func storeSecret(configFile io.Writer, c *config) error {
	enc := toml.NewEncoder(configFile)
	return enc.Encode(*c)
}

func loadSecret(configFile io.Reader, c *config) error {
	dec := toml.NewDecoder(configFile)
	return dec.Decode(c)
}
