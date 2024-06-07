package bitbucketserver

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (b *bitbucketClient) Branches(ctx context.Context, fullname string, filters sdk.VCSBranchesFilter) ([]sdk.VCSBranch, error) {
	_, end := telemetry.Span(ctx, "bitbucketserver.Branches", telemetry.Tag(telemetry.TagRepository, fullname))
	defer end()
	branches := []sdk.VCSBranch{}

	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return branches, sdk.ErrRepoNotFound
	}

	stashBranches := []Branch{}

	path := fmt.Sprintf("/projects/%s/repos/%s/branches", t[0], t[1])
	params := url.Values{}
	params.Set("orderBy", "MODIFICATION")

	nextPage := 0
	for {
		if ctx.Err() != nil {
			break
		}
		params.Set("limit", "100")
		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response BranchResponse
		if err := b.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get branches %s", path)
		}

		stashBranches = append(stashBranches, response.Values...)
		if response.IsLastPage || (filters.Limit > 0 && len(stashBranches) >= int(filters.Limit)) {
			break
		} else {
			nextPage += response.Size
		}
	}

	hasDefaultBranch := false
	for _, sb := range stashBranches {
		b := sdk.VCSBranch{
			ID:           sb.ID,
			DisplayID:    sb.DisplayID,
			LatestCommit: sb.LatestHash,
			Default:      sb.IsDefault,
		}
		if sb.IsDefault {
			hasDefaultBranch = true
		}
		branches = append(branches, b)
	}

	if !hasDefaultBranch {
		br, err := b.GetDefaultBranch(ctx, fullname)
		if err != nil {
			return nil, err
		}
		branches = append(branches, *br)
	}

	return branches, nil
}

func (b *bitbucketClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	if filters.Default {
		return b.GetDefaultBranch(ctx, fullname)
	}

	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, sdk.ErrRepoNotFound
	}

	branches := BranchResponse{}
	path := fmt.Sprintf("/projects/%s/repos/%s/branches?filterText=%s", t[0], t[1], url.QueryEscape(filters.BranchName))

	if err := b.do(ctx, "GET", "core", path, nil, nil, &branches); err != nil {
		return nil, sdk.WrapError(err, "Unable to get branch %s %s", filters.BranchName, path)
	}

	if len(branches.Values) == 0 {
		return nil, sdk.WithStack(sdk.ErrNoBranch)
	}

	for _, b := range branches.Values {
		if b.DisplayID == filters.BranchName {
			return &sdk.VCSBranch{
				ID:           b.ID,
				DisplayID:    b.DisplayID,
				LatestCommit: b.LatestHash,
				Default:      b.IsDefault,
			}, nil
		}
	}
	return nil, sdk.ErrNoBranch
}

func (b *bitbucketClient) GetDefaultBranch(ctx context.Context, fullname string) (*sdk.VCSBranch, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, sdk.ErrRepoNotFound
	}

	defaultBranch := Branch{}
	path := fmt.Sprintf("/projects/%s/repos/%s/branches/default", t[0], t[1])

	if err := b.do(ctx, "GET", "core", path, nil, nil, &defaultBranch); err != nil {
		return nil, sdk.WrapError(err, "Unable to get default branch %s", path)
	}

	return &sdk.VCSBranch{
		ID:           defaultBranch.ID,
		DisplayID:    defaultBranch.DisplayID,
		LatestCommit: defaultBranch.LatestHash,
		Default:      defaultBranch.IsDefault,
	}, nil

}
