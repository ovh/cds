package gerrit

import (
	"context"
	"fmt"
	url2 "net/url"

	gg "github.com/andygrunwald/go-gerrit"

	"github.com/ovh/cds/sdk"
)

// Repos returns the list of accessible repositories
func (c *gerritClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	repos, _, err := c.client.Projects.ListProjects(nil)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list repositories")
	}

	if repos == nil {
		return nil, nil
	}

	vcsRepos := make([]sdk.VCSRepo, 0, len(*repos))
	for k, v := range *repos {
		vcsRepos = append(vcsRepos, c.ToVCSRepo(k, v))
	}
	return vcsRepos, nil
}

// RepoByFullname returns the repo from its fullname
func (c *gerritClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	repo, _, err := c.client.Projects.GetProject(fullname)
	if err != nil || repo == nil {
		return sdk.VCSRepo{}, sdk.WrapError(err, "unable to get repository")
	}
	return c.ToVCSRepo(fullname, *repo), nil
}

func (c *gerritClient) ToVCSRepo(name string, repo gg.ProjectInfo) sdk.VCSRepo {
	url, _ := url2.Parse(c.url)
	return sdk.VCSRepo{
		ID:           repo.ID,
		Name:         name,
		URL:          fmt.Sprintf("%s/%s", c.url, name),
		SSHCloneURL:  fmt.Sprintf("ssh://%s@%s:%d/%s", c.sshUsername, url.Hostname(), c.sshPort, name),
		HTTPCloneURL: fmt.Sprintf("%s/%s", c.url, name),
		Slug:         name,
		Fullname:     name,
	}
}
