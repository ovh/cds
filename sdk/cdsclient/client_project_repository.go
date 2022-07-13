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
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository", projectKey, url.PathEscape(vcsName))
	_, err := c.PostJSON(ctx, path, &repo, nil)
	return err
}

func (c *client) ProjectVCSRepositoryList(ctx context.Context, projectKey string, vcsName string) ([]sdk.ProjectRepository, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository", projectKey, url.PathEscape(vcsName))
	var repositories []sdk.ProjectRepository
	if _, err := c.GetJSON(ctx, path, &repositories); err != nil {
		return nil, err
	}
	return repositories, nil
}

func (c *client) ProjectRepositoryDelete(ctx context.Context, projectKey string, vcsName string, repositoryName string) error {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s", projectKey, url.PathEscape(vcsName), url.PathEscape(repositoryName))
	if _, err := c.DeleteJSON(ctx, path, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectRepositoryAnalyze(ctx context.Context, analyze sdk.AnalyzeRequest) (sdk.AnalyzeResponse, error) {
	path := "/v2/repository/analyze"
	var resp sdk.AnalyzeResponse
	_, err := c.PostJSON(ctx, path, &analyze, &resp)
	return resp, err
}
