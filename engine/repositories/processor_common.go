package repositories

import (
	"context"
	"strings"

	"github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processGitClone(ctx context.Context, op *sdk.Operation) (gitRepo repo.Repo, basedir string, currentBranch string, err error) {
	r := s.Repo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		return gitRepo, "", "", err
	}

	// Get the git repository
	opts := []repo.Option{repo.WithVerbose(log.InfoWithoutCtx)}

	if op.RepositoryStrategy.ConnectionType == "ssh" {
		log.Debug("processGitClone> %s > using ssh key %s", op.UUID, op.RepositoryStrategy.SSHKey)
		opts = append(opts, repo.WithSSHAuth([]byte(op.RepositoryStrategy.SSHKeyContent)))
	} else if op.RepositoryStrategy.User != "" && op.RepositoryStrategy.Password != "" {
		log.Debug("processGitClone> %s > using user %s", op.UUID, op.RepositoryStrategy.User)
		opts = append(opts, repo.WithHTTPAuth(op.RepositoryStrategy.User, op.RepositoryStrategy.Password))
	}

	gitRepo, err = repo.New(ctx, r.Basedir, opts...)
	if err != nil {
		log.Info(ctx, "processGitClone> %s > cloning %s into %s", op.UUID, r.URL, r.Basedir)
		gitRepo, err = repo.Clone(ctx, r.Basedir, r.URL, opts...)
		if err != nil {
			if strings.Contains(err.Error(), "Invalid username or password") ||
				strings.Contains(err.Error(), "Permission denied (publickey)") ||
				strings.Contains(err.Error(), "could not read Username for") ||
				strings.Contains(err.Error(), "you do not have permission to access it") {
				return gitRepo, "", "", sdk.NewError(sdk.ErrForbidden, err)
			}
			return gitRepo, "", "", sdk.NewErrorFrom(err, "cannot clone repository at given url: %s", r.URL)
		}
	}

	f, err := gitRepo.FetchURL(ctx)
	if err != nil {
		return gitRepo, "", "", sdk.WithStack(err)
	}

	d, err := gitRepo.DefaultBranch(ctx)
	if err != nil {
		return gitRepo, "", "", sdk.WithStack(err)
	}

	op.RepositoryInfo = &sdk.OperationRepositoryInfo{
		Name:          op.RepoFullName,
		FetchURL:      f,
		DefaultBranch: d,
	}

	//Check branch
	currentBranch, err = gitRepo.CurrentBranch(ctx)
	if err != nil {
		return gitRepo, "", "", sdk.WithStack(err)
	}
	return gitRepo, r.Basedir, currentBranch, nil
}
