package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) EnvironmentKeysList(key string, envName string) ([]sdk.EnvironmentKey, error) {
	k := []sdk.EnvironmentKey{}
	code, err := c.GetJSON("/project/"+key+"/environment/"+url.QueryEscape(envName)+"/keys", &k)
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

func (c *client) EnvironmentKeyCreate(projectKey string, envName string, keyEnvironment *sdk.EnvironmentKey) error {
	code, err := c.PostJSON("/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/keys", keyEnvironment, keyEnvironment)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) EnvironmentKeysDelete(projectKey string, envName string, keyName string) error {
	_, code, err := c.Request("DELETE", "/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/keys/"+url.QueryEscape(keyName), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}
