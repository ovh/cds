package cdsclient

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
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

func (c *client) TemplateExecute(projectKey string, id int64, req sdk.WorkflowTemplateRequest) (*tar.Reader, error) {
	url := fmt.Sprintf("/project/%s/template/%d/execute", projectKey, id)

	bs, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	body, _, _, err := c.Request(context.Background(), "POST", url, bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}

	// Open the tar archive for reading.
	r := bytes.NewReader(body)
	tr := tar.NewReader(r)
	return tr, nil
}

func (c *client) TemplateUpdate(projectKey, workflowName string, req sdk.WorkflowTemplateRequest) ([]string, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s/update", projectKey, workflowName)

	messages := []string{}
	if _, err := c.PostJSON(context.Background(), url, req, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}
