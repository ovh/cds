package gitlab

import (
	"fmt"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
)

//Repos returns the list of accessible repositories
func (c *gitlabClient) Repos() ([]sdk.VCSRepo, error) {
	var repos []sdk.VCSRepo

	pp := 1000
	opts := &gitlab.ListProjectsOptions{}
	opts.PerPage = pp

	projects, resp, err := c.client.Projects.ListProjects(opts)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		r := sdk.VCSRepo{
			ID:           fmt.Sprintf("%d", p.ID),
			Name:         p.NameWithNamespace,
			Slug:         p.PathWithNamespace,
			Fullname:     p.PathWithNamespace,
			URL:          p.WebURL,
			HTTPCloneURL: p.HTTPURLToRepo,
			SSHCloneURL:  p.SSHURLToRepo,
		}
		repos = append(repos, r)
	}

	for resp.NextPage != 0 {
		opts.Page = resp.NextPage

		projects, resp, err = c.client.Projects.ListProjects(opts)
		if err != nil {
			return nil, err
		}

		for _, p := range projects {
			r := sdk.VCSRepo{
				ID:           fmt.Sprintf("%d", p.ID),
				Name:         p.NameWithNamespace,
				Slug:         p.PathWithNamespace,
				Fullname:     p.PathWithNamespace,
				URL:          p.WebURL,
				HTTPCloneURL: p.HTTPURLToRepo,
				SSHCloneURL:  p.SSHURLToRepo,
			}
			repos = append(repos, r)
		}
	}

	return repos, nil
}

//RepoByFullname returns the repo from its fullname
func (c *gitlabClient) RepoByFullname(fullname string) (sdk.VCSRepo, error) {
	repo := sdk.VCSRepo{}

	p, _, err := c.client.Projects.GetProject(fullname)
	if err != nil {
		return repo, err
	}
	repo.ID = fmt.Sprintf("%d", p.ID)
	repo.Name = p.NameWithNamespace
	repo.Slug = p.Name
	repo.Fullname = p.PathWithNamespace
	repo.URL = p.WebURL
	repo.HTTPCloneURL = p.HTTPURLToRepo
	repo.SSHCloneURL = p.SSHURLToRepo

	return repo, nil
}

func (c *gitlabClient) GrantReadPermission(repo string) error {
	return nil
}
