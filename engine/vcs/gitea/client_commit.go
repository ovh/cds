package gitea

import (
	"context"
	"strings"

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
	if giteaCommit.RepoCommit == nil || giteaCommit.RepoCommit.Author == nil {
		return sdk.VCSCommit{}, sdk.NewErrorFrom(sdk.ErrNotFound, "commit data not found")
	}
	return g.toVCSCommit(giteaCommit), nil
}

func (g *giteaClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	return nil, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) toVCSCommit(commit *gg.Commit) sdk.VCSCommit {
	vcsCommit := sdk.VCSCommit{
		KeySignID: "",
		Verified:  false,
		Message:   commit.RepoCommit.Message,
		Hash:      commit.SHA,
		URL:       commit.URL,
		Author: sdk.VCSAuthor{
			Name:        commit.Committer.UserName,
			Avatar:      commit.Committer.AvatarURL,
			Email:       commit.Committer.Email,
			DisplayName: commit.Committer.UserName,
		},
		Timestamp: commit.Created.Unix() * 1000,
	}
	if commit.RepoCommit.Verification != nil && commit.RepoCommit.Verification.Signature != "" {
		reasonSplitted := strings.Split(commit.RepoCommit.Verification.Reason, " ")
		vcsCommit.KeySignID = reasonSplitted[len(reasonSplitted)-1]
		vcsCommit.Verified = commit.RepoCommit.Verification.Verified

	}
	return vcsCommit
}
