package bitbucketcloud

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (client *bitbucketcloudClient) ListForks(ctx context.Context, repo string) ([]sdk.VCSRepo, error) {
	var repos []Repository
	path := fmt.Sprintf("/repositories/%s/forks", repo)
	params := url.Values{}
	params.Set("pagelen", "100")
	nextPage := 1
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 1 {
			params.Set("page", fmt.Sprintf("%d", nextPage))
		}

		var response Repositories
		if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get repos")
		}
		if cap(repos) == 0 {
			repos = make([]Repository, 0, response.Size)
		}

		repos = append(repos, response.Values...)

		if response.Next == "" {
			break
		} else {
			nextPage++
		}
	}

	responseRepos := make([]sdk.VCSRepo, 0, len(repos))
	for _, repo := range repos {
		responseRepos = append(responseRepos, client.ToVCSRepo(repo))
	}

	return responseRepos, nil
}
