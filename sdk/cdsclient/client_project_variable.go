package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVariablesList(key string) ([]sdk.Variable, error) {
	k := []sdk.Variable{}
	if _, err := c.GetJSON(context.Background(), "/project/"+key+"/variable", &k); err != nil {
		return nil, err
	}
	return k, nil
}

func (c *client) ProjectVariableCreate(projectKey string, variable *sdk.Variable) error {
	_, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/variable/"+url.QueryEscape(variable.Name), variable, variable)
	return err
}

func (c *client) ProjectVariableDelete(projectKey string, varName string) error {
	_, _, _, err := c.Request(context.Background(), "DELETE", "/project/"+projectKey+"/variable/"+url.QueryEscape(varName), nil)
	return err
}

func (c *client) ProjectVariableUpdate(projectKey string, variable *sdk.Variable) error {
	_, err := c.PutJSON(context.Background(), "/project/"+projectKey+"/variable/"+url.QueryEscape(variable.Name), variable, variable, nil)
	return err
}

func (c *client) ProjectVariableGet(projectKey string, varName string) (*sdk.Variable, error) {
	variable := &sdk.Variable{}
	if _, err := c.GetJSON(context.Background(), "/project/"+projectKey+"/variable/"+url.QueryEscape(varName), variable, nil); err != nil {
		return nil, err
	}
	return variable, nil
}

func (c *client) VariableEncrypt(projectKey string, varName string, content string) (*sdk.Variable, error) {
	variable := &sdk.Variable{
		Name:  varName,
		Value: content,
		Type:  sdk.SecretVariable,
	}
	if _, err := c.PostJSON(context.Background(), "/project/"+projectKey+"/encrypt", variable, variable); err != nil {
		return nil, err
	}
	return variable, nil
}

func (c *client) VariableListEncrypt(projectKey string) ([]sdk.Secret, error) {
	secrets := []sdk.Secret{}
	if _, err := c.GetJSON(context.Background(), "/project/"+projectKey+"/encrypt/list", &secrets, nil); err != nil {
		return nil, err
	}
	return secrets, nil
}

func (c *client) VariableEncryptDelete(projectKey, name string) error {
	if _, err := c.DeleteJSON(context.Background(), "/project/"+projectKey+"/encrypt?name="+url.QueryEscape(name), nil, nil); err != nil {
		return err
	}
	return nil
}
