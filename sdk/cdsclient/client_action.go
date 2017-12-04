package cdsclient

import (
	"github.com/ovh/cds/sdk"
)

func (c *client) Requirements() ([]sdk.Requirement, error) {
	var req []sdk.Requirement
	if _, err := c.GetJSON("/action/requirement", &req); err != nil {
		return nil, err
	}
	return req, nil
}

func (c *client) ActionDelete(actionName string) error {
	_, err := c.DeleteJSON("/action/"+actionName, nil)
	return err
}

func (c *client) ActionGet(actionName string, mods ...RequestModifier) (*sdk.Action, error) {
	action := &sdk.Action{}
	if _, err := c.GetJSON("/action/"+actionName, action, mods...); err != nil {
		return nil, err
	}
	return action, nil
}

func (c *client) ActionList() ([]sdk.Action, error) {
	actions := []sdk.Action{}
	if _, err := c.GetJSON("/action", &actions); err != nil {
		return nil, err
	}
	return actions, nil
}
