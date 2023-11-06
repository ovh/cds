package grpcplugins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func GetRunResults(workerHTTPPort int32) ([]sdk.WorkflowRunResult, error) {
	if workerHTTPPort == 0 {
		return nil, nil
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/run-result", workerHTTPPort), nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request to get run result: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot get run result /run-result: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read body on get run result /run-result: %v", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cannot get run result /run-result: HTTP %d", resp.StatusCode)
	}

	var results []sdk.WorkflowRunResult
	if err := sdk.JSONUnmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unable to unmarshal response: %v", err)
	}
	return results, nil
}

func GetWorkerDirectories(workerHTTPPort int32, c *actionplugin.Common) (*sdk.WorkerDirectories, error) {
	if workerHTTPPort == 0 {
		return nil, errors.Errorf("worker port must not be 0")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/directories", workerHTTPPort), nil)
	if err != nil {
		return nil, errors.Errorf("unable to create request to get directories: %v", err)
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create run result")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("unable to read body on get run result /working-directory: %v", err)
	}

	if resp.StatusCode >= 300 {
		return nil, errors.Errorf("cannot get working directory: HTTP %d", resp.StatusCode)
	}

	var workDir sdk.WorkerDirectories
	if err := sdk.JSONUnmarshal(body, &workDir); err != nil {
		return nil, errors.Errorf("unable to unmarshal response: %v", err)
	}
	return &workDir, nil
}

func CreateRunResult(ctx context.Context, c *actionplugin.Common, result *workerruntime.V2RunResultRequest) (*workerruntime.V2AddResultResponse, error) {
	btes, err := json.Marshal(result)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req, err := c.NewRequest(ctx, http.MethodPost, "/v2/result", bytes.NewReader(btes))
	if err != nil {
		return nil, err
	}

	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create run result")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if resp.StatusCode >= 300 {
		return nil, errors.Wrapf(err, "unable to create run result (status code %d) %v", resp.StatusCode, string(body))
	}

	var response workerruntime.V2AddResultResponse
	if err := sdk.JSONUnmarshal(body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func UpdateRunResult(ctx context.Context, c *actionplugin.Common, result *workerruntime.V2RunResultRequest) (*workerruntime.V2UpdateResultResponse, error) {
	btes, err := json.Marshal(result)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req, err := c.NewRequest(ctx, http.MethodPut, "/v2/result", bytes.NewReader(btes))
	if err != nil {
		return nil, err
	}
	resp, err := c.DoRequest(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to update run result")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if resp.StatusCode >= 300 {
		return nil, errors.Wrapf(err, "unable to update run result (status code %d) %v", resp.StatusCode, string(body))
	}

	var response workerruntime.V2UpdateResultResponse
	if err := sdk.JSONUnmarshal(body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
