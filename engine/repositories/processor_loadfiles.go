package repositories

import (
	"io/ioutil"

	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processLoadFiles(op *sdk.Operation) error {
	r := s.Repo(*op)

	gitRepo, err := repo.New(r.Basedir)
	if err != nil {
		log.Debug("Repositories> processLoadFiles> Error: %v", err)
		return err
	}

	files, err := gitRepo.Glob(op.LoadFiles.Pattern)
	if err != nil {
		log.Debug("Repositories> processLoadFiles> Error: %v", err)
		return err
	}

	op.LoadFiles.Results = make(map[string][]byte, len(files))

	for _, f := range files {
		fi, err := gitRepo.Open(f)
		if err != nil {
			log.Debug("Repositories> processLoadFiles> Error: %v", err)
			return err
		}
		defer fi.Close()

		btes, err := ioutil.ReadAll(fi)
		if err != nil {
			log.Debug("Repositories> processLoadFiles> Error: %v", err)
			return err
		}

		op.LoadFiles.Results[f] = btes
	}

	return nil
}
