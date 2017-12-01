package cdsclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) EnvironmentCreate(key string, env *sdk.Environment) error {
	code, err := c.PostJSON("/project/"+key+"/environment", env, nil)
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

func (c *client) EnvironmentGroupsImport(projectKey, envName string, content io.Reader, format string, force bool) (sdk.Environment, error) {
	var url string
	var env sdk.Environment
	url = fmt.Sprintf("/project/%s/environment/%s/group/import?format=%s", projectKey, envName, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, code, errReq := c.Request("POST", url, content)
	if code != 200 && errReq == nil {
		return env, fmt.Errorf("HTTP Code %d", code)
	}
	if errReq != nil {
		return env, errReq
	}

	if err := json.Unmarshal(btes, &env); err != nil {
		return env, errReq
	}

	return env, errReq
}
