package repositories

import (
	"fmt"

	repo "github.com/fsamin/go-repo"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processCheckout(op *sdk.Operation) error {
	r := s.Repo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		log.Error("Repositories> processCheckout> Error %v", err)
		return err
	}

	// Get the git repository
	opts := []repo.Option{repo.WithVerbose()}
	if op.RepositoryStrategy.ConnectionType == "ssh" {
		opts = append(opts, repo.WithSSHAuth([]byte(op.RepositoryStrategy.SSHKey)))
	} else if op.RepositoryStrategy.User != "" && op.RepositoryStrategy.Password != "" {
		opts = append(opts, repo.WithHTTPAuth(op.RepositoryStrategy.User, op.RepositoryStrategy.Password))
	}

	gitRepo, err := repo.New(r.Basedir, opts...)
	if err != nil {
		log.Debug("Repositories> processCheckout> cloning %s", r.URL)
		if _, err = repo.Clone(r.Basedir, r.URL, opts...); err != nil {
			log.Error("Repositories> processCheckout> error %v", err)
			return err
		}
	}

	n, err := gitRepo.Name()
	if err != nil {
		log.Error("Repositories> processCheckout> Error: %v", err)
		return err
	}
	f, err := gitRepo.FetchURL()
	if err != nil {
		log.Error("Repositories> processCheckout> Error: %v", err)
		return err
	}
	d, err := gitRepo.DefaultBranch()
	if err != nil {
		log.Error("Repositories> processCheckout> Error: %v", err)
		return err
	}

	op.RepositoryInfo = &sdk.OperationRepositoryInfo{
		Name:          n,
		FetchURL:      f,
		DefaultBranch: d,
	}

	//Check branch
	currentBranch, err := gitRepo.CurrentBranch()
	if err != nil {
		log.Error("Repositories> processCheckout> error %v", err)
		return err
	}

	if op.Setup.Checkout.Branch == "" {
		op.Setup.Checkout.Branch, err = gitRepo.DefaultBranch()
		if err != nil {
			log.Error("Repositories> processCheckout> error %v", err)
			return err
		}
		if op.Setup.Checkout.Branch == "" {
			err = fmt.Errorf("unable go get default branch")
			log.Error("Repositories> processCheckout> error %v", err)
			return err
		}
	}

	if currentBranch != op.Setup.Checkout.Branch {
		log.Debug("Repositories> processCheckout> fetching branch %s from %s", op.Setup.Checkout.Branch, r.URL)
		if err := gitRepo.FetchRemoteBranch("origin", op.Setup.Checkout.Branch); err != nil {
			log.Error("Repositories> processCheckout> error %v", err)
			return err
		}
	}

	//Check commit
	if op.Setup.Checkout.Commit == "" {
		log.Debug("Repositories> processCheckout> pulling branch %s", op.Setup.Checkout.Branch)
		if err := gitRepo.Pull("origin", op.Setup.Checkout.Branch); err != nil {
			log.Error("Repositories> processCheckout> error %v", err)
			return err
		}
	} else {
		currentCommit, err := gitRepo.LatestCommit()
		if err != nil {
			log.Error("Repositories> processCheckout> error %v", err)
			return err
		}
		if currentCommit.LongHash != op.Setup.Checkout.Commit {
			//Not the same commit
			//Pull and reset HARD the commit
			log.Debug("Repositories> processCheckout> pulling branch %s", op.Setup.Checkout.Branch)
			if err := gitRepo.Pull("origin", op.Setup.Checkout.Branch); err != nil {
				log.Error("Repositories> processCheckout> error %v", err)
				return err
			}

			log.Debug("Repositories> processCheckout> reseting commit %s", op.Setup.Checkout.Commit)
			if err := gitRepo.ResetHard(op.Setup.Checkout.Commit); err != nil {
				log.Error("Repositories> processCheckout> error %v", err)
				return err
			}
		}
	}

	log.Info("Repositories> processCheckout> repository %s ready", r.URL)
	return nil
}
