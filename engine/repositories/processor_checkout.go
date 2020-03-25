package repositories

import (
	"context"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) processCheckout(ctx context.Context, op *sdk.Operation) error {
	gitRepo, _, currentBranch, err := s.processGitClone(ctx, op)
	if err != nil {
		return sdk.WrapError(err, "unable to process gitclone")
	}

	if err := gitRepo.ResetHard("origin/" + currentBranch); err != nil {
		return sdk.WithStack(err)
	}

	if op.Setup.Checkout.Tag != "" {
		log.Debug("processCheckout> fetching tag %s from %s", op.Setup.Checkout.Tag, op.URL)
		if err := gitRepo.FetchRemoteTag("origin", op.Setup.Checkout.Tag); err != nil {
			return sdk.WithStack(err)
		}
	} else {
		if op.Setup.Checkout.Branch == "" {
			op.Setup.Checkout.Branch = op.RepositoryInfo.DefaultBranch
		}
		log.Debug("processCheckout> fetching branch %s from %s", op.Setup.Checkout.Branch, op.URL)
		if err := gitRepo.FetchRemoteBranch("origin", op.Setup.Checkout.Branch); err != nil {
			return sdk.WithStack(err)
		}
	}

	// Check commit
	if op.Setup.Checkout.Commit == "" {
		log.Debug("processCheckout> pulling branch %s", op.Setup.Checkout.Branch)
		if err := gitRepo.Pull("origin", op.Setup.Checkout.Branch); err != nil {
			return sdk.WithStack(err)
		}
	} else {
		currentCommit, err := gitRepo.LatestCommit()
		if err != nil {
			return sdk.WithStack(err)
		}
		if currentCommit.LongHash != op.Setup.Checkout.Commit {
			// Not the same commit, pull and reset HARD the commit
			log.Debug("processCheckout> resetting the branch %s from remote", op.Setup.Checkout.Branch)
			if err := gitRepo.ResetHard("origin/" + op.Setup.Checkout.Branch); err != nil {
				return sdk.WithStack(err)
			}

			log.Debug("Repositories> processCheckout> reseting commit %s", op.Setup.Checkout.Commit)
			if err := gitRepo.ResetHard(op.Setup.Checkout.Commit); err != nil {
				return sdk.WithStack(err)
			}
		}
	}

	log.Info(ctx, "processCheckout> repository %s ready", op.URL)
	return nil
}
