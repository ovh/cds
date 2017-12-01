package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) Version() (*sdk.Version, error) {
	v := &sdk.Version{}

	code, err := c.GetJSON("/mon/version", v)
	if err != nil {
		return nil, err
	}
	if code >= 400 {
		return nil, fmt.Errorf("Error %d", code)
	}
	return v, err
}
