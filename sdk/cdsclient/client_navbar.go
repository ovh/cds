package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) Navbar() (*sdk.NavbarData, error) {
	navbar := sdk.NavbarData{}
	if _, err := c.GetJSON("/navbar", &navbar); err != nil {
		return nil, err
	}
	return &navbar, nil
}
