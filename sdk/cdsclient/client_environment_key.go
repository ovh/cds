package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) EnvironmentKeysList(key string, envName string) ([]sdk.EnvironmentKey, error) {
	k := []sdk.EnvironmentKey{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key+"/environment/"+url.QueryEscape(envName)+"/keys", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) EnvironmentKeyCreate(projectKey string, envName string, keyEnvironment *sdk.EnvironmentKey) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/keys", keyEnvironment, keyEnvironment)
	return err
}

func (c *client) EnvironmentKeysDelete(projectKey string, envName string, keyName string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/keys/"+url.QueryEscape(keyName), nil)
	return err
}
