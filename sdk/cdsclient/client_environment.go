package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) EnvironmentCreate(key string, env *sdk.Environment) error {
	code, err := c.PostJSON("/project/"+key+"/environment", env, nil)
	if code != 201 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *client) EnvironmentDelete(key string, envName string) error {
	code, err := c.DeleteJSON("/project/"+key+"/environment/"+url.QueryEscape(envName), nil, nil)
	if code != 200 {
		if err == nil {
			return fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (c *client) EnvironmentGet(key string, envName string, mods ...RequestModifier) (*sdk.Environment, error) {
	env := &sdk.Environment{}
	code, err := c.GetJSON("/project/"+key+"/environment/"+url.QueryEscape(envName), env)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return env, nil
}

func (c *client) EnvironmentList(key string) ([]sdk.Environment, error) {
	envs := []sdk.Environment{}
	code, err := c.GetJSON("/project/"+key+"/environment", &envs)
	if code != 200 {
		if err == nil {
			return nil, fmt.Errorf("HTTP Code %d", code)
		}
	}
	if err != nil {
		return nil, err
	}
	return envs, nil
}
