package gitlab

import (
	"context"
	"fmt"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
)

// Repos returns the list of accessible repositories
func (c *gitlabClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	var repos []sdk.VCSRepo

	pp := 1000
	opts := &gitlab.ListProjectsOptions{
		Membership: gitlab.Bool(true),
	}
	opts.PerPage = pp

	projects, resp, err := c.client.Projects.ListProjects(opts)
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

		projects, resp, err = c.client.Projects.ListProjects(opts)
		if err != nil {
			return nil, err
		}

		for _, p := range projects {
			repos = append(repos, c.ToVCSRepo(p))
		}
	}

	return repos, nil
}

// RepoByFullname returns the repo from its fullname
func (c *gitlabClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	repo := sdk.VCSRepo{}

	p, _, err := c.client.Projects.GetProject(fullname, nil)
	if err != nil {
		return repo, err
	}

	return c.ToVCSRepo(p), nil
}

func (c *gitlabClient) ToVCSRepo(p *gitlab.Project) sdk.VCSRepo {
	return sdk.VCSRepo{
		ID:              fmt.Sprintf("%d", p.ID),
		Name:            p.NameWithNamespace,
		Slug:            p.PathWithNamespace,
		Fullname:        p.PathWithNamespace,
		URL:             p.WebURL,
		URLCommitFormat: p.WebURL + "/-/commit/%s",
		URLTagFormat:    p.WebURL + "/-/tree/%s?ref_type=tags",
		URLBranchFormat: p.WebURL + "/-/tree/%s?ref_type=heads",
		HTTPCloneURL:    p.HTTPURLToRepo,
		SSHCloneURL:     p.SSHURLToRepo,
	}
}
