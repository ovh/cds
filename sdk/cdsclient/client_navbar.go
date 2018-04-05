package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) Navbar() ([]sdk.NavbarProjectData, error) {
	navbar := []sdk.NavbarProjectData{}
	if _, err := c.GetJSON("/navbar", &navbar); err != nil {
		return nil, err
	}
	return navbar, nil
}
