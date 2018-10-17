package bitbucket

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (b *bitbucketClient) Commits(ctx context.Context, repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	commits := []sdk.VCSCommit{}
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WrapError(err, "commits>")
	}

	stashCommits := []Commit{}

	var stashCommitsKey = cache.Key("vcs", "bitbucket", b.consumer.URL, repo, "commits", "since@"+since, "until@"+until)

	if !b.consumer.cache.Get(stashCommitsKey, &stashCommits) {
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
			if response.NextPageStart != 0 {
				params.Set("start", fmt.Sprintf("%d", response.NextPageStart))
			}

			if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
				if err == sdk.ErrNotFound {
					return nil, nil
				}
				return nil, sdk.WrapError(err, "Unable to get commits %s", path)
			}

			stashCommits = append(stashCommits, response.Values...)
			if response.IsLastPage {
				break
			}
		}
		b.consumer.cache.SetWithTTL(stashCommitsKey, stashCommits, 3*60*60) //3 hours
	}

	urlCommit := b.consumer.URL + "/projects/" + project + "/repos/" + slug + "/commits/"
	for _, sc := range stashCommits {
		c := sdk.VCSCommit{
			Hash:      sc.Hash,
			Timestamp: sc.Timestamp,
			Message:   sc.Message,
			Author: sdk.VCSAuthor{
				Name:  sc.Author.Name,
				Email: sc.Author.Email,
			},
			URL: urlCommit + sc.Hash,
		}
		stashUser := b.findUser(ctx, sc.Author.Email)
		if stashUser == nil {
			newStashUserUnknown := newUnknownStashUser(*sc.Author)
			var stashUserKey = cache.Key("vcs", "bitbucket", b.consumer.URL, sc.Author.Email)
			b.consumer.cache.SetWithTTL(stashUserKey, newStashUserUnknown, 86400) // 1 day
			stashUser = newUnknownStashUser(*sc.Author)
		}

		c.Author.DisplayName = stashUser.DisplayName
		if stashUser.Slug != "" && stashUser.Slug != "unknownSlug" {
			c.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", b.consumer.URL, stashUser.Slug)
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func (b *bitbucketClient) findUser(ctx context.Context, email string) *User {
	var stashUser = &User{}
	var stashUserKey = cache.Key("reposmanager", "stash", b.consumer.URL, email)
	if !b.consumer.cache.Get(stashUserKey, &stashUser) && email != "" {
		newStashUser, err := b.findByEmail(ctx, email)
		if err != nil {
			if !strings.Contains(err.Error(), "User not found") {
				log.Warning("Unable to get stash user %s : %s", email, err)
			}
			return nil
		}
		b.consumer.cache.Set(stashUserKey, newStashUser)
		stashUser = newStashUser
	}
	return stashUser
}

func (b *bitbucketClient) Commit(ctx context.Context, repo, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}
	project, slug, err := getRepo(repo)
	if err != nil {
		return commit, sdk.WrapError(err, "commit>")
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
			Name:  sc.Author.Name,
			Email: sc.Author.Email,
		},
		URL: urlCommit,
	}

	stashUser := b.findUser(ctx, sc.Author.Email)
	if stashUser == nil {
		newStashUserUnknown := newUnknownStashUser(*sc.Author)
		var stashUserKey = cache.Key("vcs", "bitbucket", b.consumer.URL, sc.Author.Email)
		b.consumer.cache.SetWithTTL(stashUserKey, newStashUserUnknown, 86400) // 1 day
		stashUser = newUnknownStashUser(*sc.Author)
	}
	commit.Author.DisplayName = stashUser.DisplayName
	if stashUser.Slug != "" && stashUser.Slug != "unknownSlug" {
		commit.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", b.consumer.URL, stashUser.Slug)
	}
	return commit, nil
}

func newUnknownStashUser(author Author) *User {
	return &User{
		Username:     author.Name,
		EmailAddress: author.Email,
		DisplayName:  author.Name,
		Slug:         "unknownSlug",
	}
}

func (b *bitbucketClient) CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]sdk.VCSCommit, error) {
	var commits []sdk.VCSCommit
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WrapError(err, "CommitsBetweenRefs>")
	}

	var stashCommits []Commit

	var stashCommitsKey = cache.Key("vcs", "bitbucket", b.consumer.URL, repo, "compare/commits", "from@"+base, "to@"+head)

	if !b.consumer.cache.Get(stashCommitsKey, &stashCommits) {
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
				if err == sdk.ErrNotFound {
					return nil, nil
				}
				return nil, sdk.WrapError(err, "Unable to get commits %s", path)
			}

			stashCommits = append(stashCommits, response.Values...)
			if response.IsLastPage {
				break
			}
		}
		b.consumer.cache.SetWithTTL(stashCommitsKey, stashCommits, 3*60*60) //3 hours
	}

	urlCommit := b.consumer.URL + "/projects/" + project + "/repos/" + slug + "/compare/commits"
	for _, sc := range stashCommits {
		c := sdk.VCSCommit{
			Hash:      sc.Hash,
			Timestamp: sc.Timestamp,
			Message:   sc.Message,
			Author: sdk.VCSAuthor{
				Name:  sc.Author.Name,
				Email: sc.Author.Email,
			},
			URL: urlCommit + sc.Hash,
		}
		stashUser := b.findUser(ctx, sc.Author.Email)
		if stashUser == nil {
			newStashUserUnknown := newUnknownStashUser(*sc.Author)
			var stashUserKey = cache.Key("vcs", "bitbucket", b.consumer.URL, sc.Author.Email)
			b.consumer.cache.SetWithTTL(stashUserKey, newStashUserUnknown, 86400) // 1 day
			stashUser = newUnknownStashUser(*sc.Author)
		}

		c.Author.DisplayName = stashUser.DisplayName
		if stashUser.Slug != "" && stashUser.Slug != "unknownSlug" {
			c.Author.Avatar = fmt.Sprintf("%s/users/%s/avatar.png", b.consumer.URL, stashUser.Slug)
		}
		commits = append(commits, c)
	}
	return commits, nil
}
