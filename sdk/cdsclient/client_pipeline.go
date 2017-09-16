package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (c *client) PipelineGet(projectKey, name string) (*sdk.Pipeline, error) {
	pipeline := sdk.Pipeline{}
	code, err := c.GetJSON("/project/"+projectKey+"/pipeline/"+name, &pipeline)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

func (c *client) PipelineExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error) {
	pip, err := c.PipelineGet(projectKey, name)
	if err != nil {
		return nil, err
	}

	p := exportentities.NewPipeline(pip)

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

func (c *client) PipelineList(projectKey string) ([]sdk.Pipeline, error) {
	pipelines := []sdk.Pipeline{}
	code, err := c.GetJSON("/project/"+projectKey+"/pipeline", &pipelines)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return pipelines, nil
}
