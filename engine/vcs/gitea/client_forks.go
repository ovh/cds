package gitea

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/ovh/cds/sdk"
)

func (c *giteaClient) ListForks(ctx context.Context, fullname string) ([]sdk.VCSRepo, error) {
	t := strings.Split(fullname, "/")
	if len(t) != 2 {
		return nil, fmt.Errorf("giteaCliet.Tags> invalid fullname %s", fullname)
	}
	repos, _, err := c.client.ListForks(t[0], t[1], gitea.ListForksOptions{})
	if err != nil {
		return nil, err
	}

	var ret []sdk.VCSRepo
	for _, repo := range repos {
		ret = append(ret, c.ToVCSRepo(repo))
	}

	return ret, nil

}
