package bitbucketcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
)

// Tags returns list of tags for a repo
func (client *bitbucketcloudClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	var tags []Tag
	path := fmt.Sprintf("/2.0/repositories/%s/refs/tags", fullname)
	params := url.Values{}
	params.Set("pagelen", "100")
	nextPage := 1
	for {
		if nextPage != 1 {
			params.Set("page", fmt.Sprintf("%d", nextPage))
		}

		var response Tags
		if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
			return nil, sdk.WrapError(err, "Unable to get repos")
		}
		if cap(tags) == 0 {
			tags = make([]Tag, 0, response.Size)
		}

		tags = append(tags, response.Values...)

		if response.Next == "" {
			break
		} else {
			nextPage++
		}
	}

	responseTags := make([]sdk.VCSTag, 0, len(tags))
	for _, tag := range tags {
		email := strings.Trim(rawEmailCommitRegexp.FindString(tag.Target.Author.Raw), "<>")
		t := sdk.VCSTag{
			Hash:    tag.Target.Hash,
			Message: tag.Message,
			Sha:     tag.Target.Hash,
			Tagger: sdk.VCSAuthor{
				Avatar:      tag.Target.Author.User.Links.Avatar.Href,
				DisplayName: tag.Target.Author.User.DisplayName,
				Email:       email,
				Name:        tag.Target.Author.User.Nickname,
			},
		}
		responseTags = append(responseTags, t)
	}

	return responseTags, nil
}
