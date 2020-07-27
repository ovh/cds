package bitbucketserver

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (b *bitbucketClient) Commits(ctx context.Context, repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	commits := []sdk.VCSCommit{}
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	stashCommits := []Commit{}

	var stashCommitsKey = cache.Key("vcs", "bitbucket", b.consumer.URL, repo, "commits", "since@"+since, "until@"+until)

	find, err := b.consumer.cache.Get(stashCommitsKey, &stashCommits)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", stashCommitsKey, err)
	}
	if !find {
		response := CommitsResponse{}
		path := fmt.Sprintf("/projects/%s/repos/%s/commits", project, slug)
		params := url.Values{}
		if since != "" {
			params.Add("since", since)
		}
		if until != "" {
			params.Add("until", until)
		}

		for {
			if ctx.Err() != nil {
				break
			}

			if response.NextPageStart != 0 {
				params.Set("start", fmt.Sprintf("%d", response.NextPageStart))
			}

			if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					return nil, nil
				}
				return nil, sdk.WrapError(err, "Unable to get commits %s", path)
			}

			stashCommits = append(stashCommits, response.Values...)
			if response.IsLastPage {
				break
			}
		}
		//3 hours
		if err := b.consumer.cache.SetWithTTL(stashCommitsKey, stashCommits, 3*60*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", stashCommitsKey, err)
		}
	}

	urlCommit := b.consumer.URL + "/projects/" + project + "/repos/" + slug + "/commits/"
	for _, sc := range stashCommits {
		c := sdk.VCSCommit{
			Hash:      sc.Hash,
			Timestamp: sc.Timestamp,
			Message:   sc.Message,
			Author: sdk.VCSAuthor{
				Name:        sc.Author.Name,
				Email:       sc.Author.Email,
				DisplayName: sc.Author.DisplayName,
			},
			URL: urlCommit + sc.Hash,
		}
		if sc.Author.Slug != "" && sc.Author.Slug != "unknownSlug" {
			c.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", b.consumer.URL, sc.Author.Slug)
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func (b *bitbucketClient) Commit(ctx context.Context, repo, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}
	project, slug, err := getRepo(repo)
	if err != nil {
		return commit, sdk.WithStack(err)
	}
	var stashURL, _ = url.Parse(b.consumer.URL)

	sc := Commit{}
	path := fmt.Sprintf("/projects/%s/repos/%s/commits/%s", project, slug, hash)
	if err := b.do(ctx, "GET", "core", path, nil, nil, &sc, nil); err != nil {
		return commit, sdk.WrapError(err, "Unable to get commit %s", path)
	}

	urlCommit := stashURL.String() + "/projects/" + project + "/repos/" + slug + "/commits/" + sc.Hash
	commit = sdk.VCSCommit{
		Hash:      sc.Hash,
		Timestamp: sc.Timestamp,
		Message:   sc.Message,
		Author: sdk.VCSAuthor{
			Name:        sc.Author.Name,
			Email:       sc.Author.Email,
			DisplayName: sc.Author.DisplayName,
		},
		URL: urlCommit,
	}
	if sc.Author.Slug != "" && sc.Author.Slug != "unknownSlug" {
		commit.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", b.consumer.URL, sc.Author.Slug)
	}
	return commit, nil
}

func (b *bitbucketClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	var commits []sdk.VCSCommit
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	var stashCommits []Commit

	var stashCommitsKey = cache.Key("vcs", "bitbucket", b.consumer.URL, repo, "compare/commits", "from@"+base, "to@"+head)

	find, err := b.consumer.cache.Get(stashCommitsKey, &stashCommits)
	if err != nil {
		log.Error(ctx, "cannot get from cache %s: %v", stashCommitsKey, err)
	}
	if !find {
		response := CommitsResponse{}
		path := fmt.Sprintf("/projects/%s/repos/%s/compare/commits", project, slug)
		params := url.Values{}
		if base != "" {
			params.Add("from", base)
		}
		if head != "" {
			params.Add("to", head)
		}

		for {
			if response.NextPageStart != 0 {
				params.Set("start", fmt.Sprintf("%d", response.NextPageStart))
			}

			if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
				if sdk.ErrorIs(err, sdk.ErrNotFound) {
					return nil, nil
				}
				return nil, sdk.WrapError(err, "Unable to get commits %s", path)
			}

			stashCommits = append(stashCommits, response.Values...)
			if response.IsLastPage {
				break
			}
		}
		//3 hours
		if err := b.consumer.cache.SetWithTTL(stashCommitsKey, stashCommits, 3*60*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", stashCommitsKey, err)
		}
	}

	urlCommit := b.consumer.URL + "/projects/" + project + "/repos/" + slug + "/compare/commits"
	for _, sc := range stashCommits {
		c := sdk.VCSCommit{
			Hash:      sc.Hash,
			Timestamp: sc.Timestamp,
			Message:   sc.Message,
			Author: sdk.VCSAuthor{
				Name:        sc.Author.Name,
				Email:       sc.Author.Email,
				DisplayName: sc.Author.DisplayName,
			},
			URL: urlCommit + sc.Hash,
		}

		if sc.Author.Slug != "" && sc.Author.Slug != "unknownSlug" {
			c.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", b.consumer.URL, sc.Author.Slug)
		}
		commits = append(commits, c)
	}
	return commits, nil
}
