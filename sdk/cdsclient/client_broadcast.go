package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) BroadcastDelete(id string) error {
	_, err := c.DeleteJSON("/broadcast/"+id, nil)
	return err
}

func (c *client) BroadcastCreate(broadcast *sdk.Broadcast) error {
	code, err := c.PostJSON("/broadcast", broadcast, nil)
	if code != 201 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
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
