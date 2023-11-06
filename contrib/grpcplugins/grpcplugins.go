package grpcplugins

import (
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk"
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

func GetWorkerDirectories(workerHTTPPort int32) (*sdk.WorkerDirectories, error) {
	if workerHTTPPort == 0 {
		return nil, errors.Errorf("worker port must not be 0")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/directories", workerHTTPPort), nil)
	if err != nil {
		return nil, errors.Errorf("unable to create request to get directories: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("cannot get run result directories: %v", err)
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
