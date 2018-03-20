package cdsclient

import "github.com/ovh/cds/sdk"

func (c *client) RepositoriesList(projectKey string, repoManager string) ([]sdk.VCSRepo, error) {
	repos := []sdk.VCSRepo{}
	path := "/project/" + projectKey + "/repositories_manager/" + repoManager + "/repos"
	if _, err := c.GetJSON(path, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}
