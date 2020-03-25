package repositories

import (
	"context"

	"github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processGitClone(ctx context.Context, op *sdk.Operation) (repo.Repo, string, string, error) {
	var gitRepo repo.Repo

	r := s.Repo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		return gitRepo, "", "", err
	}

	// Get the git repository
	opts := []repo.Option{repo.WithVerbose()}

	if op.RepositoryStrategy.ConnectionType == "ssh" {
		log.Debug("processGitClone> using ssh key %s", op.RepositoryStrategy.SSHKey)
		opts = append(opts, repo.WithSSHAuth([]byte(op.RepositoryStrategy.SSHKeyContent)))
	} else if op.RepositoryStrategy.User != "" && op.RepositoryStrategy.Password != "" {
		log.Debug("processGitClone> using user %s", op.RepositoryStrategy.User)
		opts = append(opts, repo.WithHTTPAuth(op.RepositoryStrategy.User, op.RepositoryStrategy.Password))
	}

	gitRepo, err := repo.New(r.Basedir, opts...)
	if err != nil {
		log.Info(ctx, "processGitClone> cloning %s into %s", r.URL, r.Basedir)
		gitRepo, err = repo.Clone(r.Basedir, r.URL, opts...)
		if err != nil {
			return gitRepo, "", "", err
		}
	}

	f, err := gitRepo.FetchURL()
	if err != nil {
		return gitRepo, "", "", err
	}

	d, err := gitRepo.DefaultBranch()
	if err != nil {
		return gitRepo, "", "", err
	}

	op.RepositoryInfo = &sdk.OperationRepositoryInfo{
		Name:          op.RepoFullName,
		FetchURL:      f,
		DefaultBranch: d,
	}

	//Check branch
	currentBranch, err := gitRepo.CurrentBranch()
	if err != nil {
		return gitRepo, "", "", err
	}
	return gitRepo, r.Basedir, currentBranch, nil
}
