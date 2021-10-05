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

func (c *client) WorkflowSearch(opts ...RequestModifier) ([]sdk.Workflow, error) {
	url := fmt.Sprintf("/workflow/search")
	w := []sdk.Workflow{}
	if _, err := c.GetJSON(context.Background(), url, &w, opts...); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) WorkflowList(projectKey string, opts ...RequestModifier) ([]sdk.Workflow, error) {
	url := fmt.Sprintf("/project/%s/workflows", projectKey)
	w := []sdk.Workflow{}
	if _, err := c.GetJSON(context.Background(), url, &w, opts...); err != nil {
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

func (c *client) WorkflowLabelAdd(projectKey, name, labelName string) error {
	lbl := sdk.Label{
		Name: labelName,
	}
	url := fmt.Sprintf("/project/%s/workflows/%s/label", projectKey, name)
	if _, err := c.PostJSON(context.Background(), url, lbl, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowLabelDelete(projectKey, name string, labelID int64) error {
	url := fmt.Sprintf("/project/%s/workflows/%s/label/%d", projectKey, name, labelID)
	if _, err := c.DeleteJSON(context.Background(), url, nil); err != nil {
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

func (c *client) WorkflowDelete(projectKey string, workflowName string, opts ...RequestModifier) error {
	_, err := c.DeleteJSON(context.Background(), fmt.Sprintf("/project/%s/workflows/%s", projectKey, workflowName), nil, opts...)
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

func (c *client) WorkflowRunArtifactsLinks(projectKey string, workflowName string, number int64) (sdk.CDNItemLinks, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/artifacts/links", projectKey, workflowName, number)
	var resp sdk.CDNItemLinks
	if _, err := c.GetJSON(context.Background(), url, &resp); err != nil {
		return resp, err
	}
	return resp, nil
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
		return newAPIError(fmt.Errorf("Cannot update workflow run number. HTTP code error : %d", code))
	}
	return nil
}

func (c *client) WorkflowNodeRunJobStepLinks(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64) (*sdk.CDNLogLinks, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/nodes/%d/job/%d/links", projectKey, workflowName, nodeRunID, job)
	var a sdk.CDNLogLinks
	if _, err := c.GetJSON(ctx, url, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *client) WorkflowNodeRunJobStepLink(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, step int64) (*sdk.CDNLogLink, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/nodes/%d/job/%d/step/%d/link", projectKey, workflowName, nodeRunID, job, step)
	var a sdk.CDNLogLink
	if _, err := c.GetJSON(ctx, url, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *client) WorkflowNodeRunJobServiceLink(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, serviceName string) (*sdk.CDNLogLink, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/nodes/%d/job/%d/service/%s/link", projectKey, workflowName, nodeRunID, job, serviceName)
	var a sdk.CDNLogLink
	if _, err := c.GetJSON(ctx, url, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *client) WorkflowAccess(ctx context.Context, projectKey string, workflowID int64, sessionID string, itemType sdk.CDNItemType) error {
	url := fmt.Sprintf("/project/%s/workflows/%d/type/%s/access", projectKey, workflowID, itemType)
	if _, err := c.GetJSON(ctx, url, nil, SetHeader(sdk.CDSSessionID, sessionID)); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowLogDownload(ctx context.Context, link sdk.CDNLogLink) ([]byte, error) {
	downloadURL := fmt.Sprintf("%s/item/%s/%s/download", link.CDNURL, link.ItemType, link.APIRef)
	data, _, _, err := c.Request(context.Background(), http.MethodGet, downloadURL, nil, func(req *http.Request) {
		auth := "Bearer " + c.config.SessionToken
		req.Header.Add("Authorization", auth)
	})
	if err != nil {
		return nil, newError(fmt.Errorf("can't download log from: %s: %v", downloadURL, err))
	}
	return data, nil
}

func (c *client) WorkflowNodeRunArtifactDownload(projectKey string, workflowName string, a sdk.WorkflowNodeRunArtifact, w io.Writer) error {
	var url = fmt.Sprintf("/project/%s/workflows/%s/artifact/%d", projectKey, workflowName, a.ID)
	var reader io.ReadCloser
	var err error

	if a.TempURL != "" {
		url = a.TempURL
	}

	reader, _, _, err = c.Stream(context.Background(), c.HTTPNoTimeoutClient(), "GET", url, nil)
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
	res, _, code, err := c.Stream(context.Background(), c.HTTPNoTimeoutClient(), "POST", url, bytes.NewReader(btes))
	if err != nil {
		return err
	}
	defer res.Close()
	if code >= 300 {
		return newAPIError(fmt.Errorf("Cannot create workflow node run release. HTTP code error : %d", code))
	}
	return nil
}

func (c *client) WorkflowRunFromHook(projectKey string, workflowName string, hook sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error) {
	// Check that the hook exists before run it
	w, err := c.WorkflowGet(projectKey, workflowName)
	if err != nil {
		return nil, err
	}

	hooks := w.WorkflowData.GetHooks()
	if _, has := hooks[hook.WorkflowNodeHookUUID]; !has {
		// If the hook doesn't exist, raise an error
		return nil, sdk.ErrHookNotFound
	}

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
		return nil, newAPIError(fmt.Errorf("Cannot create workflow node run release. HTTP code error : %d", code))
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
		return nil, newAPIError(fmt.Errorf("Cannot run workflow node. HTTP code error: %d", code))
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
		return nil, newAPIError(fmt.Errorf("Cannot stop workflow %s. HTTP code error: %d", workflowName, code))
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
		return nil, newAPIError(fmt.Errorf("Cannot stop workflow node %d. HTTP code error: %d", fromNodeID, code))
	}

	return nodeRun, nil
}

func (c *client) WorkflowCachePush(projectKey, integrationName, ref string, tarContent io.Reader, size int) error {
	store := new(sdk.ArtifactsStore)
	uri := fmt.Sprintf("/project/%s/storage/%s", projectKey, integrationName)
	_, _ = c.GetJSON(context.Background(), uri, store)
	if store.TemporaryURLSupported {
		return c.workflowCachePushIndirectUpload(projectKey, integrationName, ref, tarContent, size)
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
	_, _, code, err := c.Stream(context.Background(), c.HTTPNoTimeoutClient(), "POST", uri, tarContent, mods...)
	if err != nil {
		return err
	}

	if code >= 400 {
		return newAPIError(fmt.Errorf("HTTP Code %d", code))
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
		return newAPIError(fmt.Errorf("HTTP Code %d", code))
	}

	// FIXME temporary fix that will be deprecated with cdn artifacts
	time.Sleep(2 * time.Second)

	return c.workflowCachePushIndirectUploadPost(cacheObj.TmpURL, tarContent, size)
}

func (c *client) workflowCachePushIndirectUploadPost(url string, tarContent io.Reader, size int) error {
	req, err := http.NewRequest("PUT", url, tarContent)
	if err != nil {
		return newError(err)
	}
	req.Header.Set("Content-Type", "application/tar")
	req.ContentLength = int64(size)

	resp, err := c.HTTPNoTimeoutClient().Do(req)
	if err != nil {
		return newTransportError(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return newError(err)
	}

	if resp.StatusCode >= 300 {
		return newAPIError(fmt.Errorf("unable to upload cache: (HTTP %d) %s", resp.StatusCode, string(body)))
	}
	return nil
}

func (c *client) WorkflowRunResultsList(ctx context.Context, projectKey string, name string, number int64) ([]sdk.WorkflowRunResult, error) {
	var results []sdk.WorkflowRunResult
	uri := fmt.Sprintf("/project/%s/workflows/%s/runs/%d/results", projectKey, name, number)
	if _, err := c.GetJSON(ctx, uri, &results); err != nil {
		return nil, err
	}
	return results, nil
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
			return nil, newAPIError(fmt.Errorf("HTTP Code %d", code))
		}
		downloadURL = cacheObj.TmpURL
	}

	mods := []RequestModifier{
		(func(r *http.Request) {
			r.Header.Set("Content-Type", "application/tar")
		}),
	}

	res, _, code, err := c.Stream(context.Background(), c.HTTPNoTimeoutClient(), "GET", downloadURL, nil, mods...)
	if err != nil {
		return nil, err
	}

	if code >= 400 {
		if code == 404 {
			return nil, newError(fmt.Errorf("Cache not found"))
		}
		return nil, newAPIError(fmt.Errorf("HTTP Code %d", code))
	}

	return res, nil
}
