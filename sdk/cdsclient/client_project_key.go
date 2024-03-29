package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectKeysList(key string) ([]sdk.ProjectKey, error) {
	k := []sdk.ProjectKey{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key+"/keys", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) ProjectKeyCreate(projectKey string, keyProject *sdk.ProjectKey) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/keys", keyProject, keyProject)
	return err
}

func (c *client) ProjectKeysDelete(projectKey string, keyName string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/project/"+projectKey+"/keys/"+url.QueryEscape(keyName), nil)
	return err
}

func (c *client) ProjectKeysDisable(projectKey string, keyProjectName string) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/keys/"+url.QueryEscape(keyProjectName)+"/disable", nil, nil)
	return err
}

func (c *client) ProjectKeysEnable(projectKey string, keyProjectName string) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/keys/"+url.QueryEscape(keyProjectName)+"/enable", nil, nil)
	return err
}
