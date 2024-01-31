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

func (c *client) ProjectRepositoryAnalysis(ctx context.Context, analyze sdk.AnalysisRequest) (sdk.AnalysisResponse, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/analysis", analyze.ProjectKey, url.PathEscape(analyze.VcsName), url.PathEscape(analyze.RepoName))
	var resp sdk.AnalysisResponse
	_, err := c.PostJSON(ctx, path, &analyze, &resp)
	return resp, err
}

func (c *client) ProjectRepositoryAnalysisList(ctx context.Context, projectKey string, vcsIdentifier string, repositoryIdentifier string) ([]sdk.ProjectRepositoryAnalysis, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/analysis", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repositoryIdentifier))
	var analyses []sdk.ProjectRepositoryAnalysis
	_, err := c.GetJSON(ctx, path, &analyses)
	return analyses, err
}

func (c *client) ProjectRepositoryAnalysisGet(ctx context.Context, projectKey string, vcsIdentifier string, repositoryIdentifier string, analyzeID string) (sdk.ProjectRepositoryAnalysis, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/analysis/%s", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repositoryIdentifier), analyzeID)
	var analysis sdk.ProjectRepositoryAnalysis
	_, err := c.GetJSON(ctx, path, &analysis)
	return analysis, err
}

func (c *client) ProjectRepositoryHookSecret(ctx context.Context, projectKey, vcsType, vcsName, repoName string) (sdk.HookAccessData, error) {
	path := fmt.Sprintf("/v2/hooks/%s/vcs/%s/%s/repository/%s/secret", projectKey, vcsType, url.PathEscape(vcsName), url.PathEscape(repoName))
	var hookData sdk.HookAccessData
	_, err := c.GetJSON(ctx, path, &hookData)
	return hookData, err
}
