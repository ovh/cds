package bitbucketserver

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/rockbears/log"

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
		if err := b.do(ctx, "GET", "core", path, params, nil, &response, Options{DisableCache: filters.NoCache}); err != nil {
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
		br, err := b.GetDefaultBranch(ctx, fullname, Options{DisableCache: filters.NoCache})
		if err != nil {
			return nil, err
		}
		branches = append(branches, *br)
	}

	return branches, nil
}

func (b *bitbucketClient) Branch(ctx context.Context, fullname string, filters sdk.VCSBranchFilters) (*sdk.VCSBranch, error) {
	if filters.Default {
		return b.GetDefaultBranch(ctx, fullname, Options{DisableCache: filters.NoCache})
	}

	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, sdk.ErrRepoNotFound
	}

	branches := BranchResponse{}
	path := fmt.Sprintf("/projects/%s/repos/%s/branches?orderBy=MODIFICATION&filterText=%s", t[0], t[1], url.QueryEscape(filters.BranchName))

	if err := b.do(ctx, "GET", "core", path, nil, nil, &branches, Options{DisableCache: filters.NoCache}); err != nil {
		return nil, sdk.WrapError(err, "Unable to get branch %s %s", filters.BranchName, path)
	}

	displayIDs := make([]string, 0, len(branches.Values))
	for _, sb := range branches.Values {
		if sb.DisplayID == filters.BranchName {
			return &sdk.VCSBranch{
				ID:           sb.ID,
				DisplayID:    sb.DisplayID,
				LatestCommit: sb.LatestHash,
				Default:      sb.IsDefault,
			}, nil
		}
		displayIDs = append(displayIDs, sb.DisplayID)
	}

	// filterText is a tokenized substring filter and the response is paginated: the branches
	// listing can omit an existing branch, the default one included. Before concluding that the
	// branch does not exist, ask for the default branch explicitly. Same workaround as in Branches().
	var defaultBranchName string
	defaultBranch, err := b.GetDefaultBranch(ctx, fullname, Options{DisableCache: filters.NoCache})
	if err != nil {
		// this fallback must not hide the diagnosis of the initial lookup
		log.Error(ctx, "bitbucketClient.Branch> unable to get default branch of %s: %v", fullname, err)
		defaultBranchName = "unknown"
	} else {
		defaultBranchName = defaultBranch.DisplayID
		// an empty latest commit means the configured default branch points to a ref that does not
		// exist anymore: do not confirm a branch on the repository configuration only.
		if defaultBranch.DisplayID == filters.BranchName && defaultBranch.LatestCommit != "" {
			log.Warn(ctx, "bitbucketClient.Branch> branch %s of %s is missing from the branches listing but is the default branch: %s", filters.BranchName, fullname, path)
			return defaultBranch, nil
		}
	}

	if len(branches.Values) == 0 {
		return nil, sdk.NewErrorFrom(sdk.WithStack(sdk.ErrNoBranch),
			"bitbucket returned no branch for filterText=%q on %s, default branch is %q",
			filters.BranchName, path, defaultBranchName)
	}
	return nil, sdk.NewErrorFrom(sdk.WithStack(sdk.ErrNoBranch),
		"bitbucket returned %d branch(es) for filterText=%q (lastPage=%t) on %s, none matching exactly and default branch is %q: %s",
		len(branches.Values), filters.BranchName, branches.IsLastPage, path, defaultBranchName, strings.Join(displayIDs, ", "))
}

func (b *bitbucketClient) GetDefaultBranch(ctx context.Context, fullname string, opts Options) (*sdk.VCSBranch, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, sdk.ErrRepoNotFound
	}

	defaultBranch := Branch{}
	path := fmt.Sprintf("/projects/%s/repos/%s/branches/default", t[0], t[1])

	if err := b.do(ctx, "GET", "core", path, nil, nil, &defaultBranch, opts); err != nil {
		return nil, sdk.WrapError(err, "Unable to get default branch %s", path)
	}

	return &sdk.VCSBranch{
		ID:           defaultBranch.ID,
		DisplayID:    defaultBranch.DisplayID,
		LatestCommit: defaultBranch.LatestHash,
		// this comes from /branches/default, it is the default branch whatever isDefault holds
		Default: true,
	}, nil

}
