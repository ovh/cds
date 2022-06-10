package cdsclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVCSRepositoryAdd(ctx context.Context, projectKey string, vcsName string, repoName string) error {
	repo := sdk.ProjectRepository{
		Name: repoName,
	}
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository", projectKey, vcsName)
	_, err := c.PostJSON(ctx, path, &repo, nil)
	return err
}

func (c *client) ProjectVCSRepositoryList(ctx context.Context, projectKey string, vcsName string) ([]sdk.ProjectRepository, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository", projectKey, vcsName)
	var repositories []sdk.ProjectRepository
	if _, err := c.GetJSON(ctx, path, &repositories); err != nil {
		return nil, err
	}
	return repositories, nil
}

func (c *client) ProjectRepositoryDelete(ctx context.Context, projectKey string, vcsName string, repositoryName string) error {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s", projectKey, vcsName, url.PathEscape(repositoryName))
	if _, err := c.DeleteJSON(ctx, path, nil); err != nil {
		return err
	}
	return nil
}
