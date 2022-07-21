package cdsclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVCSRepositoryAdd(ctx context.Context, projectKey string, vcsName string, repo sdk.ProjectRepository) error {
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

func (c *client) ProjectRepositoryAnalyzeList(ctx context.Context, projectKey string, vcsIdentifier string, repositoryIdentifier string) ([]sdk.ProjectRepositoryAnalyze, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/analyze", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repositoryIdentifier))
	var analyzes []sdk.ProjectRepositoryAnalyze
	_, err := c.GetJSON(ctx, path, &analyzes)
	return analyzes, err
}

func (c *client) ProjectRepositoryAnalyzeGet(ctx context.Context, projectKey string, vcsIdentifier string, repositoryIdentifier string, analyzeID string) (sdk.ProjectRepositoryAnalyze, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/analyze/%s", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repositoryIdentifier), analyzeID)
	var analyze sdk.ProjectRepositoryAnalyze
	_, err := c.GetJSON(ctx, path, &analyze)
	return analyze, err
}
