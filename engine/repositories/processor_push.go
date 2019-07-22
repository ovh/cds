package repositories

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processPush(op *sdk.Operation) error {
	gitRepo, path, currentBranch, err := s.processGitClone(op)
	if err != nil {
		return sdk.WrapError(err, "unable to process gitclone")
	}

	if op.Setup.Push.ToBranch == "" {
		op.Setup.Push.ToBranch = op.RepositoryInfo.DefaultBranch
	}

	// Switch to default branch
	if currentBranch != op.RepositoryInfo.DefaultBranch {
		if err := gitRepo.FetchRemoteBranch("origin", op.RepositoryInfo.DefaultBranch); err != nil {
			log.Error("Repositories> processPush> Checkout to default branch> [%s] error %v", op.UUID, err)
			return sdk.WrapError(err, "Checkout to default branch> [%s] error %v", op.UUID, err)
		}
	}

	// Reset hard default branch
	if err := gitRepo.ResetHard("origin/" + op.RepositoryInfo.DefaultBranch); err != nil {
		log.Error("Repositories> processPush> ResetHard> [%s] Error: %v", op.UUID, err)
		return err
	}

	// Create new branch
	if err := gitRepo.CheckoutNewBranch(op.Setup.Push.FromBranch); err != nil {
		log.Error("Repositories> processPush> Create new branch %s> [%s] error %v", op.Setup.Push.FromBranch, op.UUID, err)
		return sdk.WrapError(err, "Create new branch %s> [%s] error %v", op.Setup.Push.FromBranch, op.UUID, err)
	}

	// Erase cds directory
	_, errStat := os.Stat(path + "/.cds")
	if errStat == nil {
		if err := os.RemoveAll(path + "/.cds"); err != nil {
			log.Error("Repositories> processPush> Remove old .cds directory> [%s] error %v", op.UUID, err)
			return sdk.WrapError(err, "Remove old .cds directory> [%s] error %v", op.UUID, err)
		}
	}

	// Create files
	if err := os.Mkdir(filepath.Join(path, ".cds"), os.ModePerm); err != nil {
		log.Error("Repositories> processPush> Creating cds directory> [%s] error %v", op.UUID, err)
		return sdk.WrapError(err, "Creating cds directory> [%s] error %v", op.UUID, err)
	}
	for k, v := range op.LoadFiles.Results {
		fname := filepath.Join(path, ".cds", k)
		log.Debug("Creating %s", fname)
		fi, err := os.Create(fname)
		if err != nil {
			log.Error("Repositories> processPush> Create file %s> [%s] error %v", fname, op.UUID, err)
			return sdk.WrapError(err, "Create file %s> [%s] error %v", fname, op.UUID, err)
		}

		if _, err := io.Copy(fi, bytes.NewReader(v)); err != nil {
			log.Error("Repositories> processPush> Writing file %s> [%s] error %v", fname, op.UUID, err)
			fi.Close() // nolint
			return sdk.WrapError(err, "Writing file %s> [%s] error %v", fname, op.UUID, err)
		}
		if err := fi.Close(); err != nil {
			log.Error("Repositories> processPush> Closing file %s> [%s] error %v", fname, op.UUID, err)
			return sdk.WrapError(err, "Closing file %s> [%s] error %v", fname, op.UUID, err)
		}
	}
	if err := gitRepo.Add(path + "/.cds/*"); err != nil {
		log.Error("Repositories> processPush> Git add file %s> [%s] error %v", path+"/.cds/*", op.UUID, err)
		return sdk.WrapError(err, "Git add file %s> [%s] error %v", path+"/.cds/*", op.UUID, err)
	}

	// Commit files
	opts := make([]repo.Option, 0, 1)
	if op.User.Username != "" && op.User.Email != "" {
		opts = append(opts, repo.WithUser(op.User.Email, op.User.Username))
	}
	if err := gitRepo.Commit(op.Setup.Push.Message, opts...); err != nil {
		log.Error("Repositories> processPush> Commit> [%s] error %v", op.UUID, err)
		return sdk.WrapError(err, "Commit> [%s] error %v", op.UUID, err)
	}

	// Push branch
	if err := gitRepo.Push("origin", op.Setup.Push.FromBranch); err != nil {
		log.Error("Repositories> processPush> push %s> [%s] error %v", op.Setup.Push.FromBranch, op.UUID, err)
		return sdk.WrapError(err, "push %s> [%s] error %v", op.Setup.Push.FromBranch, op.UUID, err)
	}

	log.Debug("Repositories> processPush> files pushed")
	return nil
}
