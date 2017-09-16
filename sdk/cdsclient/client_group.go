package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) GroupCreate(group *sdk.Group) error {
	code, err := c.PostJSON("/group", group, nil)
	if code != 201 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *client) GroupDelete(name string) error {
	code, err := c.DeleteJSON("/group/"+name, nil, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *client) GroupGet(name string, mods ...RequestModifier) (*sdk.Group, error) {
	group := &sdk.Group{}
	code, err := c.GetJSON("/group/"+name, group, mods...)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (c *client) GroupList() ([]sdk.Group, error) {
	groups := []sdk.Group{}
	code, err := c.GetJSON("/group", &groups)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return groups, nil
}
