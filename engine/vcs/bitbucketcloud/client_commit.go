package bitbucketcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

// Commits returns the commits list on a branch between a commit SHA (since) until another commit SHA (until). The branch is given by the branch of the first commit SHA (since)
func (client *bitbucketcloudClient) Commits(ctx context.Context, repo, theBranch, since, until string) ([]sdk.VCSCommit, error) {
	var commitsResult []sdk.VCSCommit
	//Get Commit List
	theCommits, err := client.allCommitBetween(ctx, repo, since, until, theBranch)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load all commit between since=%s and until=%s on branch %s", since, until, theBranch)
	}

	commitsResult = make([]sdk.VCSCommit, 0, len(theCommits))
	//Convert to sdk.VCSCommit
	for _, c := range theCommits {
		email := strings.Trim(rawEmailCommitRegexp.FindString(c.Author.Raw), "<>")
		commit := sdk.VCSCommit{
			Timestamp: c.Date.Unix() * 1000,
			Message:   c.Message,
			Hash:      c.Hash,
			URL:       c.Links.HTML.Href,
			Author: sdk.VCSAuthor{
				DisplayName: c.Author.User.DisplayName,
				Email:       email,
				Name:        c.Author.User.Username,
				Avatar:      c.Author.User.Links.Avatar.Href,
			},
		}

		commitsResult = append(commitsResult, commit)
	}

	return commitsResult, nil
}

func (client *bitbucketcloudClient) allCommitBetween(ctx context.Context, repo, sinceCommit, untilCommit, branch string) ([]Commit, error) {
	var commits []Commit
	params := url.Values{}
	params.Add("exclude", sinceCommit)
	path := fmt.Sprintf("/repositories/%s/commits/%s", repo, untilCommit)
	nextPage := 1

	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 1 {
			params.Set("page", fmt.Sprintf("%d", nextPage))
		}

		var response Commits
		if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get commits")
		}
		if cap(commits) == 0 {
			commits = make([]Commit, 0, response.Size)
		}
		commits = append(commits, response.Values...)

		if response.Next == "" {
			break
		} else {
			nextPage++
		}
	}

	return commits, nil
}

// Commit Get a single commit
func (client *bitbucketcloudClient) Commit(ctx context.Context, repo, hash string) (sdk.VCSCommit, error) {
	var commit sdk.VCSCommit
	url := fmt.Sprintf("/repositories/%s/commit/%s", repo, hash)
	status, body, _, err := client.get(url)
	if err != nil {
		log.Warn(ctx, "bitbucketcloudClient.Commit> Error %s", err)
		return commit, err
	}
	if status >= 400 {
		return commit, sdk.NewError(sdk.ErrRepoNotFound, errorAPI(body))
	}
	var c Commit
	if err := sdk.JSONUnmarshal(body, &c); err != nil {
		log.Warn(ctx, "bitbucketcloudClient.Commit> Unable to parse bitbucket cloud commit: %s", err)
		return sdk.VCSCommit{}, err
	}

	email := strings.Trim(rawEmailCommitRegexp.FindString(c.Author.Raw), "<>")
	commit = sdk.VCSCommit{
		Timestamp: c.Date.Unix() * 1000,
		Message:   c.Message,
		Hash:      c.Hash,
		URL:       c.Links.HTML.Href,
		Author: sdk.VCSAuthor{
			DisplayName: c.Author.User.DisplayName,
			Email:       email,
			Name:        c.Author.User.Username,
			Avatar:      c.Author.User.Links.Avatar.Href,
		},
	}

	return commit, nil
}

func (client *bitbucketcloudClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	var commits []Commit
	if base == "" {
		base = "HEAD"
	}
	if head == "" {
		head = "HEAD"
	}
	params := url.Values{}
	params.Add("exclude", base)
	path := fmt.Sprintf("/repositories/%s/commits/%s", repo, head)
	nextPage := 1
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 1 {
			params.Set("page", fmt.Sprintf("%d", nextPage))
		}

		var response Commits
		if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get commits")
		}
		if cap(commits) == 0 {
			commits = make([]Commit, 0, response.Size)
		}
		commits = append(commits, response.Values...)

		if response.Next == "" {
			break
		} else {
			nextPage++
		}
	}

	commitsResult := make([]sdk.VCSCommit, 0, len(commits))
	//Convert to sdk.VCSCommit
	for _, c := range commits {
		email := strings.Trim(rawEmailCommitRegexp.FindString(c.Author.Raw), "<>")
		commit := sdk.VCSCommit{
			Timestamp: c.Date.Unix() * 1000,
			Message:   c.Message,
			Hash:      c.Hash,
			URL:       c.Links.HTML.Href,
			Author: sdk.VCSAuthor{
				DisplayName: c.Author.User.DisplayName,
				Email:       email,
				Name:        c.Author.User.Username,
				Avatar:      c.Author.User.Links.Avatar.Href,
			},
		}

		commitsResult = append(commitsResult, commit)
	}

	return commitsResult, nil
}
