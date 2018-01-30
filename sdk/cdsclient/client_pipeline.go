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
	if _, err := c.GetJSON("/project/"+projectKey+"/pipeline/"+name, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

func (c *client) PipelineCreate(projectKey string, pip *sdk.Pipeline) error {
	_, err := c.PostJSON("/project/"+projectKey+"/pipeline", pip, nil)
	return err
}

func (c *client) PipelineDelete(projectKey, name string) error {
	_, err := c.DeleteJSON("/project/"+projectKey+"/pipeline/"+url.QueryEscape(name), nil, nil)
	return err
}

func (c *client) PipelineList(projectKey string) ([]sdk.Pipeline, error) {
	pipelines := []sdk.Pipeline{}
	if _, err := c.GetJSON("/project/"+projectKey+"/pipeline", &pipelines); err != nil {
		return nil, err
	}
	return pipelines, nil
}

func (c *client) PipelineGroupsImport(projectKey, pipelineName string, content io.Reader, format string, force bool) (sdk.Pipeline, error) {
	var pip sdk.Pipeline
	url := fmt.Sprintf("/project/%s/pipeline/%s/group/import?format=%s", projectKey, pipelineName, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, _, _, errReq := c.Request("POST", url, content)
	if errReq != nil {
		return pip, errReq
	}

	if err := json.Unmarshal(btes, &pip); err != nil {
		return pip, err
	}

	return pip, nil
}
