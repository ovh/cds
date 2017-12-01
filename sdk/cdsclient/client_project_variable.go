package cdsclient

import (
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVariablesList(key string) ([]sdk.Variable, error) {
	k := []sdk.Variable{}
	if _, err := c.GetJSON("/project/"+key+"/variable", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) ProjectVariableCreate(projectKey string, variable *sdk.Variable) error {
	_, err := c.PostJSON("/project/"+projectKey+"/variable/"+url.QueryEscape(variable.Name), variable, variable)
	return err
}

func (c *client) ProjectVariableDelete(projectKey string, varName string) error {
	_, _, err := c.Request("DELETE", "/project/"+projectKey+"/variable/"+url.QueryEscape(varName), nil)
	return err
}

func (c *client) ProjectVariableUpdate(projectKey string, variable *sdk.Variable) error {
	_, err := c.PutJSON("/project/"+projectKey+"/variable/"+url.QueryEscape(variable.Name), variable, variable, nil)
	return err
}

func (c *client) ProjectVariableGet(projectKey string, varName string) (*sdk.Variable, error) {
	variable := &sdk.Variable{}
	if _, err := c.GetJSON("/project/"+projectKey+"/variable/"+url.QueryEscape(varName), variable, nil); err != nil {
		return nil, err
	}
	return variable, nil
}
