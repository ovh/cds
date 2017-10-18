package bitbucket

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (b *bitbucketClient) Commits(repo, branch, since, until string) ([]sdk.VCSCommit, error) {
	commits := []sdk.VCSCommit{}
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return commits, fmt.Errorf("fullname %s must be <project>/<slug>", repo)
	}

	project := t[0]
	slug := t[1]

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

			if err := b.do("GET", "core", path, params, nil, &response); err != nil {
				return nil, err
			}

			stashCommits = append(stashCommits, response.Values...)
			if response.IsLastPage {
				break
			}
		}
		b.consumer.cache.SetWithTTL(stashCommitsKey, stashCommits, 3*60*60) //3 hours
	}

	urlCommit := b.consumer.URL + "/projects/" + t[0] + "/repos/" + t[1] + "/commits/"
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
		stashUser := b.findUser(sc.Author.Email)
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

func (b *bitbucketClient) findUser(email string) *User {
	var stashUser = &User{}
	var stashUserKey = cache.Key("reposmanager", "stash", b.consumer.URL, email)
	if !b.consumer.cache.Get(stashUserKey, &stashUser) && email != "" {
		newStashUser, err := b.findByEmail(email)
		if err != nil {
			log.Warning("Unable to get stash user %s : %s", email, err)
			return nil
		}
		b.consumer.cache.Set(stashUserKey, newStashUser)
		stashUser = newStashUser
	}
	return stashUser
}

func (b *bitbucketClient) Commit(repo, hash string) (sdk.VCSCommit, error) {
	commit := sdk.VCSCommit{}
	t := strings.Split(repo, "/")
	if len(t) != 2 {
		return commit, fmt.Errorf("fullname %s must be <project>/<slug>", repo)
	}

	var stashURL, _ = url.Parse(b.consumer.URL)

	sc := Commit{}
	path := fmt.Sprintf("/projects/%s/repos/%s/commits/%s", t[0], t[1], hash)
	if err := b.do("GET", "core", path, nil, nil, &sc); err != nil {
		return commit, err
	}

	urlCommit := stashURL.String() + "/projects/" + t[0] + "/repos/" + t[1] + "/commits/" + sc.Hash
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

	stashUser := b.findUser(sc.Author.Email)
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
