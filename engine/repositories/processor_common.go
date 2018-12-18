package repositories

import (
	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processGitClone(op *sdk.Operation) (repo.Repo, string, error) {
	var gitRepo repo.Repo

	r := s.Repo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		log.Error("Repositories> processCheckout> checkOrCreateFS> [%s] Error %v", op.UUID, err)
		return gitRepo, "", err
	}

	// Get the git repository
	opts := []repo.Option{repo.WithVerbose()}
	if op.RepositoryStrategy.ConnectionType == "ssh" {
		log.Debug("Repositories> processCheckout> using ssh key %s", op.RepositoryStrategy.SSHKey)
		opts = append(opts, repo.WithSSHAuth([]byte(op.RepositoryStrategy.SSHKeyContent)))
	} else if op.RepositoryStrategy.User != "" && op.RepositoryStrategy.Password != "" {
		opts = append(opts, repo.WithHTTPAuth(op.RepositoryStrategy.User, op.RepositoryStrategy.Password))
	}

	gitRepo, err := repo.New(r.Basedir, opts...)
	if err != nil {
		log.Debug("Repositories> processCheckout> cloning %s into %s", r.URL, r.Basedir)
		if _, err = repo.Clone(r.Basedir, r.URL, opts...); err != nil {
			log.Error("Repositories> processCheckout> Clone> [%s] error %v", op.UUID, err)
			return gitRepo, "", err
		}
	}

	f, err := gitRepo.FetchURL()
	if err != nil {
		log.Error("Repositories> processCheckout> gitRepo.FetchURL> [%s] Error: %v", op.UUID, err)
		return gitRepo, "", err
	}
	d, err := gitRepo.DefaultBranch()
	if err != nil {
		log.Error("Repositories> processCheckout> DefaultBranch> [%s] Error: %v", op.UUID, err)
		return gitRepo, "", err
	}

	op.RepositoryInfo = &sdk.OperationRepositoryInfo{
		Name:          op.RepoFullName,
		FetchURL:      f,
		DefaultBranch: d,
	}

	//Check branch
	currentBranch, err := gitRepo.CurrentBranch()
	if err != nil {
		log.Error("Repositories> processCheckout> CurrentBranch> [%s] error %v", op.UUID, err)
		return gitRepo, "", err
	}

	return gitRepo, currentBranch, nil
}
