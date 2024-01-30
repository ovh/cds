package gitea

import (
	"context"

	"code.gitea.io/sdk/gitea"

	"github.com/ovh/cds/sdk"
)

func (g *giteaClient) Branches(ctx context.Context, fullname string, filters sdk.VCSBranchesFilter) ([]sdk.VCSBranch, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	repository, _, err := g.client.GetRepo(owner, repo)
	if err != nil {
		return nil, err
	}

	branches, _, err := g.client.ListRepoBranches(owner, repo, gitea.ListRepoBranchesOptions{gitea.ListOptions{
		Page:     -1,
		PageSize: 1000,
	}})
	if err != nil {
		return nil, err
	}
	vcsBranches := make([]sdk.VCSBranch, 0, len(branches))
	for _, b := range branches {
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

func (g *giteaClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	if filters.Default {
		return g.GetDefaultBranch(ctx, fullname)
	}

	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	repository, _, err := g.client.GetRepo(owner, repo)
	if err != nil {
		return nil, err
	}

	b, _, err := g.client.GetRepoBranch(owner, repo, filters.BranchName)
	if err != nil {
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

func (g *giteaClient) GetDefaultBranch(ctx context.Context, fullname string) (*sdk.VCSBranch, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	repository, _, err := g.client.GetRepo(owner, repo)
	if err != nil {
		return nil, err
	}
	b, _, err := g.client.GetRepoBranch(owner, repo, repository.DefaultBranch)
	if err != nil {
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
