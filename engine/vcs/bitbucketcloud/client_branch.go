package bitbucketcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Branches returns list of branches for a repo
func (client *bitbucketcloudClient) Branches(ctx context.Context, fullname string) ([]sdk.VCSBranch, error) {
	repo, err := client.repoByFullname(ctx, fullname)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get repo by fullname")
	}

	var branches []Branch
	path := fmt.Sprintf("/repositories/%s/refs/branches", fullname)
	params := url.Values{}
	params.Set("pagelen", "100")
	params.Set("sort", "-target.date")
	nextPage := 1
	for {
		if nextPage != 1 {
			params.Set("page", fmt.Sprintf("%d", nextPage))
		}

		var response Branches
		if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get branches")
		}
		if cap(branches) == 0 {
			branches = make([]Branch, 0, response.Size)
		}

		branches = append(branches, response.Values...)

		if response.Next == "" {
			break
		} else {
			nextPage++
		}
	}

	branchesResult := make([]sdk.VCSBranch, 0, len(branches))
	for _, b := range branches {
		branch := sdk.VCSBranch{
			DisplayID:    b.Name,
			ID:           b.Name,
			LatestCommit: b.Target.Hash,
			Default:      b.Name == repo.Mainbranch.Name,
		}
		for _, p := range b.Target.Parents {
			branch.Parents = append(branch.Parents, p.Hash)
		}
		branchesResult = append(branchesResult, branch)
	}

	return branchesResult, nil
}

// Branch returns only detail of a branch
func (client *bitbucketcloudClient) Branch(ctx context.Context, fullname, theBranch string) (*sdk.VCSBranch, error) {
	repo, err := client.repoByFullname(ctx, fullname)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("/repositories/%s/refs/branches/%s", fullname, theBranch)
	status, body, _, err := client.get(url)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
	}

	var branch Branch
	if err := json.Unmarshal(body, &branch); err != nil {
		log.Warning(ctx, "bitbucketcloudClient.Branch> Unable to parse github branch: %s", err)
		return nil, err
	}

	if branch.Name == "" {
		return nil, fmt.Errorf("bitbucketcloudClient.Branch > Cannot find branch %s", theBranch)
	}

	branchResult := &sdk.VCSBranch{
		DisplayID:    branch.Name,
		ID:           branch.Name,
		LatestCommit: branch.Target.Hash,
		Default:      branch.Name == repo.Mainbranch.Name,
	}

	if branch.Target.Hash != "" {
		for _, p := range branch.Target.Parents {
			branchResult.Parents = append(branchResult.Parents, p.Hash)
		}
	}

	return branchResult, nil
}
