package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) GroupCreate(group *sdk.Group) error {
	code, err := c.PostJSON(context.Background(), "/group", group, nil)
	if code != 201 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) GroupDelete(name string) error {
	_, err := c.DeleteJSON(context.Background(), "/group/"+name, nil, nil)
	return err
}

func (c *client) GroupGet(name string, mods ...RequestModifier) (*sdk.Group, error) {
	group := &sdk.Group{}
	if _, err := c.GetJSON(context.Background(), "/group/"+name, group, mods...); err != nil {
		return nil, err
	}
	return group, nil
}

func (c *client) GroupList() ([]sdk.Group, error) {
	groups := []sdk.Group{}
	if _, err := c.GetJSON(context.Background(), "/group", &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

func (c *client) GroupRename(oldGroupname, newGroupname string) error {
	group := &sdk.Group{}
	if _, err := c.GetJSON(context.Background(), "/group/"+oldGroupname, group); err != nil {
		return err
	}

	group.Name = newGroupname
	code, err := c.PutJSON(context.Background(), "/group/"+oldGroupname, group, nil)
	if code > 400 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}
