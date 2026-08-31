package repositories

import (
	"context"
	"os"
	"strings"

	"github.com/fsamin/go-repo"
	"github.com/pkg/errors"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// useBareAnalysisCache returns true when the operation only analyzes git data
// (signature, semver, changeset) and can run on the bare clones cache.
func (s *Service) useBareAnalysisCache(op sdk.Operation) bool {
	if !s.Cfg.BareAnalysisCache {
		return false
	}
	if op.Setup.Checkout.Branch == "" && op.Setup.Checkout.Tag == "" {
		return false
	}
	if op.LoadFiles.Pattern != "" || op.Setup.Push.FromBranch != "" {
		return false
	}
	return op.Setup.Checkout.CheckSignature || op.Setup.Checkout.GetChangeSet || op.Setup.Checkout.ProcessSemver
}

func (s *Service) processAnalyzeBare(ctx context.Context, op *sdk.Operation) error {
	gitRepo, err := s.processGitCloneBare(ctx, op)
	if err != nil {
		return sdk.WrapError(err, "unable to process bare git clone")
	}

	// Fetch the data we need
	// * the current ref (branch or tag)
	// * changeset branch if needed ( compute changeset for a pr )
	// -> return the current commit ( for tags or the branch name)
	target, err := s.fetchAnalysisTarget(ctx, gitRepo, op)
	if err != nil {
		return err
	}

	return s.processAnalyses(ctx, gitRepo, op, target)
}

// processGitCloneBare opens the cached bare partial clone of the operation
// repository, cloning it first if needed, and fills op.RepositoryInfo.
func (s *Service) processGitCloneBare(ctx context.Context, op *sdk.Operation) (repo.Repo, error) {
	r := s.BareRepo(*op)
	if err := s.checkOrCreateFS(r); err != nil {
		return repo.Repo{}, err
	}

	// Get the git repository
	opts := []repo.Option{repo.WithVerbose(func(format string, args ...interface{}) { log.Info(ctx, format, args...) })}

	if op.RepositoryStrategy.ConnectionType == "ssh" {
		log.Debug(ctx, "processGitCloneBare> %s > using ssh key %s", op.UUID, op.RepositoryStrategy.SSHKey)
		opts = append(opts, repo.WithSSHAuth([]byte(op.RepositoryStrategy.SSHKeyContent)))
	} else if op.RepositoryStrategy.User != "" && op.RepositoryStrategy.Password != "" {
		log.Debug(ctx, "processGitCloneBare> %s > using user %s", op.UUID, op.RepositoryStrategy.User)
		opts = append(opts, repo.WithHTTPAuth(op.RepositoryStrategy.User, op.RepositoryStrategy.Password))
	}

	bareRepo, err := repo.NewBare(ctx, r.Basedir, opts...)
	if err != nil {
		// A non-empty directory that is not a bare repository is a clone that was
		// interrupted or corrupted; git refuses to clone into it, so start over.
		if empty, errDir := isEmptyDir(r.Basedir); errDir != nil {
			return repo.Repo{}, sdk.WithStack(errDir)
		} else if !empty {
			log.Warn(ctx, "processGitCloneBare> %s > %s is not a valid bare repository (%v), removing it", op.UUID, r.Basedir, err)
			if err := os.RemoveAll(r.Basedir); err != nil {
				return repo.Repo{}, sdk.WrapError(err, "unable to remove broken bare repository %s", r.Basedir)
			}
			if err := s.checkOrCreateFS(r); err != nil {
				return repo.Repo{}, err
			}
		}
		log.Info(ctx, "processGitCloneBare> %s > cloning %s into %s", op.UUID, r.URL, r.Basedir)
		if _, err := repo.CloneBare(ctx, r.Basedir, r.URL, append(opts, repo.WithFilter("blob:none"))...); err != nil {
			if strings.Contains(err.Error(), "Invalid username or password") ||
				strings.Contains(err.Error(), "Permission denied (publickey)") ||
				strings.Contains(err.Error(), "could not read Username for") ||
				strings.Contains(err.Error(), "you do not have permission to access it") {
				return repo.Repo{}, sdk.NewError(sdk.ErrForbidden, err)
			}
			return repo.Repo{}, sdk.NewErrorFrom(err, "cannot clone repository at given url: %s", r.URL)
		}
		bareRepo, err = repo.NewBare(ctx, r.Basedir, opts...)
		if err != nil {
			return repo.Repo{}, sdk.WithStack(err)
		}
	}
	gitRepo := bareRepo.Repo()

	f, err := gitRepo.FetchURL(ctx)
	if err != nil {
		return repo.Repo{}, sdk.WithStack(err)
	}

	op.RepositoryInfo = &sdk.OperationRepositoryInfo{Name: op.RepoFullName, FetchURL: f}

	// The default branch costs a remote round-trip (git remote show); analyses
	// only need it as the fallback target when neither branch nor tag is given
	if op.Setup.Checkout.Branch == "" && op.Setup.Checkout.Tag == "" {
		d, err := gitRepo.DefaultBranch(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "you do not have permission to access it") {
				return repo.Repo{}, sdk.NewError(sdk.ErrForbidden, err)
			}
			return repo.Repo{}, sdk.WithStack(err)
		}
		op.RepositoryInfo.DefaultBranch = d
	}
	return gitRepo, nil
}

// fetchAnalysisTarget fetches the refs the operation needs and returns the
// commit all analyses must run on; a bare clone has no meaningful HEAD.
func (s *Service) fetchAnalysisTarget(ctx context.Context, gitRepo repo.Repo, op *sdk.Operation) (string, error) {
	if op.Setup.Checkout.Tag != "" {
		log.Debug(ctx, "fetchAnalysisTarget> fetching tags from %s", op.URL)
		if err := gitRepo.FetchRemoteTags(ctx, "origin"); err != nil {
			return "", sdk.WithStack(err)
		}
		if err := s.fetchChangesetBranches(ctx, gitRepo, op); err != nil {
			return "", err
		}
		// Peel to the commit: annotated tags are objects of their own
		target, err := gitRepo.RevParse(ctx, "refs/tags/"+op.Setup.Checkout.Tag)
		if err != nil {
			return "", sdk.NewErrorFrom(sdk.ErrNotFound, "tag %s not found on %s", op.Setup.Checkout.Tag, op.URL)
		}
		log.Info(ctx, "fetchAnalysisTarget> repository %s ready on tag '%s' (%s)", op.URL, op.Setup.Checkout.Tag, target)
		return target, nil
	}

	if op.Setup.Checkout.Branch == "" {
		op.Setup.Checkout.Branch = op.RepositoryInfo.DefaultBranch
	}
	log.Debug(ctx, "fetchAnalysisTarget> fetching branch %s from %s", op.Setup.Checkout.Branch, op.URL)
	if err := gitRepo.FetchBranchWithoutCheckout(ctx, "origin", op.Setup.Checkout.Branch); err != nil {
		return "", sdk.WithStack(err)
	}
	if err := s.fetchChangesetBranches(ctx, gitRepo, op); err != nil {
		return "", err
	}

	if op.Setup.Checkout.Commit != "" {
		return op.Setup.Checkout.Commit, nil
	}
	return "refs/heads/" + op.Setup.Checkout.Branch, nil
}

// changesetBranchTo returns the branch name a PR changeset compares against:
// the field always carries a full ref ("refs/heads/x"), as the hooks send it.
func changesetBranchTo(op *sdk.Operation) string {
	return strings.TrimPrefix(op.Setup.Checkout.ChangeSetBranchTo, sdk.GitRefBranchPrefix)
}

// fetchChangesetBranches fetches the branches a branch-to-branch changeset
// compares: a bare clone has no default fetch refspec, every ref is explicit.
func (s *Service) fetchChangesetBranches(ctx context.Context, gitRepo repo.Repo, op *sdk.Operation) error {
	if !op.Setup.Checkout.GetChangeSet || op.Setup.Checkout.ChangeSetBranchTo == "" {
		return nil
	}
	branches := []string{changesetBranchTo(op)}
	if op.Setup.Checkout.Tag != "" && op.Setup.Checkout.Branch != "" {
		branches = append(branches, op.Setup.Checkout.Branch)
	}
	for _, b := range branches {
		if b == op.Setup.Checkout.Branch && op.Setup.Checkout.Tag == "" {
			continue // already fetched as the analyzed branch
		}
		log.Debug(ctx, "fetchAnalysisTarget> fetching changeset branch %s from %s", b, op.URL)
		if err := gitRepo.FetchBranchWithoutCheckout(ctx, "origin", b); err != nil {
			return sdk.WithStack(err)
		}
	}
	return nil
}

// processAnalyses runs the requested analyses against target, without any
// checkout: on a bare clone every git command names the commit-ish it reads.
func (s *Service) processAnalyses(ctx context.Context, gitRepo repo.Repo, op *sdk.Operation, target string) error {
	// Every step reads the same commit: fetch it once, on demand
	var commit *repo.Commit
	getCommit := func() (*repo.Commit, error) {
		if commit != nil {
			return commit, nil
		}
		c, err := gitRepo.GetCommit(ctx, target, repo.CommitOption{DisableDiffDetail: true})
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		commit = &c
		return commit, nil
	}

	if op.Setup.Checkout.GetMessage {
		c, err := getCommit()
		if err != nil {
			return err
		}
		op.Setup.Checkout.Result.CommitMessage = c.Subject
		op.Setup.Checkout.Result.Author = c.Author
		op.Setup.Checkout.Result.AuthorEmail = c.AuthorEmail
	}

	if op.Setup.Checkout.ProcessSemver {
		describe, err := gitRepo.Describe(ctx, &repo.DescribeOpt{
			Long:  true,
			Match: []string{"v[0-9]*", "[0-9]*"},
			Ref:   target,
		})
		if err != nil {
			log.ErrorWithStackTrace(ctx, errors.Wrap(err, "git describe failed"))
		} else if describe.Semver != nil {
			op.Setup.Checkout.Result.Semver.Current = describe.SemverString
			op.Setup.Checkout.Result.Semver.Next = describe.Semver.IncMinor().String()
		}
	}

	// The signature of a commit is read from the commit itself; tags are read apart
	var signed *repo.Commit
	if op.Setup.Checkout.CheckSignature && op.Setup.Checkout.Tag == "" && op.Setup.Checkout.Commit != "" {
		c, err := getCommit()
		if err != nil {
			return err
		}
		signed = c
	}
	if err := s.checkCommitSignature(ctx, gitRepo, op, signed); err != nil {
		return err
	}

	if op.Setup.Checkout.GetChangeSet {
		computeFromLastCommit := false
		if op.Setup.Checkout.ChangeSetBranchTo != "" {
			// Full refs: a tag named like a branch would otherwise win the resolution
			files, err := gitRepo.DiffBetweenBranches(ctx, sdk.GitRefBranchPrefix+op.Setup.Checkout.Branch, sdk.GitRefBranchPrefix+changesetBranchTo(op))
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				computeFromLastCommit = true
			} else {
				setChangesetFiles(op, files)
			}
		} else if op.Setup.Checkout.ChangeSetCommitSince != "" {
			files, err := gitRepo.DiffMergeBase(ctx, op.Setup.Checkout.ChangeSetCommitSince, target)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				computeFromLastCommit = true
			} else {
				setChangesetFiles(op, files)
			}
		} else {
			computeFromLastCommit = true
		}

		if computeFromLastCommit {
			c, err := getCommit()
			if err != nil {
				return err
			}
			setChangesetFiles(op, c.Files)
		}
	}

	log.Info(ctx, "processAnalyses> repository %s ready", op.URL)
	return nil
}

// setChangesetFiles records a git diff as the operation changeset.
func setChangesetFiles(op *sdk.Operation, files map[string]repo.File) {
	op.Setup.Checkout.Result.Files = make(map[string]sdk.OperationChangetsetFile, len(files))
	for k, v := range files {
		op.Setup.Checkout.Result.Files[k] = sdk.OperationChangetsetFile{Filename: v.Filename, Status: v.Status}
	}
}
