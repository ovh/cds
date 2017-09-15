package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

/*

"/group", r.GET(api.getGroupsHandler), r.POST(api.addGroupHandler))
"/group/public", r.GET(api.getPublicGroupsHandler))
"/group/{permGroupName}", r.GET(api.getGroupHandler), r.PUT(api.updateGroupHandler), r.DELETE(api.deleteGroupHandler))
"/group/{permGroupName}/user", r.POST(api.addUserInGroupHandler))
"/group/{permGroupName}/user/{user}", r.DELETE(api.removeUserFromGroupHandler))
"/group/{permGroupName}/user/{user}/admin", r.POST(api.setUserGroupAdminHandler), r.DELETE(api.removeUserGroupAdminHandler))
"/group/{permGroupName}/token/{expiration}", r.POST(api.generateTokenHandler))

*/

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
