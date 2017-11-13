package cdsclient

import (
	"fmt"
	"io"
	"log"

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

func (c *client) WorkflowGet(projectKey, workflowName string) (*sdk.Workflow, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s", projectKey, workflowName)
	w := &sdk.Workflow{}
	if _, err := c.GetJSON(url, &w); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) WorkflowRunGet(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d", projectKey, workflowName, number)
	run := sdk.WorkflowRun{}
	if _, err := c.GetJSON(url, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowDelete(projectKey string, workflowName string) error {
	code, err := c.DeleteJSON(fmt.Sprintf("/project/%s/workflows/%s", projectKey, workflowName), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) WorkflowRunArtifacts(projectKey string, workflowName string, number int64) ([]sdk.Artifact, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/artifacts", projectKey, workflowName, number)
	arts := []sdk.Artifact{}
	if _, err := c.GetJSON(url, &arts); err != nil {
		return nil, err
	}
	return arts, nil
}

func (c *client) WorkflowNodeRun(projectKey string, workflowName string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d", projectKey, workflowName, number, nodeRunID)
	run := sdk.WorkflowNodeRun{}
	if _, err := c.GetJSON(url, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowNodeRunJobStep(projectKey string, workflowName string, number int64, nodeRunID, job int64, step int) (*sdk.BuildState, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/job/%d/step/%d", projectKey, workflowName, number, nodeRunID, job, step)
	buildState := sdk.BuildState{}
	if _, err := c.GetJSON(url, &buildState); err != nil {
		return nil, err
	}
	return &buildState, nil
}

func (c *client) WorkflowNodeRunArtifacts(projectKey string, workflowName string, number int64, nodeRunID int64) ([]sdk.Artifact, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/artifacts", projectKey, workflowName, number, nodeRunID)
	arts := []sdk.Artifact{}
	if _, err := c.GetJSON(url, &arts); err != nil {
		return nil, err
	}
	return arts, nil
}

func (c *client) WorkflowNodeRunArtifactDownload(projectKey string, workflowName string, artifactID int64, w io.Writer) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/artifact/%d", projectKey, workflowName, artifactID)
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
		return fmt.Errorf("Cannot create workflow node run release. HTTP code error : %d", code)
	}
	return nil
}

func (c *client) WorkflowRunFromHook(projectKey string, workflowName string, hook sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error) {
	if c.config.Verbose {
		log.Println("Payload: ", hook.Payload)
	}

	url := fmt.Sprintf("/project/%s/workflows/%s/runs", projectKey, workflowName)
	content := sdk.WorkflowRunPostHandlerOption{Hook: &hook}
	run := &sdk.WorkflowRun{}
	code, err := c.PostJSON(url, &content, run)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("Cannot create workflow node run release. HTTP code error : %d", code)
	}
	return run, nil
}

func (c *client) WorkflowRunFromManual(projectKey string, workflowName string, manual sdk.WorkflowNodeRunManual, number, fromNodeID int64) (*sdk.WorkflowRun, error) {
	if c.config.Verbose {
		log.Println("Payload: ", manual.Payload)
	}

	url := fmt.Sprintf("/project/%s/workflows/%s/runs", projectKey, workflowName)
	content := sdk.WorkflowRunPostHandlerOption{Manual: &manual}
	if number > 0 {
		content.Number = &number
	}
	if fromNodeID > 0 {
		content.FromNodeIDs = []int64{fromNodeID}
	}
	run := &sdk.WorkflowRun{}
	code, err := c.PostJSON(url, &content, run)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("Cannot run workflow node. HTTP code error: %d", code)
	}

	return run, nil
}
