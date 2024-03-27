package cdsclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkflowV2RunFromHook(ctx context.Context, projectKey, vcsIdentifier, repoIdentifier, wkfName string, runRequest sdk.V2WorkflowRunHookRequest, mods ...RequestModifier) (*sdk.V2WorkflowRun, error) {
	var run sdk.V2WorkflowRun
	path := fmt.Sprintf("/v2/hooks/project/%s/vcs/%s/repository/%s/workflow/%s/run", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier), wkfName)
	_, _, _, err := c.RequestJSON(ctx, "POST", path, runRequest, &run, mods...)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowV2Run(ctx context.Context, projectKey, vcsIdentifier, repoIdentifier, wkfName string, payload sdk.V2WorkflowRunManualRequest, mods ...RequestModifier) (*sdk.V2WorkflowRunManualResponse, error) {
	var resp sdk.V2WorkflowRunManualResponse
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workflow/%s/run", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier), wkfName)
	_, _, _, err := c.RequestJSON(ctx, "POST", path, payload, &resp, mods...)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *client) WorkflowV2RunSearchAllProjects(ctx context.Context, offset, limit int64, mods ...RequestModifier) ([]sdk.V2WorkflowRun, error) {
	if offset < 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 50
	}

	mods = append(mods, WithQueryParameter("offset", strconv.FormatInt(offset, 10)))
	mods = append(mods, WithQueryParameter("limit", strconv.FormatInt(limit, 10)))

	var runs []sdk.V2WorkflowRun
	if _, err := c.GetJSON(ctx, "/v2/run", &runs, mods...); err != nil {
		return nil, err
	}
	return runs, nil
}

func (c *client) WorkflowV2RunSearch(ctx context.Context, projectKey string, mods ...RequestModifier) ([]sdk.V2WorkflowRun, error) {
	var runs []sdk.V2WorkflowRun
	path := fmt.Sprintf("/v2/project/%s/run", projectKey)
	_, err := c.GetJSON(ctx, path, &runs, mods...)
	if err != nil {
		return nil, err
	}
	return runs, nil
}

func (c *client) WorkflowV2RunInfoList(ctx context.Context, projectKey, workflowRunID string, mods ...RequestModifier) ([]sdk.V2WorkflowRunInfo, error) {
	var runInfos []sdk.V2WorkflowRunInfo
	path := fmt.Sprintf("/v2/project/%s/run/%s/infos", projectKey, workflowRunID)
	_, err := c.GetJSON(ctx, path, &runInfos, mods...)
	if err != nil {
		return nil, err
	}
	return runInfos, nil
}

func (c *client) WorkflowV2Restart(ctx context.Context, projectKey, workflowRunID string, mods ...RequestModifier) (*sdk.V2WorkflowRun, error) {
	var run sdk.V2WorkflowRun
	path := fmt.Sprintf("/v2/project/%s/run/%s/restart", projectKey, workflowRunID)
	_, _, _, err := c.RequestJSON(ctx, http.MethodPost, path, nil, &run, mods...)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowV2JobStart(ctx context.Context, projectKey, workflowRunID, jobIdentifier string, payload map[string]interface{}, mods ...RequestModifier) (*sdk.V2WorkflowRun, error) {
	var run sdk.V2WorkflowRun
	path := fmt.Sprintf("/v2/project/%s/run/%s/job/%s/run", projectKey, workflowRunID, jobIdentifier)
	_, _, _, err := c.RequestJSON(ctx, http.MethodPost, path, payload, &run, mods...)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowV2RunStatus(ctx context.Context, projectKey, workflowRunID string) (*sdk.V2WorkflowRun, error) {
	var run sdk.V2WorkflowRun
	path := fmt.Sprintf("/v2/project/%s/run/%s", projectKey, workflowRunID)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &run)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowV2RunJobs(ctx context.Context, projKey, workflowRunID string) ([]sdk.V2WorkflowRunJob, error) {
	var runJobs []sdk.V2WorkflowRunJob
	path := fmt.Sprintf("/v2/project/%s/run/%s/job", projKey, workflowRunID)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &runJobs)
	if err != nil {
		return nil, err
	}
	return runJobs, nil
}

func (c *client) WorkflowV2RunJob(ctx context.Context, projKey, workflowRunID, jobRunID string) (*sdk.V2WorkflowRunJob, error) {
	var runJob sdk.V2WorkflowRunJob
	path := fmt.Sprintf("/v2/project/%s/run/%s/job/%s", projKey, workflowRunID, jobRunID)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &runJob)
	if err != nil {
		return nil, err
	}
	return &runJob, nil
}

func (c *client) WorkflowV2RunResultList(ctx context.Context, projKey, runIdentifier string) ([]sdk.V2WorkflowRunResult, error) {
	var results []sdk.V2WorkflowRunResult
	path := fmt.Sprintf("/v2/project/%s/run/%s/result", projKey, runIdentifier)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (c *client) WorkflowV2RunJobLogLinks(ctx context.Context, projKey, workflowRunID, jobRunID string) (sdk.CDNLogLinks, error) {
	var logsLinks sdk.CDNLogLinks
	path := fmt.Sprintf("/v2/project/%s/run/%s/job/%s/logs/links", projKey, workflowRunID, jobRunID)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &logsLinks)
	if err != nil {
		return logsLinks, err
	}
	return logsLinks, nil
}

func (c *client) WorkflowV2Stop(ctx context.Context, projKey, workflowRunID string) error {
	path := fmt.Sprintf("/v2/project/%s/run/%s/stop", projKey, workflowRunID)
	if _, _, _, err := c.RequestJSON(ctx, http.MethodPost, path, nil, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowV2StopJob(ctx context.Context, projKey, workflowRunID, jobIdentifier string) error {
	path := fmt.Sprintf("/v2/project/%s/run/%s/job/%s/stop", projKey, workflowRunID, jobIdentifier)
	if _, _, _, err := c.RequestJSON(ctx, "POST", path, nil, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowV2RunJobInfoList(ctx context.Context, projKey, workflowRunID, jobRunID string) ([]sdk.V2WorkflowRunJobInfo, error) {
	var infos []sdk.V2WorkflowRunJobInfo
	path := fmt.Sprintf("/v2/project/%s/run/%s/job/%s/infos", projKey, workflowRunID, jobRunID)
	if _, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &infos); err != nil {
		return nil, err
	}
	return infos, nil
}
