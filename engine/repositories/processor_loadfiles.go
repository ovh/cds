package repositories

import (
	"fmt"
	"io/ioutil"

	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processLoadFiles(op *sdk.Operation) error {
	r := s.Repo(*op)

	gitRepo, err := repo.New(r.Basedir)
	if err != nil {
		log.Error("Repositories> processLoadFiles> repo.New > [%s] Error: %v", op.UUID, err)
		return err
	}

	files, err := gitRepo.Glob(op.LoadFiles.Pattern)
	if err != nil {
		log.Error("Repositories> processLoadFiles> Glob> [%s] Error: %v", op.UUID, err)
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("No file found in %s", op.LoadFiles.Pattern)
	}

	op.LoadFiles.Results = make(map[string][]byte, len(files))

	for _, f := range files {
		fi, err := gitRepo.Open(f)
		if err != nil {
			log.Debug("Repositories> processLoadFiles> Open > [%s] Error: %v", op.UUID, err)
			return err
		}
		defer fi.Close()

		btes, err := ioutil.ReadAll(fi)
		if err != nil {
			log.Debug("Repositories> processLoadFiles> ReadAll> [%s] Error: %v", op.UUID, err)
			return err
		}

		op.LoadFiles.Results[f] = btes
	}

	return nil
}
