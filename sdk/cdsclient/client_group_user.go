package cdsclient

import (
	"net/url"
)

func (c *client) GroupUserAdd(groupname string, users []string) error {
	_, err := c.PostJSON("/group/"+url.QueryEscape(groupname)+"/user", users, nil)
	return err
}

func (c *client) GroupUserRemove(groupname, username string) error {
	_, _, _, err := c.Request("DELETE", "/group/"+url.QueryEscape(groupname)+"/user/"+url.QueryEscape(username), nil)
	return err
}

func (c *client) GroupUserAdminSet(groupname string, username string) error {
	_, err := c.PostJSON("/group/"+url.QueryEscape(groupname)+"/user/"+url.QueryEscape(username)+"/admin", nil, nil)
	return err
}

func (c *client) GroupUserAdminRemove(groupname, username string) error {
	_, _, _, err := c.Request("DELETE", "/group/"+url.QueryEscape(groupname)+"/user/"+url.QueryEscape(username)+"/admin", nil)
	return err
}
