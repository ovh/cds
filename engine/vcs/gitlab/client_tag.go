package gitlab

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *gitlabClient) Tag(ctx context.Context, fullname string, tagName string) (sdk.VCSTag, error) {
	tag, _, err := c.client.Tags.GetTag(fullname, tagName)
	if err != nil {
		return sdk.VCSTag{}, err
	}
	return sdk.VCSTag{
		Tag:     tag.Name,
		Hash:    tag.Commit.ID,
		Message: tag.Message,
		Tagger: sdk.VCSAuthor{
			DisplayName: tag.Commit.AuthorName,
			Name:        tag.Commit.AuthorName,
			Email:       tag.Commit.AuthorEmail,
		},
	}, nil
}

// Tags retrieves the tags
func (c *gitlabClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	tags, _, err := c.client.Tags.ListTags(fullname, nil)
	if err != nil {
		return nil, err
	}

	respTags := make([]sdk.VCSTag, len(tags))
	for i, tag := range tags {
		respTags[i] = sdk.VCSTag{
			Tag:     tag.Name,
			Hash:    tag.Commit.ID,
			Message: tag.Message,
			Tagger: sdk.VCSAuthor{
				DisplayName: tag.Commit.AuthorName,
				Name:        tag.Commit.AuthorName,
				Email:       tag.Commit.AuthorEmail,
			},
		}
	}

	return respTags, nil
}
