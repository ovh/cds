package forgejo

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) ListForks(ctx context.Context, fullname string) ([]sdk.VCSRepo, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	var repos []*Repository
	apiPath := fmt.Sprintf("/repos/%s/%s/forks", owner, repo)
	if _, err := f.client.get(ctx, apiPath, &repos); err != nil {
		return nil, err
	}

	var ret []sdk.VCSRepo
	for _, repo := range repos {
		ret = append(ret, f.ToVCSRepo(repo))
	}

	return ret, nil
}
