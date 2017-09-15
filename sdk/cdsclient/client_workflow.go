package cdsclient

import (
	"io"

	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkflowList(projectKey string) ([]sdk.Workflow, error) {
	url := fmt.Sprintf("/project/%s/workflows", projectKey)
	w := []sdk.Workflow{}
	if _, err := c.GetJSON(url, &w); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) WorkflowGet(projectKey, name string) (*sdk.Workflow, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s", projectKey, name)
	w := &sdk.Workflow{}
	if _, err := c.GetJSON(url, &w); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) WorkflowRun(projectKey string, name string, number int64) (*sdk.WorkflowRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d", projectKey, name, number)
	run := sdk.WorkflowRun{}
	if _, err := c.GetJSON(url, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.Artifact, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/artifacts", projectKey, name, number)
	arts := []sdk.Artifact{}
	if _, err := c.GetJSON(url, &arts); err != nil {
		return nil, err
	}
	return arts, nil
}

func (c *client) WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d", projectKey, name, number, nodeRunID)
	run := sdk.WorkflowNodeRun{}
	if _, err := c.GetJSON(url, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowNodeRunArtifacts(projectKey string, name string, number int64, nodeRunID int64) ([]sdk.Artifact, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/artifacts", projectKey, name, number, nodeRunID)
	arts := []sdk.Artifact{}
	if _, err := c.GetJSON(url, &arts); err != nil {
		return nil, err
	}
	return arts, nil
}

func (c *client) WorkflowNodeRunArtifactDownload(projectKey string, name string, artifactID int64, w io.Writer) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/artifact/%d", projectKey, name, artifactID)
	reader, _, err := c.Stream("GET", url, nil, true)
	if err != nil {
		return err
	}
	defer reader.Close()
	if _, err := io.Copy(w, reader); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/release", projectKey, workflowName, runNumber, nodeRunID)
	code, err := c.PostJSON(url, release, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("Cannot create workflow node run release. Http code error : %d", code)
	}
	return nil
}
