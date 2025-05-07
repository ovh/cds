package cdsclient

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectCreate(p *sdk.Project) error {
	_, err := c.PostJSON(context.Background(), "/project", p, nil)
	return err
}

func (c *client) ProjectDelete(key string) error {
	_, err := c.DeleteJSON(context.Background(), "/project/"+key, nil, nil)
	return err
}

func (c *client) ProjectGroupAdd(key, groupName string, permission int, onlyProject bool) error {
	gp := sdk.GroupPermission{
		Group:      sdk.Group{Name: groupName},
		Permission: permission,
	}
	_, err := c.PostJSON(context.Background(), fmt.Sprintf("/project/%s/group?onlyProject=%v", key, onlyProject), gp, nil)
	return err
}

func (c *client) ProjectGroupDelete(key, groupName string) error {
	_, err := c.DeleteJSON(context.Background(), fmt.Sprintf("/project/%s/group/%s", key, groupName), nil, nil)
	return err
}

func (c *client) ProjectGet(key string, mods ...RequestModifier) (*sdk.Project, error) {
	p := &sdk.Project{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key, p, mods...); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ProjectUpdate(key string, project *sdk.Project) error {
	url := fmt.Sprintf("/project/%s", key)
	if _, err := c.PutJSON(context.Background(), url, project, project); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectList(withApplications, withWorkflows bool, filters ...Filter) ([]sdk.Project, error) {
	p := []sdk.Project{}
	path := fmt.Sprintf("/project?application=%v&workflow=%v", withApplications, withWorkflows)

	for _, f := range filters {
		path += fmt.Sprintf("&%s=%s", url.QueryEscape(f.Name), url.QueryEscape(f.Value))
	}

	if _, err := c.GetJSON(context.Background(), path, &p); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ProjectAccess(ctx context.Context, projectKey, sessionID string, itemType sdk.CDNItemType) error {
	url := fmt.Sprintf("/project/%s/type/%s/access", projectKey, itemType)
	if _, err := c.GetJSON(ctx, url, nil, SetHeader(sdk.CDSSessionID, sessionID)); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectV2Access(ctx context.Context, projectKey, sessionID string, itemType sdk.CDNItemType) error {
	url := fmt.Sprintf("/v2/project/%s/type/%s/access", projectKey, itemType)
	if _, err := c.GetJSON(ctx, url, nil, SetHeader(sdk.CDSSessionID, sessionID)); err != nil {
		return err
	}
	return nil
}
