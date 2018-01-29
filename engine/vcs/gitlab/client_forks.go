package gitlab

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	gitlab "github.com/xanzy/go-gitlab"
)

func (c *gitlabClient) ListForks(repo string) ([]sdk.VCSRepo, error) {
	var repos []sdk.VCSRepo

	pp := 1000
	opts := &gitlab.ListProjectsOptions{}
	opts.PerPage = pp

	projects, resp, err := c.client.Projects.ListProjectForks(repo, opts)
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

		projects, resp, err = c.client.Projects.ListProjectForks(repo, opts)
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
