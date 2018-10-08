package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ApplicationKeysList(key string, appName string) ([]sdk.ApplicationKey, error) {
	k := []sdk.ApplicationKey{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key+"/application/"+appName+"/keys", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) ApplicationKeyCreate(projectKey string, appName string, keyApplication *sdk.ApplicationKey) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/application/"+appName+"/keys", keyApplication, keyApplication)
	return err
}

func (c *client) ApplicationKeysDelete(projectKey string, appName string, keyName string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/project/"+projectKey+"/application/"+appName+"/keys/"+url.QueryEscape(keyName), nil)
	return err
}
