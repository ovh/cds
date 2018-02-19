package repositories

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ovh/cds/sdk"
)

func (s *Service) checkOrCreateRootFS() error {
	fi, err := os.Stat(s.Cfg.Basedir)
	if os.IsNotExist(err) {
		return os.MkdirAll(s.Cfg.Basedir, os.FileMode(0700))
	}
	if fi.IsDir() {
		return nil
	}
	return fmt.Errorf("bad configuration: %s is not a directory", s.Cfg.Basedir)
}

func (s *Service) checkOrCreateFS(r *sdk.OperationRepo) error {
	if err := s.checkOrCreateRootFS(); err != nil {
		return sdk.WrapError(err, "checkOrCreateFS> ")
	}
	path := filepath.Join(s.Cfg.Basedir, r.ID())
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return sdk.WrapError(os.MkdirAll(path, os.FileMode(0700)), "checkOrCreateFS> unable to create directory %s", path)
	}
	if fi.IsDir() {
		return nil
	}
	r.Basedir = path
	return nil
}
