package gitea

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) Commits(_ context.Context, fullname, branch, since, until string) ([]sdk.VCSCommit, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, fmt.Errorf("giteaClient.Commits> invalid fullname %s", fullname)
	}

	commits, _, err := g.client.ListRepoCommits(t[0], t[1], gitea.ListCommitOptions{})
	//allCommitBetween(ctx, repo, since, until, theBranch)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load commits on branch %s", branch)
	}

	commitsResult := make([]sdk.VCSCommit, 0, len(commits))
	//Convert to sdk.VCSCommit
	for _, c := range commits {
		email := c.Author.Email
		commit := sdk.VCSCommit{
			Timestamp: c.Created.Unix() * 1000,
			Message:   c.RepoCommit.Message,
			Hash:      c.RepoCommit.Tree.SHA,
			URL:       c.RepoCommit.URL,
			Author: sdk.VCSAuthor{
				DisplayName: c.Author.FullName,
				Email:       email,
				Name:        c.Author.UserName,
				Avatar:      c.Author.AvatarURL,
				ID:          fmt.Sprintf("%d", c.Author.ID),
			},
		}

		commitsResult = append(commitsResult, commit)
	}

	return commitsResult, nil
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

func (g *giteaClient) toVCSCommit(commit *gitea.Commit) sdk.VCSCommit {
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
