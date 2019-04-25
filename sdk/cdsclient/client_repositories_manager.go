package cdsclient

import (
	"context"
	"strconv"

	"github.com/ovh/cds/sdk"
)

func (c *client) RepositoriesList(projectKey string, repoManager string, resync bool) ([]sdk.VCSRepo, error) {
	repos := []sdk.VCSRepo{}
	path := "/project/" + projectKey + "/repositories_manager/" + repoManager + "/repos?synchronize=" + strconv.FormatBool(resync)
	if _, err := c.GetJSON(context.Background(), path, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}
