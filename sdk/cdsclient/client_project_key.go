package cdsclient

import (
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectKeysList(key string) ([]sdk.ProjectKey, error) {
	k := []sdk.ProjectKey{}
	if _, err := c.GetJSON("/project/"+key+"/keys", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) ProjectKeyCreate(projectKey string, keyProject *sdk.ProjectKey) error {
	_, err := c.PostJSON("/project/"+projectKey+"/keys", keyProject, keyProject)
	return err
}

func (c *client) ProjectKeysDelete(projectKey string, keyName string) error {
	_, _, err := c.Request("DELETE", "/project/"+projectKey+"/keys/"+url.QueryEscape(keyName), nil)
	return err
}
