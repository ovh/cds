package cdsclient

import (
	"context"
	"fmt"
)

func (c *client) WorkflowV3Get(projectKey, workflowName string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/project/%s/workflowv3/%s", projectKey, workflowName)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}
