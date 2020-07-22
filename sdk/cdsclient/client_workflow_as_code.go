package cdsclient

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkflowTransformAsCode(projectKey, workflowName, branch, message string) (*sdk.Operation, error) {
	var ope sdk.Operation
	path := fmt.Sprintf("/project/%s/workflows/%s/ascode", projectKey, workflowName)
	if _, err := c.PostJSON(context.Background(), path, nil, &ope, func(r *http.Request) {
		q := r.URL.Query()
		q.Set("migrate", "true")
		q.Set("branch", branch)
		q.Set("message", message)
		r.URL.RawQuery = q.Encode()
	}); err != nil {
		return nil, err
	}
	return &ope, nil
}

func (c client) WorkflowTransformAsCodeFollow(projectKey, workflowName, opeUUID string) (*sdk.Operation, error) {
	var ope sdk.Operation
	path := fmt.Sprintf("/project/%s/workflows/%s/ascode/%s", projectKey, workflowName, opeUUID)
	if _, err := c.GetJSON(context.Background(), path, &ope); err != nil {
		return nil, err
	}
	return &ope, nil
}

func (c *client) WorkflowAsCodeStart(projectKey string, repoURL string, repoStrategy sdk.RepositoryStrategy) (*sdk.Operation, error) {
	ope := new(sdk.Operation)
	ope.URL = repoURL
	ope.RepositoryStrategy = repoStrategy

	path := fmt.Sprintf("/import/%s", projectKey)
	if _, err := c.PostJSON(context.Background(), path, ope, ope); err != nil {
		return nil, err
	}

	return ope, nil
}

func (c *client) WorkflowAsCodeInfo(projectKey string, operationID string) (*sdk.Operation, error) {
	ope := new(sdk.Operation)
	path := fmt.Sprintf("/import/%s/%s", projectKey, operationID)
	if _, err := c.GetJSON(context.Background(), path, ope); err != nil {
		return nil, err
	}
	return ope, nil
}

func (c *client) WorkflowAsCodePerform(projectKey string, operationID string) ([]string, error) {
	messages := []string{}
	path := fmt.Sprintf("/import/%s/%s/perform", projectKey, operationID)
	if _, err := c.PostJSON(context.Background(), path, nil, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}
