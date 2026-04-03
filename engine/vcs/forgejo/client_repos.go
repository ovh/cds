package forgejo

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) Repos(ctx context.Context) ([]sdk.VCSRepo, error) {
	const maxResults = 200
	const pageSize = 50
	basePath := "/repos/search"

	var allRepos []*Repository
	for page := 1; ; page++ {
		var searchResult struct {
			Repos []*Repository `json:"data"`
			OK    bool          `json:"ok"`
		}
		apiPath := buildPaginatedPath(basePath, ListOptions{Page: page, PageSize: pageSize})
		if _, err := f.client.get(ctx, apiPath, &searchResult); err != nil {
			return nil, err
		}
		allRepos = append(allRepos, searchResult.Repos...)
		if len(searchResult.Repos) < pageSize || len(allRepos) >= maxResults {
			break
		}
	}
	if len(allRepos) > maxResults {
		allRepos = allRepos[:maxResults]
	}

	repositories := make([]sdk.VCSRepo, 0, len(allRepos))
	for _, r := range allRepos {
		repositories = append(repositories, f.ToVCSRepo(r))
	}
	return repositories, nil
}

func (f *forgejoClient) RepoByFullname(ctx context.Context, fullname string) (sdk.VCSRepo, error) {
	owner, repoName, err := getRepo(fullname)
	if err != nil {
		return sdk.VCSRepo{}, err
	}

	var repo Repository
	if _, err = f.client.get(ctx, fmt.Sprintf("/repos/%s/%s", owner, repoName), &repo); err != nil {
		return sdk.VCSRepo{}, err
	}
	return f.ToVCSRepo(&repo), nil
}

func (f *forgejoClient) UserHasWritePermission(ctx context.Context, repo string) (bool, error) {
	return false, sdk.WithStack(sdk.ErrNotImplemented)
}

func (f *forgejoClient) ToVCSRepo(repo *Repository) sdk.VCSRepo {
	return sdk.VCSRepo{
		URL:                  repo.HTMLURL,
		URLCommitFormat:      repo.HTMLURL + "/commit/%s",
		URLTagFormat:         repo.HTMLURL + "/commits/tag/%s",
		URLBranchFormat:      repo.HTMLURL + "/commits/branch/%s",
		URLPullRequestFormat: repo.HTMLURL + "/pulls/%d",
		Name:                 repo.Name,
		ID:                   fmt.Sprintf("%d", repo.ID),
		Fullname:             repo.FullName,
		HTTPCloneURL:         repo.CloneURL,
		SSHCloneURL:          repo.SSHURL,
		Slug:                 repo.Name,
	}
}
