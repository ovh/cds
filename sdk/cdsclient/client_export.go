package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk/exportentities"
)

func (c *client) PipelineExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error) {
	pip, err := c.PipelineGet(projectKey, name)
	if err != nil {
		return nil, err
	}

	p := exportentities.NewPipeline(pip, exportWithPermissions)

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
	body, _, err := c.Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) WorkflowExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/export/workflows/%s?format=%s", projectKey, name, exportFormat)
	if exportWithPermissions {
		path += "&withPermissions=true"
	}
	body, _, err := c.Request("GET", path, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}
