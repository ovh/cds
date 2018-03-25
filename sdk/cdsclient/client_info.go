package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) Infos() ([]sdk.Info, error) {
	srvs := []sdk.Info{}
	if _, err := c.GetJSON("/info", &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}

func (c *client) InfosByLevel(s string) ([]sdk.Info, error) {
	srvs := []sdk.Info{}
	if _, err := c.GetJSON("/info/"+s, &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}
