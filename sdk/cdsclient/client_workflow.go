package cdsclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkflowList(projectKey string) ([]sdk.Workflow, error) {
	url := fmt.Sprintf("/project/%s/workflows", projectKey)
	w := []sdk.Workflow{}
	if _, err := c.GetJSON(context.Background(), url, &w); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) WorkflowGet(projectKey, workflowName string, mods ...RequestModifier) (*sdk.Workflow, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s", projectKey, workflowName)
	w := &sdk.Workflow{}
	if _, err := c.GetJSON(context.Background(), url, &w, mods...); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) WorkflowUpdate(projectKey, name string, wf *sdk.Workflow) error {
	url := fmt.Sprintf("/project/%s/workflows/%s", projectKey, name)
	if _, err := c.PutJSON(context.Background(), url, wf, wf); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowGroupAdd(projectKey, name, groupName string, permission int) error {
	gp := sdk.GroupPermission{
		Group:      sdk.Group{Name: groupName},
		Permission: permission,
	}
	url := fmt.Sprintf("/project/%s/workflows/%s/groups", projectKey, name)
	if _, err := c.PostJSON(context.Background(), url, gp, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowGroupDelete(projectKey, name, groupName string) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/groups/%s", projectKey, name, groupName)
	if _, err := c.DeleteJSON(context.Background(), url, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowRunGet(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d", projectKey, workflowName, number)
	run := sdk.WorkflowRun{}
	if _, err := c.GetJSON(context.Background(), url, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowRunsDeleteByBranch(projectKey string, workflowName string, branch string) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/branch/%s", projectKey, workflowName, url.PathEscape(branch))
	if _, err := c.DeleteJSON(context.Background(), url, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowRunResync(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/resync", projectKey, workflowName, number)
	var run sdk.WorkflowRun
	if _, err := c.PostJSON(context.Background(), url, nil, &run); err != nil {
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
	if _, err := c.GetJSON(context.Background(), path, &runs); err != nil {
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
	if _, err := c.GetJSON(context.Background(), url, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (c *client) WorkflowDelete(projectKey string, workflowName string) error {
	_, err := c.DeleteJSON(context.Background(), fmt.Sprintf("/project/%s/workflows/%s", projectKey, workflowName), nil)
	return err
}

func (c *client) WorkflowRunArtifacts(projectKey string, workflowName string, number int64) ([]sdk.WorkflowNodeRunArtifact, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/artifacts", projectKey, workflowName, number)
	arts := []sdk.WorkflowNodeRunArtifact{}
	if _, err := c.GetJSON(context.Background(), url, &arts); err != nil {
		return nil, err
	}
	return arts, nil
}

func (c *client) WorkflowNodeRun(projectKey string, workflowName string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d", projectKey, workflowName, number, nodeRunID)
	run := sdk.WorkflowNodeRun{}
	if _, err := c.GetJSON(context.Background(), url, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func (c *client) WorkflowRunNumberGet(projectKey string, workflowName string) (*sdk.WorkflowRunNumber, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/num", projectKey, workflowName)
	runNumber := sdk.WorkflowRunNumber{}
	if _, err := c.GetJSON(context.Background(), url, &runNumber); err != nil {
		return nil, err
	}
	return &runNumber, nil
}

func (c *client) WorkflowRunNumberSet(projectKey string, workflowName string, number int64) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/num", projectKey, workflowName)
	runNumber := sdk.WorkflowRunNumber{Num: number}
	code, err := c.PostJSON(context.Background(), url, runNumber, nil)
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
	if _, err := c.GetJSON(context.Background(), url, &buildState); err != nil {
		return nil, err
	}
	return &buildState, nil
}

func (c *client) WorkflowNodeRunArtifactDownload(projectKey string, workflowName string, a sdk.WorkflowNodeRunArtifact, w io.Writer) error {
	var url = fmt.Sprintf("/project/%s/workflows/%s/artifact/%d", projectKey, workflowName, a.ID)
	var reader io.ReadCloser
	var err error

	if a.TempURL != "" {
		url = a.TempURL
	}

	reader, _, _, err = c.Stream(context.Background(), "GET", url, nil, true)
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(w, reader)
	return err
}

func (c *client) WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/nodes/%d/release", projectKey, workflowName, runNumber, nodeRunID)
	btes, _ := json.Marshal(release)
	res, _, code, err := c.Stream(context.Background(), "POST", url, bytes.NewReader(btes), true)
	if err != nil {
		return err
	}
	defer res.Close()
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
	code, err := c.PostJSON(context.Background(), url, &content, run)
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
	code, err := c.PostJSON(context.Background(), url, &content, run)
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
	code, err := c.PostJSON(context.Background(), url, nil, run)
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
	code, err := c.PostJSON(context.Background(), url, nil, nodeRun)
	if err != nil {
		return nil, err
	}
	if code >= 300 {
		return nil, fmt.Errorf("Cannot stop workflow node %d. HTTP code error: %d", fromNodeID, code)
	}

	return nodeRun, nil
}

func (c *client) WorkflowCachePush(projectKey, integrationName, ref string, tarContent io.Reader, size int) error {
	store := new(sdk.ArtifactsStore)
	uri := fmt.Sprintf("/project/%s/storage/%s", projectKey, integrationName)
	_, _ = c.GetJSON(context.Background(), uri, store)
	if store.TemporaryURLSupported {
		err := c.workflowCachePushIndirectUpload(projectKey, integrationName, ref, tarContent, size)
		return err
	}
	return c.workflowCachePushDirectUpload(projectKey, integrationName, ref, tarContent)
}

func (c *client) workflowCachePushDirectUpload(projectKey, integrationName, ref string, tarContent io.Reader) error {
	mods := []RequestModifier{
		(func(r *http.Request) {
			r.Header.Set("Content-Type", "application/tar")
		}),
	}

	uri := fmt.Sprintf("/project/%s/storage/%s/cache/%s", projectKey, integrationName, ref)
	_, _, code, err := c.Stream(context.Background(), "POST", uri, tarContent, true, mods...)
	if err != nil {
		return err
	}

	if code >= 400 {
		return fmt.Errorf("HTTP Code %d", code)
	}

	return nil
}

func (c *client) workflowCachePushIndirectUpload(projectKey, integrationName, ref string, tarContent io.Reader, size int) error {
	uri := fmt.Sprintf("/project/%s/storage/%s/cache/%s/url", projectKey, integrationName, ref)
	cacheObj := sdk.Cache{}
	code, err := c.PostJSON(context.Background(), uri, cacheObj, &cacheObj)
	if err != nil {
		return err
	}

	if code >= 400 {
		return fmt.Errorf("HTTP Code %d", code)
	}

	return c.workflowCachePushIndirectUploadPost(cacheObj.TmpURL, tarContent, size)
}

func (c *client) workflowCachePushIndirectUploadPost(url string, tarContent io.Reader, size int) error {
	//Post the file to the temporary URL
	var retry = 10
	var globalErr error
	var body []byte
	for i := 0; i < retry; i++ {
		req, errRequest := http.NewRequest("PUT", url, tarContent)
		if errRequest != nil {
			return errRequest
		}
		req.Header.Set("Content-Type", "application/tar")
		req.ContentLength = int64(size)

		var resp *http.Response
		resp, globalErr = http.DefaultClient.Do(req)

		if globalErr == nil {
			defer resp.Body.Close()

			var err error
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				globalErr = err
				continue
			}

			if resp.StatusCode >= 300 {
				globalErr = fmt.Errorf("[%d] Unable to upload cache: (HTTP %d) %s", i, resp.StatusCode, string(body))
				continue
			}

			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return globalErr
}

func (c *client) WorkflowCachePull(projectKey, integrationName, ref string) (io.Reader, error) {
	uri := fmt.Sprintf("/project/%s/storage/%s", projectKey, integrationName)
	store := new(sdk.ArtifactsStore)
	_, _ = c.GetJSON(context.Background(), uri, store)

	downloadURL := fmt.Sprintf("/project/%s/storage/%s/cache/%s", projectKey, integrationName, ref)

	if store.TemporaryURLSupported {
		url := fmt.Sprintf("/project/%s/storage/%s/cache/%s/url", projectKey, integrationName, ref)
		var cacheObj sdk.Cache
		code, err := c.GetJSON(context.Background(), url, &cacheObj)
		if err != nil {
			return nil, err
		}
		if code >= 400 {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
		downloadURL = cacheObj.TmpURL
	}

	mods := []RequestModifier{
		(func(r *http.Request) {
			r.Header.Set("Content-Type", "application/tar")
		}),
	}

	res, _, code, err := c.Stream(context.Background(), "GET", downloadURL, nil, true, mods...)
	if err != nil {
		return nil, err
	}

	if code >= 400 {
		if code == 404 {
			return nil, fmt.Errorf("Cache not found")
		}
		return nil, fmt.Errorf("HTTP Code %d", code)
	}

	body, err := ioutil.ReadAll(res)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(body), nil
}

func (c *client) WorkflowTemplateInstanceGet(projectKey, workflowName string) (*sdk.WorkflowTemplateInstance, error) {
	url := fmt.Sprintf("/project/%s/workflow/%s/templateInstance", projectKey, workflowName)

	var i sdk.WorkflowTemplateInstance
	if _, err := c.GetJSON(context.Background(), url, &i); err != nil {
		return nil, err
	}

	return &i, nil
}
