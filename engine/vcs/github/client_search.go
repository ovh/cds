package github

import (
	"context"
	"fmt"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"net/url"
	"strings"
)

func (g *githubClient) SearchPullRequest(ctx context.Context, repoFullName, commit, state string) (*sdk.VCSPullRequest, error) {
	query := url.QueryEscape(fmt.Sprintf("commit:%s repo:%s", commit, repoFullName))
	var nextPage = fmt.Sprintf("/search/issues?q=%s", query)
	for nextPage != "" {
		if ctx.Err() != nil {
			break
		}
		status, body, headers, err := g.get(ctx, nextPage)
		if err != nil {
			log.Warn(ctx, "githubClient.SearchPullRequest> Error %s", err)
			return nil, err
		}
		if status >= 400 {
			return nil, sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		}
		var search SearchResult
		if err := sdk.JSONUnmarshal(body, &search); err != nil {
			log.Warn(ctx, "githubClient.SearchPullRequest> Unable to parse github issues: %s", err)
			return nil, err
		}

		for _, i := range search.Items {
			if strings.HasSuffix(i.RepositoryURL, repoFullName) && i.PullRequest != nil {
				prURL := i.PullRequest.URL
				prURLSplit := strings.Split(prURL, "/")
				prID := prURLSplit[len(prURLSplit)-1]
				pr, err := g.PullRequest(ctx, repoFullName, prID)
				if err != nil {
					return nil, err
				}
				if pr.State == state {
					return &pr, nil
				}
			}
		}
		nextPage = getNextPage(headers)
	}
	return nil, sdk.WithStack(sdk.ErrNotFound)
}
