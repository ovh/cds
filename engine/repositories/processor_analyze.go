package repositories

import (
	"context"
	"strings"

	"github.com/fsamin/go-repo"
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

	d, err := gitRepo.DefaultBranch(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "you do not have permission to access it") {
			return repo.Repo{}, sdk.NewError(sdk.ErrForbidden, err)
		}
		return repo.Repo{}, sdk.WithStack(err)
	}

	op.RepositoryInfo = &sdk.OperationRepositoryInfo{
		Name:          op.RepoFullName,
		FetchURL:      f,
		DefaultBranch: d,
	}
	return gitRepo, nil
}

// fetchAnalysisTarget fetches the refs the operation needs and returns the
// commit-ish all analyses must run on; a bare clone has no meaningful HEAD.
func (s *Service) fetchAnalysisTarget(ctx context.Context, gitRepo repo.Repo, op *sdk.Operation) (string, error) {
	if op.Setup.Checkout.Tag != "" {
		log.Debug(ctx, "fetchAnalysisTarget> fetching tags from %s", op.URL)
		if err := gitRepo.FetchRemoteTags(ctx, "origin"); err != nil {
			return "", sdk.WithStack(err)
		}
		log.Info(ctx, "fetchAnalysisTarget> repository %s ready on tag '%s'", op.URL, op.Setup.Checkout.Tag)
		return "refs/tags/" + op.Setup.Checkout.Tag, nil
	}

	if op.Setup.Checkout.Branch == "" {
		op.Setup.Checkout.Branch = op.RepositoryInfo.DefaultBranch
	}
	log.Debug(ctx, "fetchAnalysisTarget> fetching branch %s from %s", op.Setup.Checkout.Branch, op.URL)
	if err := gitRepo.FetchBranchWithoutCheckout(ctx, "origin", op.Setup.Checkout.Branch); err != nil {
		return "", sdk.WithStack(err)
	}

	// The changeset target branch never comes with the analyzed branch: a bare
	// clone has no default fetch refspec, every needed ref is fetched explicitly
	if op.Setup.Checkout.GetChangeSet && op.Setup.Checkout.ChangeSetBranchTo != "" && op.Setup.Checkout.ChangeSetBranchTo != op.Setup.Checkout.Branch {
		log.Debug(ctx, "fetchAnalysisTarget> fetching changeset target branch %s from %s", op.Setup.Checkout.ChangeSetBranchTo, op.URL)
		if err := gitRepo.FetchBranchWithoutCheckout(ctx, "origin", op.Setup.Checkout.ChangeSetBranchTo); err != nil {
			return "", sdk.WithStack(err)
		}
	}

	if op.Setup.Checkout.Commit != "" {
		return op.Setup.Checkout.Commit, nil
	}
	return "refs/heads/" + op.Setup.Checkout.Branch, nil
}

func (s *Service) processAnalyses(ctx context.Context, gitRepo repo.Repo, op *sdk.Operation, target string) error {
	return sdk.NewErrorFrom(sdk.ErrNotImplemented, "bare analysis cache processor")
}
