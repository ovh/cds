package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) Broadcasts() ([]sdk.Broadcast, error) {
	srvs := []sdk.Broadcast{}
	if _, err := c.GetJSON("/broadcast", &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}

func (c *client) BroadcastsByLevel(s string) ([]sdk.Broadcast, error) {
	srvs := []sdk.Broadcast{}
	if _, err := c.GetJSON("/broadcast/"+s, &srvs); err != nil {
		return nil, err
	}
	return srvs, nil
}
