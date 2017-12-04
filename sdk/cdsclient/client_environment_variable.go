package cdsclient

import (
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) EnvironmentVariablesList(key string, envName string) ([]sdk.Variable, error) {
	k := []sdk.Variable{}
	if _, err := c.GetJSON("/project/"+key+"/environment/"+url.QueryEscape(envName)+"/variable", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) EnvironmentVariableCreate(projectKey string, envName string, variable *sdk.Variable) error {
	_, err := c.PostJSON("/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/variable/"+url.QueryEscape(variable.Name), variable, variable)
	return err
}

func (c *client) EnvironmentVariableDelete(projectKey string, envName string, varName string) error {
	_, _, err := c.Request("DELETE", "/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/variable/"+url.QueryEscape(varName), nil)
	return err
}

func (c *client) EnvironmentVariableUpdate(projectKey string, envName string, variable *sdk.Variable) error {
	_, err := c.PutJSON("/project/"+projectKey+"/environment/"+url.QueryEscape(envName)+"/variable/"+url.QueryEscape(variable.Name), variable, variable, nil)
	return err
}

func (c *client) EnvironmentVariableGet(projectKey string, envName string, varName string) (*sdk.Variable, error) {
	variable := &sdk.Variable{}
	if _, err := c.GetJSON("/project/"+projectKey+"/environment/"+envName+"/variable/"+url.QueryEscape(varName), variable, nil); err != nil {
		return nil, err
	}
	return variable, nil
}
