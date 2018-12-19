package repositories

import (
	"encoding/base64"
	"github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processGitClone(op *sdk.Operation) (repo.Repo, string, string, error) {
	var gitRepo repo.Repo

	r := s.Repo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		log.Error("Repositories> processGitClone> checkOrCreateFS> [%s] Error %v", op.UUID, err)
		return gitRepo, "", "", err
	}

	// Get the git repository
	opts := []repo.Option{repo.WithVerbose()}
	if op.RepositoryStrategy.ConnectionType == "ssh" {
		log.Debug("Repositories> processGitClone> using ssh key %s", op.RepositoryStrategy.SSHKey)
		opts = append(opts, repo.WithSSHAuth([]byte(op.RepositoryStrategy.SSHKeyContent)))
	} else if op.RepositoryStrategy.User != "" && op.RepositoryStrategy.Password != "" {
		// Decrypt base64 password

		decoded, err := base64.StdEncoding.DecodeString(op.RepositoryStrategy.Password)
		if err != nil {
			log.Error("Repositories> processGitClone> decoding password> [%s] Error %v", op.UUID, err)
			return gitRepo, "", "", err
		}

		opts = append(opts, repo.WithHTTPAuth(op.RepositoryStrategy.User, string(decoded)))
	}

	gitRepo, err := repo.New(r.Basedir, opts...)
	if err != nil {
		log.Info("Repositories> processGitClone> cloning %s into %s", r.URL, r.Basedir)
		if _, err = repo.Clone(r.Basedir, r.URL, opts...); err != nil {
			log.Error("Repositories> processGitClone> Clone> [%s] error %v", op.UUID, err)
			return gitRepo, "", "", err
		}
	}

	f, err := gitRepo.FetchURL()
	if err != nil {
		log.Error("Repositories> processGitClone> gitRepo.FetchURL> [%s] Error: %v", op.UUID, err)
		return gitRepo, "", "", err
	}
	d, err := gitRepo.DefaultBranch()
	if err != nil {
		log.Error("Repositories> processGitClone> DefaultBranch> [%s] Error: %v", op.UUID, err)
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
		log.Error("Repositories> processGitClone> CurrentBranch> [%s] error %v", op.UUID, err)
		return gitRepo, "", "", err
	}
	return gitRepo, r.Basedir, currentBranch, nil
}
