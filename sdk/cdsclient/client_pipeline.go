package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) PipelineGet(projectKey, name string) (*sdk.Pipeline, error) {
	pipeline := sdk.Pipeline{}
	if _, err := c.GetJSON(context.Background(), "/project/"+projectKey+"/pipeline/"+name, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

func (c *client) PipelineCreate(projectKey string, pip *sdk.Pipeline) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/pipeline", pip, nil)
	return err
}

func (c *client) PipelineDelete(projectKey, name string) error {
	_, err := c.DeleteJSON(context.Background(), "/project/"+projectKey+"/pipeline/"+url.QueryEscape(name), nil, nil)
	return err
}

func (c *client) PipelineList(projectKey string) ([]sdk.Pipeline, error) {
	pipelines := []sdk.Pipeline{}
	if _, err := c.GetJSON(context.Background(), "/project/"+projectKey+"/pipeline", &pipelines); err != nil {
		return nil, err
	}
	return pipelines, nil
}
