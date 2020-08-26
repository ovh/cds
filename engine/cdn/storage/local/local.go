package local

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type Local struct {
	storage.AbstractUnit
	encryption.ConvergentEncryption
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
	s.ConvergentEncryption = encryption.New(config.Encryption)
	return os.MkdirAll(s.config.Path, os.FileMode(0755))
}

func (s *Local) filename(i storage.ItemUnit) (string, error) {
	loc := i.Locator
	if err := os.MkdirAll(filepath.Join(s.config.Path, loc[:3]), os.FileMode(0700)); err != nil {
		return "", nil
	}
	return filepath.Join(s.config.Path, loc[:3], loc), nil
}

func (s *Local) ItemExists(i index.Item) (bool, error) {
	iu, err := s.ExistsInDatabase(i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	// Lookup on the filesystem according to the locator
	path, err := s.filename(*iu)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	return !os.IsNotExist(err), nil
}

func (s *Local) NewWriter(i storage.ItemUnit) (io.WriteCloser, error) {
	// Open the file from the filesystem according to the locator
	path, err := s.filename(i)
	if err != nil {
		return nil, err
	}
	log.Debug("[%T] writing to %s", s, path)
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(0644))
}

func (s *Local) NewReader(i storage.ItemUnit) (io.ReadCloser, error) {
	// Open the file from the filesystem according to the locator
	path, err := s.filename(i)
	if err != nil {
		return nil, err
	}
	log.Debug("[%T] reading to %s", s, path)
	return os.Open(path)
}
