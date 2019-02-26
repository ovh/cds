package gitlab

import (
	"context"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
)

//Commits returns commit data from a given starting commit, between two commits
//The commits may be identified by branch or tag name or by hash.
func (c *gitlabClient) Commits(ctx context.Context, repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	// Gitlab commit listing only allow 'since' and 'until' parameter as dates
	// Need to fetch commit date, then use it to filter

	opt := &gitlab.ListCommitsOptions{
		RefName: &branch,
	}

	commit, err := c.Commit(ctx, repo, since)
	if err == nil {
		since := time.Unix(commit.Timestamp, 0)
		opt.Since = &since
	}

	commit, err = c.Commit(ctx, repo, until)
	if err == nil {
		since := time.Unix(commit.Timestamp, 0)
		opt.Since = &since
	}

	commits, _, err := c.client.Commits.ListCommits(repo, opt)
	if err != nil {
		return nil, err
	}

	vcscommits := make([]sdk.VCSCommit, len(commits))
	for i, c := range commits {
		vcscommits[i] = sdk.VCSCommit{
			Hash: c.ID,
			Author: sdk.VCSAuthor{
				Name:        c.AuthorName,
				DisplayName: c.AuthorName,
				Email:       c.AuthorEmail,
			},
			Timestamp: c.AuthoredDate.Unix() * 1000,
			Message:   c.Message,
		}
	}

	return vcscommits, nil
}

//Commit retrieves a specific according to a hash
func (c *gitlabClient) Commit(ctx context.Context, repo, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}

	gc, _, err := c.client.Commits.GetCommit(repo, hash)
	if err != nil {
		return commit, err
	}

	commit.Hash = hash
	commit.Author = sdk.VCSAuthor{
		Name:        gc.AuthorName,
		DisplayName: gc.AuthorName,
		Email:       gc.AuthorEmail,
	}
	commit.Timestamp = gc.AuthoredDate.Unix() * 1000
	commit.Message = gc.Message

	return commit, nil
}

func (c *gitlabClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	opt := &gitlab.CompareOptions{
		From: &base,
		To:   &head,
	}

	compare, _, err := c.client.Repositories.Compare(repo, opt)
	if err != nil {
		return nil, err
	}

	if compare == nil || compare.Commits == nil {
		return nil, nil
	}

	vcscommits := make([]sdk.VCSCommit, len(compare.Commits))
	for i, c := range compare.Commits {
		vcscommits[i] = sdk.VCSCommit{
			Hash: c.ID,
			Author: sdk.VCSAuthor{
				Name:        c.AuthorName,
				DisplayName: c.AuthorName,
				Email:       c.AuthorEmail,
			},
			Timestamp: c.AuthoredDate.Unix() * 1000,
			Message:   c.Message,
		}
	}

	return vcscommits, nil
}
