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
