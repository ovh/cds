package gitea

import (
	"context"
	"fmt"

	gg "code.gitea.io/sdk/gitea"

	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) Repos(_ context.Context) ([]sdk.VCSRepo, error) {
	repos, _, err := g.client.SearchRepos(gg.SearchRepoOptions{})
	if err != nil {
		return nil, err
	}
	repositories := make([]sdk.VCSRepo, 0, len(repos))
	for _, r := range repos {
		repositories = append(repositories, g.ToVCSRepo(r))
	}
	return repositories, nil
}

func (g *giteaClient) RepoByFullname(_ context.Context, fullname string) (sdk.VCSRepo, error) {
	owner, repoName, err := getRepo(fullname)
	if err != nil {
		return sdk.VCSRepo{}, err
	}
	repo, _, err := g.client.GetRepo(owner, repoName)
	if err != nil {
		return sdk.VCSRepo{}, err
	}
	return g.ToVCSRepo(repo), nil
}

func (g *giteaClient) UserHasWritePermission(ctx context.Context, repo string) (bool, error) {
	return false, sdk.WithStack(sdk.ErrNotImplemented)
}

func (g *giteaClient) ToVCSRepo(repo *gg.Repository) sdk.VCSRepo {
	return sdk.VCSRepo{
		URL:          repo.HTMLURL,
		Name:         repo.Name,
		ID:           fmt.Sprintf("%d", repo.ID),
		Fullname:     repo.FullName,
		HTTPCloneURL: repo.CloneURL,
		SSHCloneURL:  repo.SSHURL,
		Slug:         repo.Name,
	}
}
