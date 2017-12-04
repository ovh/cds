package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) Version() (*sdk.Version, error) {
	v := &sdk.Version{}
	if _, err := c.GetJSON("/mon/version", v); err != nil {
		return nil, err
	}
	return v, nil
}
