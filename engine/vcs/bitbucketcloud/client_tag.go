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
	path := fmt.Sprintf("/repositories/%s/refs/tags", fullname)
	params := url.Values{}
	params.Set("pagelen", "100")

	var response Tags
	if err := client.do(ctx, "GET", "core", path, params, nil, &response); err != nil {
		return nil, sdk.WrapError(err, "Unable to get tags")
	}
	if cap(tags) == 0 {
		tags = make([]Tag, 0, response.Size)
	}

	tags = append(tags, response.Values...)

	responseTags := make([]sdk.VCSTag, 0, len(tags))
	for _, tag := range tags {
		email := strings.Trim(rawEmailCommitRegexp.FindString(tag.Target.Author.Raw), "<>")
		t := sdk.VCSTag{
			Tag:     tag.Name,
			Hash:    tag.Target.Hash,
			Message: tag.Message,
			Sha:     tag.Target.Hash,
			Tagger: sdk.VCSAuthor{
				Avatar:      tag.Target.Author.User.Links.Avatar.Href,
				DisplayName: tag.Target.Author.User.DisplayName,
				Email:       email,
				Name:        tag.Target.Author.User.Nickname,
				ID:          tag.Target.Author.User.UUID,
			},
		}
		responseTags = append(responseTags, t)
	}

	return responseTags, nil
}
