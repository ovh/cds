package cdsclient

import (
	"fmt"

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
	code, err := c.DeleteJSON("/action/"+actionName, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ActionGet(actionName string, mods ...RequestModifier) (*sdk.Action, error) {
	action := &sdk.Action{}
	code, err := c.GetJSON("/action/"+actionName, action, mods...)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (c *client) ActionList() ([]sdk.Action, error) {
	actions := []sdk.Action{}
	code, err := c.GetJSON("/action", &actions)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return actions, nil
}
