package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) EnvironmentVariablesList(key string, envName string) ([]sdk.Variable, error) {
	k := []sdk.Variable{}
	code, err := c.GetJSON("/project/"+key+"/environment/"+url.QueryEscape(envName)+"/variable", &k)
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

func (c *client) EnvironmentVariableCreate(projectKey string, envName string, variable *sdk.Variable) error {
	code, err := c.PostJSON("/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/variable/"+url.QueryEscape(variable.Name), variable, variable)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) EnvironmentVariableDelete(projectKey string, envName string, varName string) error {
	_, code, err := c.Request("DELETE", "/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/variable/"+url.QueryEscape(varName), nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) EnvironmentVariableUpdate(projectKey string, envName string, variable *sdk.Variable) error {
	code, err := c.PutJSON("/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/variable/"+url.QueryEscape(variable.Name), variable, variable, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	return err
}

func (c *client) EnvironmentVariableGet(projectKey string, envName string, varName string) (*sdk.Variable, error) {
	variable := &sdk.Variable{}
	code, err := c.GetJSON("/project/"+projectKey+"/environment/"+envName+"/variable/"+url.QueryEscape(varName), variable, nil)
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
