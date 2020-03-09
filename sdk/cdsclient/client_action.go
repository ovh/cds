package cdsclient

import (
	"context"
	"fmt"
	"io"

	"github.com/ovh/cds/sdk"
)

func (c *client) Requirements() ([]sdk.Requirement, error) {
	var req []sdk.Requirement
	if _, err := c.GetJSON(context.Background(), "/action/requirement", &req); err != nil {
		return nil, err
	}
	return req, nil
}

func (c *client) ActionDelete(groupName, name string) error {
	path := fmt.Sprintf("/action/%s/%s", groupName, name)
	_, err := c.DeleteJSON(context.Background(), path, nil)
	return err
}

func (c *client) ActionGet(groupName, name string, mods ...RequestModifier) (*sdk.Action, error) {
	var a sdk.Action

	path := fmt.Sprintf("/action/%s/%s", groupName, name)
	if _, err := c.GetJSON(context.Background(), path, &a, mods...); err != nil {
		return nil, err
	}

	return &a, nil
}

func (c *client) ActionUsage(groupName, name string, mods ...RequestModifier) (*sdk.ActionUsages, error) {
	var a sdk.ActionUsages

	path := fmt.Sprintf("/action/%s/%s/usage", groupName, name)
	if _, err := c.GetJSON(context.Background(), path, &a, mods...); err != nil {
		return nil, err
	}

	return &a, nil
}

func (c *client) ActionList() ([]sdk.Action, error) {
	actions := []sdk.Action{}
	if _, err := c.GetJSON(context.Background(), "/action", &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

func (c *client) ActionImport(content io.Reader) error {
	url := "/action/import"
	_, _, code, err := c.Request(context.Background(), "POST", url, content)
	if err != nil {
		return err
	}
	if code > 400 {
		return fmt.Errorf("HTTP Code %d", code)
	}
	return nil
}

func (c *client) ActionExport(groupName, name string, mods ...RequestModifier) ([]byte, error) {
	path := fmt.Sprintf("/action/%s/%s/export", groupName, name)
	body, _, _, err := c.Request(context.Background(), "GET", path, nil, mods...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (c *client) ActionBuiltinList() ([]sdk.Action, error) {
	actions := []sdk.Action{}
	if _, err := c.GetJSON(context.Background(), "/actionBuiltin", &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

func (c *client) ActionBuiltinGet(name string, mods ...RequestModifier) (*sdk.Action, error) {
	var a sdk.Action

	path := fmt.Sprintf("/actionBuiltin/%s", name)
	if _, err := c.GetJSON(context.Background(), path, &a, mods...); err != nil {
		return nil, err
	}

	return &a, nil
}
