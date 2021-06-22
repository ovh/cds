package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkflowAllHooksList() ([]sdk.NodeHook, error) {
	url := fmt.Sprintf("/workflow/hook")
	var w []sdk.NodeHook
	if _, err := c.GetJSON(context.Background(), url, &w); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) WorkflowAllHooksExecutions() ([]string, error) {
	url := fmt.Sprintf("/workflow/hook/executions")
	var res []string
	if _, err := c.GetJSON(context.Background(), url, &res); err != nil {
		return nil, err
	}
	return res, nil
}
