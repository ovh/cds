package repositories

import (
	"context"

	"github.com/fsamin/go-repo"
	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

var vcsPublicKeys map[string][]sdk.Key

func (s *Service) processCheckout(ctx context.Context, op *sdk.Operation) error {
	gitRepo, _, currentBranch, err := s.processGitClone(ctx, op)
	if err != nil {
		return sdk.WrapError(err, "unable to process gitclone")
	}
	log.Debug(ctx, "processCheckout> repo cloned with current branch: %s", currentBranch)

	// Clean no commited changes if exists
	if err := gitRepo.ResetHard(ctx, "HEAD"); err != nil {
		return sdk.WithStack(err)
	}
	log.Debug(ctx, "processCheckout> repo reset to HEAD")

	if op.Setup.Checkout.Tag != "" {
		log.Debug(ctx, "processCheckout> fetching tag %s from %s", op.Setup.Checkout.Tag, op.URL)
		if err := gitRepo.FetchRemoteTag(ctx, "origin", op.Setup.Checkout.Tag); err != nil {
			return sdk.WithStack(err)
		}
		log.Info(ctx, "processCheckout> repository %s ready on tag '%s'", op.URL, op.Setup.Checkout.Tag)
	} else {
		if op.Setup.Checkout.Branch == "" {
			op.Setup.Checkout.Branch = op.RepositoryInfo.DefaultBranch
		}
		log.Debug(ctx, "processCheckout> fetching branch %s from %s", op.Setup.Checkout.Branch, op.URL)
		if err := gitRepo.FetchRemoteBranch(ctx, "origin", op.Setup.Checkout.Branch); err != nil {
			return sdk.WithStack(err)
		}
		// Check commit
		if op.Setup.Checkout.Commit == "" {
			// Reset HARD to the latest commit of the remote branch (don't use pull because there can be conflicts if the remote was forced)
			log.Debug(ctx, "processCheckout> resetting the branch %s from remote", op.Setup.Checkout.Branch)
			if err := gitRepo.ResetHard(ctx, "origin/"+op.Setup.Checkout.Branch); err != nil {
				return sdk.WithStack(err)
			}
		} else {
			currentCommit, err := gitRepo.LatestCommit(ctx, repo.CommitOption{DisableDiffDetail: true})
			if err != nil {
				return sdk.WithStack(err)
			}
			if currentCommit.LongHash != op.Setup.Checkout.Commit {
				// Not the same commit, pull and reset HARD the commit
				log.Debug(ctx, "processCheckout> resetting the branch %s from remote", op.Setup.Checkout.Branch)
				if err := gitRepo.ResetHard(ctx, "origin/"+op.Setup.Checkout.Branch); err != nil {
					return sdk.WithStack(err)
				}

				log.Debug(ctx, "processCheckout> resetting commit %s", op.Setup.Checkout.Commit)
				if err := gitRepo.ResetHard(ctx, op.Setup.Checkout.Commit); err != nil {
					return sdk.WithStack(err)
				}
			}
		}
	}

	if op.Setup.Checkout.GetMessage {
		currentCommit, err := gitRepo.LatestCommit(ctx, repo.CommitOption{DisableDiffDetail: true})
		if err != nil {
			return err
		}
		op.Setup.Checkout.Result.CommitMessage = currentCommit.Subject
		op.Setup.Checkout.Result.Author = currentCommit.Author
		op.Setup.Checkout.Result.AuthorEmail = currentCommit.AuthorEmail
	}

	if op.Setup.Checkout.ProcessSemver {
		describe, err := gitRepo.Describe(ctx, &repo.DescribeOpt{
			DirtySemver: true,
			Long:        true,
			Match:       []string{"v[0-9]*", "[0-9]*"},
			DirtyMark:   "-dirty",
		})
		if err != nil {
			log.ErrorWithStackTrace(ctx, errors.Wrap(err, "git describe failed"))
		} else {
			if describe.Semver != nil {
				op.Setup.Checkout.Result.Semver.Current = describe.SemverString
				op.Setup.Checkout.Result.Semver.Next = describe.Semver.IncMinor().String()
			}
		}
	}

	if err := s.checkCommitSignature(ctx, gitRepo, op, nil); err != nil {
		return err
	}

	if op.Setup.Checkout.GetChangeSet {
		op.Setup.Checkout.Result.Files = make(map[string]sdk.OperationChangetsetFile)
		computeFromLastCommit := false
		if op.Setup.Checkout.ChangeSetBranchTo != "" {
			files, err := gitRepo.DiffBetweenBranches(ctx, op.Setup.Checkout.Branch, op.Setup.Checkout.ChangeSetBranchTo)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				computeFromLastCommit = true
			} else {
				for k, v := range files {
					op.Setup.Checkout.Result.Files[k] = sdk.OperationChangetsetFile{
						Filename: v.Filename,
						Status:   v.Status,
					}
				}
			}
		} else if op.Setup.Checkout.ChangeSetCommitSince != "" {
			files, err := gitRepo.DiffSinceCommitMergeBase(ctx, op.Setup.Checkout.ChangeSetCommitSince)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				computeFromLastCommit = true
			} else {
				for k, v := range files {
					op.Setup.Checkout.Result.Files[k] = sdk.OperationChangetsetFile{
						Filename: v.Filename,
						Status:   v.Status,
					}
				}
			}
		} else {
			computeFromLastCommit = true
		}

		if computeFromLastCommit {
			commitWithChangesets, err := gitRepo.GetCommit(ctx, op.Setup.Checkout.Commit, repo.CommitOption{DisableDiffDetail: true})
			if err != nil {
				return err
			}

			for k, v := range commitWithChangesets.Files {
				op.Setup.Checkout.Result.Files[k] = sdk.OperationChangetsetFile{
					Filename: v.Filename,
					Status:   v.Status,
				}
			}
		}

	}

	log.Info(ctx, "processCheckout> repository %s ready", op.URL)
	return nil
}
