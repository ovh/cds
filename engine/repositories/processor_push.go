package repositories

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processPush(op *sdk.Operation) error {
	gitRepo, currentBranch, err := s.processGitClone(op)
	if err != nil {
		return sdk.WrapError(err, "unable to process gitclone")
	}

	//Check is repo has diverged
	hasDiverged, err := gitRepo.HasDiverged()
	if err != nil {
		log.Error("Repositories> processPush> HasDiverged> [%s] Error: %v", op.UUID, err)
		return err
	}

	if hasDiverged {
		if err := gitRepo.ResetHard("origin/" + currentBranch); err != nil {
			log.Error("Repositories> processPush> ResetHard> [%s] Error: %v", op.UUID, err)
			return err
		}
	}

	// Switch to default branch
	if currentBranch != op.RepositoryInfo.DefaultBranch {
		if err := gitRepo.FetchRemoteBranch("origin", op.RepositoryInfo.DefaultBranch); err != nil {
			log.Error("Repositories> processPush> Checkout to default branch> [%s] error %v", op.UUID, err)
			return err
		}
	}

	// Create new branch
	if err := gitRepo.CheckoutNewBranch(op.Setup.Push.Branch); err != nil {
		log.Error("Repositories> processPush> Create new branch %s> [%s] error %v", op.Setup.Push.Branch, op.UUID, err)
		return err
	}

	// Erase cds directory
	if err := gitRepo.Remove(".cds"); err != nil {
		log.Error("Repositories> processPush> Remove old .cds directory> [%s] error %v", op.UUID, err)
		return err
	}

	// Create files
	for k, v := range op.LoadFiles.Results {
		fname := filepath.Join(".cds", k)
		fi, err := os.Create(fname)
		if err != nil {
			log.Error("Repositories> processPush> Create file %s> [%s] error %v", fname, op.UUID, err)
			return err
		}

		if _, err := io.Copy(fi, bytes.NewReader(v)); err != nil {
			log.Error("Repositories> processPush> Writing file %s> [%s] error %v", fname, op.UUID, err)
			fi.Close()
			return err
		}
		if err := fi.Close(); err != nil {
			log.Error("Repositories> processPush> Closing file %s> [%s] error %v", fname, op.UUID, err)
			return err
		}
		if err := gitRepo.Add(fname); err != nil {
			log.Error("Repositories> processPush> Git add file %s> [%s] error %v", fname, op.UUID, err)
			return err
		}
	}

	// Commit files
	if err := gitRepo.Commit(op.Setup.Push.Message); err != nil {
		log.Error("Repositories> processPush> Commit> [%s] error %v", op.UUID, err)
		return err
	}

	// Push branch
	if err := gitRepo.Push("origin", op.Setup.Push.Branch); err != nil {
		log.Error("Repositories> processPush> push %s> [%s] error %v", op.Setup.Push.Branch, op.UUID, err)
		return err
	}

	log.Info("Repositories> processPush> files pushed")
	return nil
}
