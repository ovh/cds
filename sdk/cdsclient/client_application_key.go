package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ApplicationKeysList(key string, appName string) ([]sdk.ApplicationKey, error) {
	k := []sdk.ApplicationKey{}
	code, err := c.GetJSON("/project/"+key+"/application/"+appName+"/keys", &k)
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

func (c *client) ApplicationKeyCreate(projectKey string, appName string, keyApplication *sdk.ApplicationKey) error {
	code, err := c.PostJSON("/project/"+projectKey+"/application/"+appName+"/keys", keyApplication, keyApplication)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ApplicationKeysDelete(projectKey string, appName string, keyName string) error {
	_, code, err := c.Request("DELETE", "/project/"+projectKey+"/application/"+appName+"/keys/"+url.QueryEscape(keyName), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}
