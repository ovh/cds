package forgejo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

// Tags retrieve tags
func (c *forgejoClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	const maxResults = 200
	const pageSize = 50
	basePath := fmt.Sprintf("/repos/%s/%s/tags", owner, repo)

	var allTags []*Tag
	for page := 1; ; page++ {
		var pageTags []*Tag
		apiPath := buildPaginatedPath(basePath, ListOptions{Page: page, PageSize: pageSize})
		if _, err := c.client.get(ctx, apiPath, &pageTags); err != nil {
			return nil, err
		}
		allTags = append(allTags, pageTags...)
		if len(pageTags) < pageSize || len(allTags) >= maxResults {
			break
		}
	}
	if len(allTags) > maxResults {
		allTags = allTags[:maxResults]
	}

	ret := make([]sdk.VCSTag, 0, len(allTags))
	for _, tag := range allTags {
		vcsTag := sdk.VCSTag{
			Tag: tag.Name,
			Sha: tag.ID,
		}
		if tag.Commit != nil {
			vcsTag.Hash = tag.Commit.SHA
		}
		ret = append(ret, vcsTag)
	}

	return ret, nil
}

func (c *forgejoClient) Tag(ctx context.Context, fullname string, tagName string) (sdk.VCSTag, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return sdk.VCSTag{}, err
	}

	var tag Tag
	apiPath := fmt.Sprintf("/repos/%s/%s/tags/%s", owner, repo, url.PathEscape(tagName))
	if _, err := c.client.get(ctx, apiPath, &tag); err != nil {
		return sdk.VCSTag{}, err
	}
	return sdk.VCSTag{
		Tag: tag.Name,
		Sha: tag.ID,
		Hash: func() string {
			if tag.Commit != nil {
				return tag.Commit.SHA
			}
			return ""
		}(),
	}, nil
}
