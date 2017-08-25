package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectKeysList(key string) ([]sdk.ProjectKey, error) {
	k := []sdk.ProjectKey{}
	code, err := c.GetJSON("/project/"+key+"/keys", &k)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) ProjectKeyCreate(projectKey string, keyProject *sdk.ProjectKey) error {
	code, err := c.PostJSON("/project/"+projectKey+"/keys", keyProject, keyProject)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ProjectKeysDelete(projectKey string, keyName string) error {
	_, code, err := c.Request("DELETE", "/project/"+projectKey+"/keys/"+url.QueryEscape(keyName), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}
