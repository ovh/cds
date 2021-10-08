package cdsclient

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

func (c *client) GroupCreate(group *sdk.Group) error {
	code, err := c.PostJSON(context.Background(), "/group", group, nil)
	if code != 201 {
		if err == nil {
			return newAPIError(fmt.Errorf("HTTP Code %d", code))
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

func (c *client) GroupExport(name string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/group/%s/export", name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) GroupImport(content io.Reader, mods ...RequestModifier) ([]byte, error) {
	btes, _, _, err := c.Request(context.Background(), "POST", "/group/import", content, mods...)
	if err != nil {
		return nil, err
	}
	return btes, nil
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
			return newAPIError(fmt.Errorf("HTTP Code %d", code))
		}
	}
	return err
}
