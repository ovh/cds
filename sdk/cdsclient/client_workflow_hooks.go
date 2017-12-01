package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) WorkflowAllHooksList() ([]sdk.WorkflowNodeHook, error) {
	url := fmt.Sprintf("/workflow/hook")
	w := []sdk.WorkflowNodeHook{}
	if _, err := c.GetJSON(url, &w); err != nil {
		return nil, err
	}
	return w, nil
}
