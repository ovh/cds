package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkflowTransformAsCode(projectKey, workflowName string) (*sdk.Operation, error) {
	ope := new(sdk.Operation)
	path := fmt.Sprintf("/project/%s/workflows/%s/ascode", projectKey, workflowName)
	if _, err := c.PostJSON(context.Background(), path, nil, &ope); err != nil {
		return nil, err
	}
	return ope, nil
}

func (c client) WorkflowTransformAsCodeFollow(projectKey, workflowName string, ope *sdk.Operation) error {
	path := fmt.Sprintf("/project/%s/workflows/%s/ascode/%s", projectKey, workflowName, ope.UUID)
	if _, err := c.GetJSON(context.Background(), path, ope); err != nil {
		return err
	}
	return nil
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
