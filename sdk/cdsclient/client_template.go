package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) TemplateGet(id int64) (*sdk.WorkflowTemplate, error) {
	url := fmt.Sprintf("/template/%d", id)

	var wt sdk.WorkflowTemplate
	if _, err := c.GetJSON(context.Background(), url, &wt); err != nil {
		return nil, err
	}

	return &wt, nil
}

func (c *client) TemplateExecute(projectKey string, id int64, req sdk.WorkflowTemplateRequest) ([]string, error) {
	url := fmt.Sprintf("/project/%s/template/%d/execute", projectKey, id)

	messages := []string{}
	if _, err := c.PostJSON(context.Background(), url, req, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}
