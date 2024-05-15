package bitbucketserver

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (b *bitbucketClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	ctx, end := telemetry.Span(ctx, "bitbucketserver.ListForks", telemetry.Tag(telemetry.TagRepository, repo))
	defer end()
	bbRepos := []Repo{}
	project, slug, err := getRepo(repo)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	path := fmt.Sprintf("/projects/%s/repos/%s/forks", project, slug)
	params := url.Values{}
	nextPage := 0
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response Response
		if err := b.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get repos")
		}

		bbRepos = append(bbRepos, response.Values...)

		if response.IsLastPage {
			break
		} else {
			nextPage = response.NextPageStart
		}
	}

	repos := []sdk.VCSRepo{}
	for _, r := range bbRepos {
		repos = append(repos, b.ToVCSRepo(r))
	}
	return repos, nil
}
