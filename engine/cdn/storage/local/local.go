package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/sdk"
)

type Local struct {
	storage.AbstractUnit
	config storage.LocalStorageConfiguration
}

var _ storage.StorageUnit = new(Local)

func init() {
	storage.RegisterDriver("local", new(Local))
}

func (s *Local) Init(cfg interface{}) error {
	config, is := cfg.(*storage.LocalStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	return os.MkdirAll(s.config.Path, os.FileMode(0755))
}

func (s *Local) ItemExists(i index.Item) (bool, error) {
	// Lookup on the filesystem according to the locator
	_, err := os.Stat(filepath.Join(s.config.Path, i.ID))
	return os.IsExist(err), nil
}

func (s *Local) NewWriter(i index.Item) (io.WriteCloser, error) {
	// Open the file from the filesystem according to the locator
	// TODO calculate the locator, encrypt, etc...
	return os.OpenFile(filepath.Join(s.config.Path, i.ID), os.O_CREATE|os.O_RDWR, os.FileMode(0644))
}
func (s *Local) NewReader(i index.Item) (io.ReadCloser, error) {
	// Open the file from the filesystem according to the locator
	return os.Open(filepath.Join(s.config.Path, i.ID))
}
