package cdsclient

import (
	"context"
	"fmt"
	"github.com/ovh/cds/sdk"
	"net/url"
)

func (c *client) WorkflowV2Run(ctx context.Context, projectKey, vcsIdentifier, repoIdentifier, wkfName string, mods ...RequestModifier) (*sdk.V2WorkflowRun, error) {
	var run sdk.V2WorkflowRun
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workflow/%s/run", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier), wkfName)
	_, _, _, err := c.RequestJSON(ctx, "POST", path, nil, &run, mods...)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowV2RunStatus(ctx context.Context, projectKey, vcsIdentifier, repoIdentifier, wkfName string, runNumber int64) (*sdk.V2WorkflowRun, error) {
	var run sdk.V2WorkflowRun
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workflow/%s/run/%d", projectKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier), wkfName, runNumber)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &run)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowV2RunJobs(ctx context.Context, projKey, vcsIdentifier, repoIdentifier, wkfName string, runNumber int64) ([]sdk.V2WorkflowRunJob, error) {
	var runJobs []sdk.V2WorkflowRunJob
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workflow/%s/run/%d/jobs", projKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier), wkfName, runNumber)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &runJobs)
	if err != nil {
		return nil, err
	}
	return runJobs, nil
}

func (c *client) WorkflowV2RunJobLogLinks(ctx context.Context, projKey, vcsIdentifier, repoIdentifier, wkfName string, runNumber int64, jobName string) (sdk.CDNLogLinks, error) {
	var logsLinks sdk.CDNLogLinks
	path := fmt.Sprintf("/v2/project/%s/vcs/%s/repository/%s/workflow/%s/run/%d/jobs/%s/logs/links", projKey, url.PathEscape(vcsIdentifier), url.PathEscape(repoIdentifier), wkfName, runNumber, jobName)
	_, _, _, err := c.RequestJSON(ctx, "GET", path, nil, &logsLinks)
	if err != nil {
		return logsLinks, err
	}
	return logsLinks, nil
}
