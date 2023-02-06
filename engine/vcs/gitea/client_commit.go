package gitea

import (
	"context"
	"strconv"

	gg "code.gitea.io/sdk/gitea"

	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) Commits(_ context.Context, repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) Commit(_ context.Context, repo, hash string) (sdk.VCSCommit, error) {
	owner, repo, err := getRepo(repo)
	if err != nil {
		return sdk.VCSCommit{}, err
	}
	giteaCommit, _, err := g.client.GetSingleCommit(owner, repo, hash)
	if err != nil {
		return sdk.VCSCommit{}, err
	}
	if giteaCommit.RepoCommit == nil {
		return sdk.VCSCommit{}, sdk.NewErrorFrom(sdk.ErrNotFound, "commit %s data not found", hash)
	}
	return g.toVCSCommit(giteaCommit), nil
}

func (g *giteaClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) toVCSCommit(commit *gg.Commit) sdk.VCSCommit {

	vcsCommit := sdk.VCSCommit{
		Signature: "",
		Verified:  false,
		Message:   commit.RepoCommit.Message,
		Hash:      commit.SHA,
		URL:       commit.URL,
		Timestamp: commit.Created.Unix() * 1000,
	}
	if commit.Author != nil {
		vcsCommit.Author = sdk.VCSAuthor{
			Name:        commit.Author.UserName,
			Avatar:      commit.Author.AvatarURL,
			Email:       commit.Author.Email,
			DisplayName: commit.Author.UserName,
			Slug:        commit.Author.UserName,
			ID:          strconv.FormatInt(commit.Author.ID, 10),
		}
	}
	if commit.Committer != nil {
		vcsCommit.Committer = sdk.VCSAuthor{
			Name:        commit.Committer.UserName,
			Avatar:      commit.Committer.AvatarURL,
			Email:       commit.Committer.Email,
			DisplayName: commit.Committer.UserName,
			ID:          strconv.FormatInt(commit.Committer.ID, 10),
		}
	}

	if commit.RepoCommit.Verification != nil && commit.RepoCommit.Verification.Signature != "" {
		vcsCommit.Signature = commit.RepoCommit.Verification.Signature
		vcsCommit.Verified = commit.RepoCommit.Verification.Verified
	}
	return vcsCommit
}
