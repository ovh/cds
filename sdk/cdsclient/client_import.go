package cdsclient

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) PipelineImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error) {
	url := fmt.Sprintf("/project/%s/import/pipeline", projectKey)

	btes, _, _, err := c.Request(context.Background(), "POST", url, content, mods...)
	if err != nil {
		return nil, err
	}

	messages := []string{}
	_ = json.Unmarshal(btes, &messages)
	return messages, nil
}

func (c *client) ApplicationImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error) {
	url := fmt.Sprintf("/project/%s/import/application", projectKey)

	btes, _, _, err := c.Request(context.Background(), "POST", url, content, mods...)
	if err != nil {
		return nil, err
	}

	messages := []string{}
	_ = json.Unmarshal(btes, &messages)
	return messages, nil
}

func (c *client) EnvironmentImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error) {
	url := fmt.Sprintf("/project/%s/import/environment", projectKey)

	btes, _, _, err := c.Request(context.Background(), "POST", url, content, mods...)
	if err != nil {
		return nil, err
	}

	messages := []string{}
	_ = json.Unmarshal(btes, &messages)
	return messages, nil
}

// WorkerModelImport import a worker model via as code
func (c *client) WorkerModelImport(content io.Reader, mods ...RequestModifier) (*sdk.Model, error) {
	url := "/worker/model/import"

	btes, _, code, err := c.Request(context.Background(), "POST", url, content, mods...)
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("HTTP Status code %d", code)
	}

	var wm sdk.Model
	if err := json.Unmarshal(btes, &wm); err != nil {
		return nil, err
	}

	return &wm, nil
}

func (c *client) WorkflowImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error) {
	url := fmt.Sprintf("/project/%s/import/workflows", projectKey)

	btes, _, _, err := c.Request(context.Background(), "POST", url, content, mods...)
	messages := []string{} // could contains msg even if there is a 400 returned
	_ = json.Unmarshal(btes, &messages)
	return messages, err
}

func (c *client) WorkflowPush(projectKey string, tarContent io.Reader, mods ...RequestModifier) ([]string, *tar.Reader, error) {
	url := fmt.Sprintf("/project/%s/push/workflows", projectKey)

	mods = append(mods, func(r *http.Request) {
		r.Header.Set("Content-Type", "application/tar")
	})

	btes, headers, code, err := c.Request(context.Background(), "POST", url, tarContent, mods...)
	if err != nil {
		return nil, nil, err
	}
	if code >= 400 {
		return nil, nil, fmt.Errorf("HTTP Status code %d", code)
	}

	messages := []string{}
	if err := json.Unmarshal(btes, &messages); err != nil {
		return nil, nil, err
	}

	wName := headers.Get(sdk.ResponseWorkflowNameHeader)
	if wName == "" {
		return messages, nil, nil
	}
	tarReader, err := c.WorkflowPull(projectKey, wName, mods...)
	if err != nil {
		return nil, nil, err
	}

	return messages, tarReader, nil
}
