package cdsclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/ovh/cds/sdk"
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

func (c *client) PipelineCreate(projectKey string, pip *sdk.Pipeline) error {
	code, err := c.PostJSON("/project/"+projectKey+"/pipeline", pip, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) PipelineDelete(projectKey, name string) error {
	code, err := c.DeleteJSON("/project/"+projectKey+"/pipeline/"+url.QueryEscape(name), nil, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
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

func (c *client) PipelineGroupsImport(projectKey, pipelineName string, content io.Reader, format string, force bool) (sdk.Pipeline, error) {
	var url string
	var pip sdk.Pipeline
	url = fmt.Sprintf("/project/%s/pipeline/%s/group/import?format=%s", projectKey, pipelineName, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, code, errReq := c.Request("POST", url, content)
	if code != 200 && errReq == nil {
		return pip, fmt.Errorf("HTTP Code %d", code)
	}
	if errReq != nil {
		return pip, errReq
	}

	if err := json.Unmarshal(btes, &pip); err != nil {
		return pip, errReq
	}

	return pip, errReq
}
