package forgejo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (f *forgejoClient) ListForks(ctx context.Context, fullname string) ([]sdk.VCSRepo, error) {
	owner, repo, err := getRepo(fullname)
	if err != nil {
		return nil, err
	}

	var repos []*Repository
	apiPath := fmt.Sprintf("/repos/%s/%s/forks", owner, repo)
	if resp, err := f.client.get(ctx, apiPath, &repos); err != nil {
		// With DISABLE_FORKS, Forgejo does not register this route: no fork can exist
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return []sdk.VCSRepo{}, nil
		}
		return nil, err
	}

	var ret []sdk.VCSRepo
	for _, repo := range repos {
		ret = append(ret, f.ToVCSRepo(repo))
	}

	return ret, nil
}
