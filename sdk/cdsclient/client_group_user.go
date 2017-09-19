package cdsclient

import (
	"fmt"
	"net/url"
)

func (c *client) GroupUserAdd(groupname string, users []string) error {
	code, err := c.PostJSON("/group/"+url.QueryEscape(groupname)+"/user", users, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) GroupUserRemove(groupname, username string) error {
	_, code, err := c.Request("DELETE", "/group/"+url.QueryEscape(groupname)+"/user/"+url.QueryEscape(username), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) GroupUserAdminSet(groupname string, username string) error {
	code, err := c.PostJSON("/group/"+url.QueryEscape(groupname)+"/user/"+url.QueryEscape(username)+"/admin", nil, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) GroupUserAdminRemove(groupname, username string) error {
	_, code, err := c.Request("DELETE", "/group/"+url.QueryEscape(groupname)+"/user/"+url.QueryEscape(username)+"/admin", nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}
