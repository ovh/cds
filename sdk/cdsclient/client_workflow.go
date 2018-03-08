package cdsclient

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

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

func (c *client) WorkflowRunSearch(projectKey string, offset, limit int64, filters ...Filter) ([]sdk.WorkflowRun, error) {
	if offset < 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 50
	}

	path := fmt.Sprintf("/project/%s/runs?offset=%d&limit=%d", projectKey, offset, limit)
	for _, f := range filters {
		path += fmt.Sprintf("&%s=%s", url.QueryEscape(f.Name), url.QueryEscape(f.Value))
	}
	runs := []sdk.WorkflowRun{}
	if _, err := c.GetJSON(path, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (c *client) WorkflowRunList(projectKey string, workflowName string, offset, limit int64) ([]sdk.WorkflowRun, error) {
	if offset < 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 50
	}

	url := fmt.Sprintf("/project/%s/workflows/%s/runs?offset=%d&limit=%d", projectKey, workflowName, offset, limit)
	runs := []sdk.WorkflowRun{}
	if _, err := c.GetJSON(url, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (c *client) WorkflowDelete(projectKey string, workflowName string) error {
	_, err := c.DeleteJSON(fmt.Sprintf("/project/%s/workflows/%s", projectKey, workflowName), nil)
	return err
}

func (c *client) WorkflowRunArtifacts(projectKey string, workflowName string, number int64) ([]sdk.WorkflowNodeRunArtifact, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/artifacts", projectKey, workflowName, number)
	arts := []sdk.WorkflowNodeRunArtifact{}
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

func (c *client) WorkflowRunNumberGet(projectKey string, workflowName string) (*sdk.WorkflowRunNumber, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/num", projectKey, workflowName)
	runNumber := sdk.WorkflowRunNumber{}
	if _, err := c.GetJSON(url, &runNumber); err != nil {
		return nil, err
	}
	return &runNumber, nil
}

func (c *client) WorkflowRunNumberSet(projectKey string, workflowName string, number int64) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/num", projectKey, workflowName)
	runNumber := sdk.WorkflowRunNumber{Num: number}
	code, err := c.PostJSON(url, runNumber, nil)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("Cannot update workflow run number. HTTP code error : %d", code)
	}
	return nil
}

func (c *client) WorkflowNodeRunJobStep(projectKey string, workflowName string, number int64, nodeRunID, job int64, step int) (*sdk.BuildState, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/job/%d/step/%d", projectKey, workflowName, number, nodeRunID, job, step)
	buildState := sdk.BuildState{}
	if _, err := c.GetJSON(url, &buildState); err != nil {
		return nil, err
	}
	return &buildState, nil
}

func (c *client) WorkflowNodeRunArtifacts(projectKey string, workflowName string, number int64, nodeRunID int64) ([]sdk.WorkflowNodeRunArtifact, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/artifacts", projectKey, workflowName, number, nodeRunID)
	arts := []sdk.WorkflowNodeRunArtifact{}
	if _, err := c.GetJSON(url, &arts); err != nil {
		return nil, err
	}
	return arts, nil
}

func (c *client) WorkflowNodeRunArtifactDownload(projectKey string, workflowName string, a sdk.WorkflowNodeRunArtifact, w io.Writer) error {
	var url = fmt.Sprintf("/project/%s/workflows/%s/artifact/%d", projectKey, workflowName, a.ID)
	var reader io.ReadCloser
	var err error

	if a.TempURL != "" {
		url = a.TempURL
	}

	reader, _, _, err = c.Stream("GET", url, nil, true)
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(w, reader)
	return err
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

func (c *client) WorkflowStop(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/stop", projectKey, workflowName, number)

	run := &sdk.WorkflowRun{}
	code, err := c.PostJSON(url, nil, run)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("Cannot stop workflow %s. HTTP code error: %d", workflowName, code)
	}

	return run, nil
}

func (c *client) WorkflowNodeStop(projectKey string, workflowName string, number, fromNodeID int64) (*sdk.WorkflowNodeRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/stop", projectKey, workflowName, number, fromNodeID)

	nodeRun := &sdk.WorkflowNodeRun{}
	code, err := c.PostJSON(url, nil, nodeRun)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("Cannot stop workflow node %d. HTTP code error: %d", fromNodeID, code)
	}

	return nodeRun, nil
}

func (c *client) WorkflowCachePush(projectKey, tag string, tarContent io.Reader) error {
	url := fmt.Sprintf("%s/project/%s/cache/%s", c.APIURL(), projectKey, tag)

	req, errRequest := http.NewRequest("POST", url, tarContent)
	if errRequest != nil {
		return sdk.WrapError(errRequest, "WorkflowCachePush> Unable to create request")
	}
	req.Header.Set("Content-Type", "application/tar")

	resp, err := NoTimeout(c.HTTPClient).Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP Code %d", resp.StatusCode)
	}

	return nil
}

func (c *client) WorkflowCachePull(projectKey, tag string) (io.Reader, error) {
	url := fmt.Sprintf("%s/project/%s/cache/%s", c.APIURL(), projectKey, tag)

	req, errRequest := http.NewRequest("GET", url, nil)
	if errRequest != nil {
		return nil, sdk.WrapError(errRequest, "WorkflowCachePull> Unable to create request")
	}
	req.Header.Set("Content-Type", "application/tar")
	resp, err := NoTimeout(c.HTTPClient).Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP Code %d", resp.StatusCode)
	}

	return resp.Body, nil
}
