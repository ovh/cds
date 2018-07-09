package bitbucket

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) Repos() ([]sdk.VCSRepo, error) {
	bbRepos := []Repo{}

	path := "/repos"
	params := url.Values{}
	nextPage := 0
	for {
		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response Response
		if err := b.do("GET", "core", path, params, nil, &response, nil); err != nil {
			return nil, sdk.WrapError(err, "vcs> bitbucket> Repos> Unable to get repos")
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

func (b *bitbucketClient) RepoByFullname(fullname string) (sdk.VCSRepo, error) {
	t := strings.SplitN(fullname, "/", 2)
	r := Repo{}
	path := fmt.Sprintf("/projects/%s/repos/%s", t[0], t[1])

	var repo sdk.VCSRepo
	if err := b.do("GET", "core", path, nil, nil, &r, nil); err != nil {
		return repo, sdk.WrapError(err, "vcs> bitbucket> RepoByFullname> Unable to get repo")
	}

	var sshURL, httpURL, repoURL string
	if r.Links != nil {
		if r.Links.Clone != nil {
			for _, c := range r.Links.Clone {
				if c.Name == "http" {
					httpURL = c.URL
				}
				if c.Name == "ssh" {
					sshURL = c.URL
				}
			}
		}

		if r.Links.Self != nil {
			for _, c := range r.Links.Self {
				repoURL = c.URL
				break
			}
		}
	}

	repo = sdk.VCSRepo{
		Name:         r.Name,
		Slug:         r.Slug,
		Fullname:     fmt.Sprintf("%s/%s", r.Project.Key, r.Slug),
		URL:          repoURL,
		HTTPCloneURL: httpURL,
		SSHCloneURL:  sshURL,
	}

	return repo, nil
}

func (b *bitbucketClient) GrantReadPermission(repo string) error {
	if b.username == "" {
		return nil
	}

	project, slug, err := getRepo(repo)
	if err != nil {
		return sdk.WrapError(err, "vcs> bitbucket> GrantReadPermission>")
	}
	path := fmt.Sprintf("/projects/%s/repos/%s/permissions/users", project, slug)
	params := url.Values{}
	params.Add("name", b.username)
	params.Add("permission", "REPO_WRITE")

	return b.do("PUT", "core", path, params, nil, nil, nil)
}
