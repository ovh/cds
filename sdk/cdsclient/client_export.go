package cdsclient

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
)

func (c *client) PipelineExport(projectKey, name string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/export/pipeline/%s", projectKey, name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) ApplicationExport(projectKey, name string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/export/application/%s", projectKey, name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) EnvironmentExport(projectKey, name string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/export/environment/%s", projectKey, name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) WorkerModelExport(groupName, name string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/worker/model/%s/%s/export", groupName, name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) WorkflowExport(projectKey, name string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/export/workflows/%s", projectKey, name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) WorkflowPull(projectKey, name string, mods ...RequestModifier) (*tar.Reader, error) {
	path := fmt.Sprintf("/project/%s/pull/workflows/%s", projectKey, name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	// Open the tar archive for reading.
	r := bytes.NewReader(body)
	tr := tar.NewReader(r)
	return tr, nil
}
