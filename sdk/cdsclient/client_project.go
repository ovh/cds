package cdsclient

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectCreate(p *sdk.Project, groupName string) error {
	if groupName != "" {
		gr := sdk.Group{}
		code, err := c.GetJSON("/group/"+groupName, &gr)
		if code != 200 {
			if err == nil {
				return fmt.Errorf("Error on group %s : HTTP Code %d", groupName, code)
			}
		}
		if err != nil {
			return err
		}
		p.ProjectGroups = []sdk.GroupPermission{
			sdk.GroupPermission{
				Group:      gr,
				Permission: permission.PermissionReadWriteExecute,
			},
		}
	}

	code, err := c.PostJSON("/project", p, nil)
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

func (c *client) ProjectDelete(key string) error {
	code, err := c.DeleteJSON("/project/"+key, nil, nil)
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

func (c *client) ProjectGet(key string, mods ...RequestModifier) (*sdk.Project, error) {
	p := &sdk.Project{}
	code, err := c.GetJSON("/project/"+key, p, mods...)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ProjectList() ([]sdk.Project, error) {
	p := []sdk.Project{}
	code, err := c.GetJSON("/project", &p)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ProjectGroupsImport(projectKey string, content io.Reader, format string, force bool) (sdk.Project, error) {
	var url string
	var proj sdk.Project
	url = fmt.Sprintf("/project/%s/group/import?format=%s", projectKey, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, code, errReq := c.Request("POST", url, content)
	if code != 200 && errReq == nil {
		return proj, fmt.Errorf("HTTP Code %d", code)
	}
	if errReq != nil {
		return proj, errReq
	}

	if err := json.Unmarshal(btes, &proj); err != nil {
		return proj, errReq
	}

	return proj, errReq
}
