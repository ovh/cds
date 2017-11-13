package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ApplicationVariablesList(key string, appName string) ([]sdk.Variable, error) {
	k := []sdk.Variable{}
	code, err := c.GetJSON("/project/"+key+"/application/"+appName+"/variable", &k)
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

func (c *client) ApplicationVariableCreate(projectKey string, appName string, variable *sdk.Variable) error {
	code, err := c.PostJSON("/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(variable.Name), variable, variable)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ApplicationVariableDelete(projectKey string, appName string, varName string) error {
	_, code, err := c.Request("DELETE", "/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(varName), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ApplicationVariableUpdate(projectKey string, appName string, variable *sdk.Variable) error {
	code, err := c.PutJSON("/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(variable.Name), variable, variable, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) ApplicationVariableGet(projectKey string, appName string, varName string) (*sdk.Variable, error) {
	variable := &sdk.Variable{}
	code, err := c.GetJSON("/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(varName), variable, nil)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return variable, nil
}
