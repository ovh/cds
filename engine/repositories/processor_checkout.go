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
	log.Debug("processCheckout> %s >repo cloned with current branch: %s", op.UUID, currentBranch)

	// Clean no commited changes if exists
	if err := gitRepo.ResetHard(ctx, "HEAD"); err != nil {
		return sdk.WithStack(err)
	}
	log.Debug("processCheckout> %s >repo reset to HEAD", op.UUID)

	if op.Setup.Checkout.Tag != "" {
		log.Debug("processCheckout> %s >fetching tag %s from %s", op.UUID, op.Setup.Checkout.Tag, op.URL)
		if err := gitRepo.FetchRemoteTag(ctx, "origin", op.Setup.Checkout.Tag); err != nil {
			return sdk.WithStack(err)
		}
		log.Info(ctx, "processCheckout> %s >repository %s ready on tag '%s'", op.UUID, op.URL, op.Setup.Checkout.Tag)
		return nil
	}

	if op.Setup.Checkout.Branch == "" {
		op.Setup.Checkout.Branch = op.RepositoryInfo.DefaultBranch
	}
	log.Debug("processCheckout> %s >fetching branch %s from %s", op.UUID, op.Setup.Checkout.Branch, op.URL)
	if err := gitRepo.FetchRemoteBranch(ctx, "origin", op.Setup.Checkout.Branch); err != nil {
		return sdk.WithStack(err)
	}

	// Check commit
	if op.Setup.Checkout.Commit == "" {
		// Reset HARD to the latest commit of the remote branch (don't use pull because there can be conflicts if the remote was forced)
		log.Debug("processCheckout> %s >resetting the branch %s from remote", op.UUID, op.Setup.Checkout.Branch)
		if err := gitRepo.ResetHard(ctx, "origin/"+op.Setup.Checkout.Branch); err != nil {
			return sdk.WithStack(err)
		}
	} else {
		currentCommit, err := gitRepo.LatestCommit(ctx)
		if err != nil {
			return sdk.WithStack(err)
		}
		if currentCommit.LongHash != op.Setup.Checkout.Commit {
			// Not the same commit, pull and reset HARD the commit
			log.Debug("processCheckout> %s >resetting the branch %s from remote", op.UUID, op.Setup.Checkout.Branch)
			if err := gitRepo.ResetHard(ctx, "origin/"+op.Setup.Checkout.Branch); err != nil {
				return sdk.WithStack(err)
			}

			log.Debug("Repositories> processCheckout> %s >resetting commit %s", op.UUID, op.Setup.Checkout.Commit)
			if err := gitRepo.ResetHard(ctx, op.Setup.Checkout.Commit); err != nil {
				return sdk.WithStack(err)
			}
		}
	}

	log.Info(ctx, "processCheckout> %s >repository %s ready", op.UUID, op.URL)
	return nil
}
