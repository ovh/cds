package repositories

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processPush(ctx context.Context, op *sdk.Operation) (globalErr error) {
	var missingAuth bool
	if op.RepositoryStrategy.ConnectionType == "ssh" {
		missingAuth = op.RepositoryStrategy.SSHKey == "" || op.RepositoryStrategy.SSHKeyContent == ""
	} else {
		missingAuth = op.RepositoryStrategy.User == "" || op.RepositoryStrategy.Password == ""
	}
	if missingAuth {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "authentication data required to push on repository %s", op.URL)
	}

	gitRepo, path, currentBranch, err := s.processGitClone(ctx, op)
	if err != nil {
		return sdk.WrapError(err, "unable to process gitclone")
	}

	// FIXME create Fetch and FetchTags method in go repo
	if err := gitRepo.FetchRemoteBranch(ctx, "origin", op.RepositoryInfo.DefaultBranch); err != nil {
		return sdk.WrapError(err, "cannot fetch changes from remote at %s", op.RepositoryInfo.FetchURL)
	}

	if op.Setup.Push.ToBranch == "" {
		op.Setup.Push.ToBranch = op.RepositoryInfo.DefaultBranch
	}

	// Switch to target branch
	if currentBranch != op.Setup.Push.FromBranch {
		if err := gitRepo.CheckoutNewBranch(ctx, op.Setup.Push.FromBranch); err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return sdk.WrapError(err, "cannot checkout new branch %s", op.Setup.Push.FromBranch)
			}
			if err := gitRepo.Checkout(ctx, op.Setup.Push.FromBranch); err != nil {
				return sdk.WrapError(err, "cannot checkout existing branch %s", op.Setup.Push.FromBranch)
			}
		}
	}

	// Reset hard to remote branch or default if no remote exists
	_, hasUpstream := gitRepo.LocalBranchExists(ctx, op.Setup.Push.FromBranch)
	if hasUpstream {
		if err := gitRepo.ResetHard(ctx, "origin/"+op.Setup.Push.FromBranch); err != nil {
			return sdk.WithStack(err)
		}
	} else {
		// Reset hard default branch
		if err := gitRepo.ResetHard(ctx, "origin/"+op.RepositoryInfo.DefaultBranch); err != nil {
			return sdk.WithStack(err)
		}
	}

	// In case of error, we have to clean the filesystem, to avoid pending local branches or uncommited modification
	defer func() {
		if globalErr != nil {
			r := s.Repo(*op)
			if err := s.cleanFS(ctx, r); err != nil {
				log.Error(ctx, "unable to clean FS: %v", err)
			}
		}
	}()

	// Erase existing cds directory for migration, if update make sure that the cds directory exists
	if !op.Setup.Push.Update {
		if _, err := os.Stat(path + "/.cds"); err == nil {
			if err := os.RemoveAll(path + "/.cds"); err != nil {
				return sdk.WrapError(err, "error removing old .cds directory")
			}
		}
		if err := os.Mkdir(filepath.Join(path, ".cds"), os.ModePerm); err != nil {
			return sdk.WrapError(err, "error creating .cds directory")
		}
	} else {
		if _, err := os.Stat(path + "/.cds"); err != nil {
			if err := os.Mkdir(filepath.Join(path, ".cds"), os.ModePerm); err != nil {
				return sdk.WrapError(err, "error creating .cds directory")
			}
		}
	}

	for k, v := range op.LoadFiles.Results {
		fname := filepath.Join(path, ".cds", k)
		log.Debug("Creating %s", fname)
		_ = os.Remove(fname)
		fi, err := os.Create(fname)
		if err != nil {
			return sdk.WrapError(err, "cannot create file %s", fname)
		}

		if _, err := io.Copy(fi, bytes.NewReader(v)); err != nil {
			fi.Close() // nolint
			return sdk.WrapError(err, "writing file %s", fname)
		}
		if err := fi.Close(); err != nil {
			return sdk.WrapError(err, "closing file %s", fname)
		}
	}
	if err := gitRepo.Add(ctx, path+"/.cds/*"); err != nil {
		return sdk.WrapError(err, "git add file %s", path+"/.cds/*")
	}

	// In case that there are no changes (ex: push changes on an existing branch that was not merged)
	if !gitRepo.ExistsDiff(ctx) {
		return sdk.WrapError(sdk.ErrNothingToPush, "processPush> %s : no files changes", op.UUID)
	}

	// Commit files
	opts := make([]repo.Option, 0, 1)
	if op.User.Username != "" && op.User.Email != "" {
		opts = append(opts, repo.WithUser(op.User.Email, op.User.Username))
	}
	if err := gitRepo.Commit(ctx, op.Setup.Push.Message, opts...); err != nil {
		return sdk.WithStack(err)
	}

	// Push branch
	if err := gitRepo.Push(ctx, "origin", op.Setup.Push.FromBranch); err != nil {
		if strings.Contains(err.Error(), "Pushing requires write access") {
			return sdk.NewError(sdk.ErrForbidden, err)
		}
		return sdk.WrapError(err, "push %s", op.Setup.Push.FromBranch)
	}

	log.Debug("processPush> %s : files pushed", op.UUID)
	return nil
}
