package bitbucketserver

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

// Tags retrieve tags
func (b *bitbucketClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	ctx, end := telemetry.Span(ctx, "bitbucketserver.Tags", telemetry.Tag(telemetry.TagRepository, fullname))
	defer end()

	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, sdk.ErrRepoNotFound
	}

	bitbucketTags := []Tag{}
	path := fmt.Sprintf("/projects/%s/repos/%s/tags", t[0], t[1])
	params := url.Values{}

	nextPage := 0
	for {
		if ctx.Err() != nil {
			break
		}

		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response TagResponse
		if err := b.do(ctx, "GET", "core", path, params, nil, &response, nil); err != nil {
			return nil, sdk.WrapError(err, "Unable to get tags %s", path)
		}

		bitbucketTags = append(bitbucketTags, response.Values...)
		if response.IsLastPage {
			break
		} else {
			nextPage += response.Size
		}
	}

	tags := make([]sdk.VCSTag, len(bitbucketTags))
	for i, tag := range bitbucketTags {
		tags[i] = sdk.VCSTag{
			Tag:  strings.Replace(tag.ID, "refs/tags/", "", 1),
			Hash: tag.LatestCommit,
			Sha:  tag.Hash,
		}
	}

	return tags, nil
}
