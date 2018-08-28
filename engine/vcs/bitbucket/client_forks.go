package bitbucket

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	bbRepos := []Repo{}
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WrapError(err, "vcs> bitbucket> ListForks>")
	}
	path := fmt.Sprintf("/projects/%s/repos/%s/forks", project, slug)
	params := url.Values{}
	nextPage := 0
	for {
		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response Response
		if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
			return nil, sdk.WrapError(err, "vcs> bitbucket> ListForks> Unable to get repos")
		}

		bbRepos = append(bbRepos, response.Values...)

		if response.IsLastPage {
			break
		} else {
			nextPage = response.NextPageStart
		}
	}

	repos := []sdk.VCSRepo{}
	for _, r := range bbRepos {
		var repoURL string
		if r.Link != nil {
			repoURL = r.Link.URL
		}

		var sshURL, httpURL string
		if r.Links != nil && r.Links.Clone != nil {
			for _, c := range r.Links.Clone {
				if c.Name == "http" {
					httpURL = c.URL
				}
				if c.Name == "ssh" {
					sshURL = c.URL
				}
			}
		}

		repo := sdk.VCSRepo{
			Name:         r.Name,
			Slug:         r.Slug,
			Fullname:     fmt.Sprintf("%s/%s", r.Project.Key, r.Slug),
			URL:          fmt.Sprintf("%s%s", b.consumer.URL, repoURL),
			HTTPCloneURL: httpURL,
			SSHCloneURL:  sshURL,
		}
		repos = append(repos, repo)
	}
	return repos, nil
}
