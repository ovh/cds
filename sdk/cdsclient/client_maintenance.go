package cdsclient

import (
	"context"
	"fmt"
)

func (c *client) Maintenance(enable bool) error {
	_, err := c.PostJSON(context.Background(), fmt.Sprintf("/admin/maintenance?enable=%v", enable), nil, nil)
	return err
}
