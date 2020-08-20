package repositories

import (
	"context"
	"io/ioutil"

	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
)

func (s *Service) processLoadFiles(ctx context.Context, op *sdk.Operation) error {
	r := s.Repo(*op)

	gitRepo, err := repo.New(ctx, r.Basedir)
	if err != nil {
		return sdk.WithStack(err)
	}

	files, err := gitRepo.Glob(op.LoadFiles.Pattern)
	if err != nil {
		return sdk.WithStack(err)
	}
	if len(files) == 0 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "no file found in %s", op.LoadFiles.Pattern)
	}

	op.LoadFiles.Results = make(map[string][]byte, len(files))

	for _, f := range files {
		fi, err := gitRepo.Open(f)
		if err != nil {
			return sdk.WithStack(err)
		}
		btes, err := ioutil.ReadAll(fi)
		if err != nil {
			fi.Close()
			return sdk.WithStack(err)
		}
		op.LoadFiles.Results[f] = btes
		fi.Close()
	}

	return nil
}
