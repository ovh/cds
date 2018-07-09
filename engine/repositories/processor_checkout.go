package repositories

import (
	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processCheckout(op *sdk.Operation) error {
	r := s.Repo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		log.Error("Repositories> processCheckout> checkOrCreateFS> [%s] Error %v", op.UUID, err)
		return err
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
			return err
		}
	}

	f, err := gitRepo.FetchURL()
	if err != nil {
		log.Error("Repositories> processCheckout> gitRepo.FetchURL> [%s] Error: %v", op.UUID, err)
		return err
	}
	d, err := gitRepo.DefaultBranch()
	if err != nil {
		log.Error("Repositories> processCheckout> DefaultBranch> [%s] Error: %v", op.UUID, err)
		return err
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
		return err
	}

	if op.Setup.Checkout.Branch == "" {
		op.Setup.Checkout.Branch = d
	}

	if currentBranch != op.Setup.Checkout.Branch {
		log.Debug("Repositories> processCheckout> fetching branch %s from %s", op.Setup.Checkout.Branch, r.URL)
		if err := gitRepo.FetchRemoteBranch("origin", op.Setup.Checkout.Branch); err != nil {
			log.Error("Repositories> processCheckout> FetchRemoteBranch> [%s] error %v", op.UUID, err)
			return err
		}
	}

	//Check commit
	if op.Setup.Checkout.Commit == "" {
		log.Debug("Repositories> processCheckout> pulling branch %s", op.Setup.Checkout.Branch)
		if err := gitRepo.Pull("origin", op.Setup.Checkout.Branch); err != nil {
			log.Error("Repositories> processCheckout> Pull without commit> [%s] error %v", op.UUID, err)
			return err
		}
	} else {
		currentCommit, err := gitRepo.LatestCommit()
		if err != nil {
			log.Error("Repositories> processCheckout> LatestCommit> [%s] error %v", op.UUID, err)
			return err
		}
		if currentCommit.LongHash != op.Setup.Checkout.Commit {
			//Not the same commit
			//Pull and reset HARD the commit
			log.Debug("Repositories> processCheckout> pulling branch %s", op.Setup.Checkout.Branch)
			if err := gitRepo.Pull("origin", op.Setup.Checkout.Branch); err != nil {
				log.Error("Repositories> processCheckout> Pull with commit > [%s] error %v", op.UUID, err)
				return err
			}

			log.Debug("Repositories> processCheckout> reseting commit %s", op.Setup.Checkout.Commit)
			if err := gitRepo.ResetHard(op.Setup.Checkout.Commit); err != nil {
				log.Error("Repositories> processCheckout> ResetHard> [%s] error %v", op.UUID, err)
				return err
			}
		}
	}

	log.Info("Repositories> processCheckout> repository %s ready", r.URL)
	return nil
}
