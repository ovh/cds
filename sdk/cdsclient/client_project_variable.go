package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVariablesList(key string) ([]sdk.Variable, error) {
	k := []sdk.Variable{}
	code, err := c.GetJSON("/project/"+key+"/variable", &k)
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

func (c *client) ProjectVariableCreate(projectKey string, variable *sdk.Variable) error {
	code, err := c.PostJSON("/project/"+projectKey+"/variable", variable, variable)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ProjectVariableDelete(projectKey string, variable string) error {
	_, code, err := c.Request("DELETE", "/project/"+projectKey+"/variable/"+url.QueryEscape(variable), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ProjectVariableUpdate(projectKey string, variable *sdk.Variable) error {
	_, code, err := c.Request("PUT", "/project/"+projectKey+"/variable/"+url.QueryEscape(variable.Name), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}
