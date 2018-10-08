package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ApplicationVariablesList(key string, appName string) ([]sdk.Variable, error) {
	k := []sdk.Variable{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key+"/application/"+appName+"/variable", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) ApplicationVariableCreate(projectKey string, appName string, variable *sdk.Variable) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(variable.Name), variable, variable)
	return err
}

func (c *client) ApplicationVariableDelete(projectKey string, appName string, varName string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(varName), nil)
	return err
}

func (c *client) ApplicationVariableUpdate(projectKey string, appName string, variable *sdk.Variable) error {
	_, err := c.PutJSON(context.Background(), "/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(variable.Name), variable, variable, nil)
	return err
}

func (c *client) ApplicationVariableGet(projectKey string, appName string, varName string) (*sdk.Variable, error) {
	variable := &sdk.Variable{}
	if _, err := c.GetJSON(context.Background(), "/project/"+projectKey+"/application/"+appName+"/variable/"+url.QueryEscape(varName), variable, nil); err != nil {
		return nil, err
	}
	return variable, nil
}
