package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) GroupMemberAdd(groupName string, member *sdk.GroupMember) (sdk.Group, error) {
	var result sdk.Group
	_, err := c.PostJSON(context.Background(), "/group/"+url.QueryEscape(groupName)+"/user", member, &result)
	return result, err
}

func (c *client) GroupMemberEdit(groupName string, member *sdk.GroupMember) (sdk.Group, error) {
	var result sdk.Group
	_, err := c.PutJSON(context.Background(), "/group/"+url.QueryEscape(groupName)+"/user/"+url.QueryEscape(member.Username), member, &result)
	return result, err
}

func (c *client) GroupMemberRemove(groupName, username string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/group/"+url.QueryEscape(groupName)+"/user/"+url.QueryEscape(username), nil)
	return err
}
