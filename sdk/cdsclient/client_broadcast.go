package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) BroadcastDelete(id string) error {
	_, err := c.DeleteJSON("/broadcast/"+id, nil)
	return err
}

func (c *client) BroadcastGet(id string) (*sdk.Broadcast, error) {
	bc := &sdk.Broadcast{}
	if _, err := c.GetJSON("/broadcast/"+id, bc); err != nil {
		return nil, err
	}
	return bc, nil
}

func (c *client) Broadcasts() ([]sdk.Broadcast, error) {
	bcs := []sdk.Broadcast{}
	if _, err := c.GetJSON("/broadcast", &bcs); err != nil {
		return nil, err
	}
	return bcs, nil
}

func (c *client) BroadcastsByLevel(level string) ([]sdk.Broadcast, error) {
	bcs := []sdk.Broadcast{}
	if _, err := c.GetJSON("/broadcast/"+level, &bcs); err != nil {
		return nil, err
	}
	return bcs, nil
}
