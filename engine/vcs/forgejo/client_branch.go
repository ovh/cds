package forgejo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) Branches(ctx context.Context, fullname string, filters sdk.VCSBranchesFilter) ([]sdk.VCSBranch, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if _, err = f.client.get(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo), &repository); err != nil {
		return nil, err
	}

	maxResults := 200
	if filters.Limit > 0 && int(filters.Limit) < maxResults {
		maxResults = int(filters.Limit)
	}
	const pageSize = 50
	basePath := fmt.Sprintf("/repos/%s/%s/branches", owner, repo)

	var allBranches []*Branch
	for page := 1; ; page++ {
		var pageBranches []*Branch
		apiPath := buildPaginatedPath(basePath, ListOptions{Page: page, PageSize: pageSize})
		if _, err = f.client.get(ctx, apiPath, &pageBranches); err != nil {
			return nil, err
		}
		allBranches = append(allBranches, pageBranches...)
		if len(pageBranches) < pageSize || len(allBranches) >= maxResults {
			break
		}
	}
	if len(allBranches) > maxResults {
		allBranches = allBranches[:maxResults]
	}

	vcsBranches := make([]sdk.VCSBranch, 0, len(allBranches))
	for _, b := range allBranches {
		vcsB := sdk.VCSBranch{
			ID:        sdk.GitRefBranchPrefix + b.Name,
			DisplayID: b.Name,
			Default:   b.Name == repository.DefaultBranch,
		}
		if b.Commit != nil {
			vcsB.LatestCommit = b.Commit.ID
		}
		vcsBranches = append(vcsBranches, vcsB)
	}
	return vcsBranches, nil
}

func (f *forgejoClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	if filters.Default {
		return f.GetDefaultBranch(ctx, fullname)
	}

	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if _, err = f.client.get(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo), &repository); err != nil {
		return nil, err
	}

	var b Branch
	if _, err = f.client.get(ctx, fmt.Sprintf("/repos/%s/%s/branches/%s", owner, repo, url.PathEscape(filters.BranchName)), &b); err != nil {
		return nil, err
	}
	vcsBranch := sdk.VCSBranch{
		ID:        sdk.GitRefBranchPrefix + b.Name,
		DisplayID: b.Name,
		Default:   filters.BranchName == repository.DefaultBranch,
	}
	if b.Commit != nil {
		vcsBranch.LatestCommit = b.Commit.ID
	}
	return &vcsBranch, nil
}

func (f *forgejoClient) GetDefaultBranch(ctx context.Context, fullname string) (*sdk.VCSBranch, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	var repository Repository
	if _, err = f.client.get(ctx, fmt.Sprintf("/repos/%s/%s", owner, repo), &repository); err != nil {
		return nil, err
	}

	var b Branch
	if _, err = f.client.get(ctx, fmt.Sprintf("/repos/%s/%s/branches/%s", owner, repo, url.PathEscape(repository.DefaultBranch)), &b); err != nil {
		return nil, err
	}
	vcsBranch := sdk.VCSBranch{
		ID:        sdk.GitRefBranchPrefix + b.Name,
		DisplayID: b.Name,
		Default:   true,
	}
	if b.Commit != nil {
		vcsBranch.LatestCommit = b.Commit.ID
	}
	return &vcsBranch, nil
}
