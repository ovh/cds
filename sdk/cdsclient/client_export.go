package cdsclient

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/ovh/cds/sdk/exportentities"
)

func (c *client) PipelineExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error) {
	pip, err := c.PipelineGet(projectKey, name)
	if err != nil {
		return nil, err
	}

	p := exportentities.NewPipelineV1(*pip, exportWithPermissions)

	if !exportWithPermissions {
		p.Permissions = nil
	}

	f, err := exportentities.GetFormat(exportFormat)
	if err != nil {
		return nil, err
	}

	btes, err := exportentities.Marshal(p, f)
	if err != nil {
		return nil, err
	}
	return btes, nil
}

func (c *client) ApplicationExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/export/application/%s?format=%s", projectKey, name, exportFormat)
	if exportWithPermissions {
		path += "&withPermissions=true"
	}
	body, _, _, err := c.Request(context.Background(), "GET", path, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) EnvironmentExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/export/environment/%s?format=%s", projectKey, name, exportFormat)
	if exportWithPermissions {
		path += "&withPermissions=true"
	}
	body, _, _, err := c.Request(context.Background(), "GET", path, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) WorkflowExport(projectKey, name string, mods ...RequestModifier) ([]byte, error) {
	/*
		path := fmt.Sprintf("/project/%s/export/workflows/%s?format=%s", projectKey, name, exportFormat)
		if exportWithPermissions {
			path += "&withPermissions=true"
		}
	*/
	path := fmt.Sprintf("/project/%s/export/workflows/%s", projectKey, name)
	bodyReader, _, _, err := c.Stream(context.Background(), "GET", path, nil, true, mods...)
	if err != nil {
		return nil, err
	}
	defer bodyReader.Close()

	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) WorkflowPull(projectKey, name string, mods ...RequestModifier) (*tar.Reader, error) {
	path := fmt.Sprintf("/project/%s/pull/workflows/%s", projectKey, name)
	/*if exportWithPermissions {
		path += "?withPermissions=true"
	}
	*/
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	// Open the tar archive for reading.
	r := bytes.NewReader(body)
	tr := tar.NewReader(r)
	return tr, nil
}
