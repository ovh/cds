package cdsclient

import (
	"context"
	"fmt"
)

func (c *client) Maintenance(enable bool, hooks bool) error {
	_, err := c.PostJSON(context.Background(), fmt.Sprintf("/admin/maintenance?enable=%v&withHook=%v", enable, hooks), nil, nil)
	return err
}
