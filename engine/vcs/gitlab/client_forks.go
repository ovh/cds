package gitlab

import (
	"context"

	gitlab "github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
)

func (c *gitlabClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	var repos []sdk.VCSRepo

	pp := 1000
	opts := &gitlab.ListProjectsOptions{}
	opts.PerPage = pp

	projects, resp, err := c.client.Projects.ListProjectForks(repo, opts)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		repos = append(repos, c.ToVCSRepo(p))
	}

	for resp.NextPage != 0 {
		if ctx.Err() != nil {
			break
		}

		opts.Page = resp.NextPage

		projects, resp, err = c.client.Projects.ListProjectForks(repo, opts)
		if err != nil {
			return nil, err
		}

		for _, p := range projects {
			repos = append(repos, c.ToVCSRepo(p))
		}
	}

	return repos, nil
}
