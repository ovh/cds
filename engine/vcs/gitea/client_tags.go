package gitea

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/ovh/cds/sdk"
)

// Tags retrieve tags
func (c *giteaClient) Tags(ctx context.Context, fullname string) ([]sdk.VCSTag, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, fmt.Errorf("giteaCliet.Tags> invalid fullname %s", fullname)
	}
	tags, _, err := c.client.ListRepoTags(t[0], t[1], gitea.ListRepoTagsOptions{})
	if err != nil {
		return nil, err
	}

	var ret []sdk.VCSTag
	for _, tag := range tags {
		ret = append(ret, sdk.VCSTag{
			Tag: tag.Name,
			Sha: tag.Commit.SHA,
		})
	}

	return ret, nil
}

func (c *giteaClient) Tag(ctx context.Context, fullname string, tagName string) (sdk.VCSTag, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return sdk.VCSTag{}, fmt.Errorf("giteaCliet.Tag> invalid fullname %s", fullname)
	}
	tag, _, err := c.client.GetTag(t[0], t[1], tagName)
	if err != nil {
		return sdk.VCSTag{}, err
	}
	return sdk.VCSTag{
		Tag: tag.Name,
		Sha: tag.Commit.SHA,
	}, nil
}
