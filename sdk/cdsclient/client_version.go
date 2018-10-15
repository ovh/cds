package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) Version() (*sdk.Version, error) {
	v := &sdk.Version{}
	if _, err := c.GetJSON(context.Background(), "/mon/version", v); err != nil {
		return nil, err
	}
	return v, nil
}
