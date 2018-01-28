package cdsclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) EnvironmentCreate(key string, env *sdk.Environment) error {
	if _, err := c.PostJSON("/project/"+key+"/environment", env, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) EnvironmentDelete(key string, envName string) error {
	if _, err := c.DeleteJSON("/project/"+key+"/environment/"+url.QueryEscape(envName), nil, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) EnvironmentGet(key string, envName string, mods ...RequestModifier) (*sdk.Environment, error) {
	env := &sdk.Environment{}
	if _, err := c.GetJSON("/project/"+key+"/environment/"+url.QueryEscape(envName), env); err != nil {
		return nil, err
	}
	return env, nil
}

func (c *client) EnvironmentList(key string) ([]sdk.Environment, error) {
	envs := []sdk.Environment{}
	if _, err := c.GetJSON("/project/"+key+"/environment", &envs); err != nil {
		return nil, err
	}
	return envs, nil
}

func (c *client) EnvironmentGroupsImport(projectKey, envName string, content io.Reader, format string, force bool) (sdk.Environment, error) {
	var env sdk.Environment
	url := fmt.Sprintf("/project/%s/environment/%s/group/import?format=%s", projectKey, envName, format)

	if force {
		url += "&forceUpdate=true"
	}

	btes, _, errReq := c.Request("POST", url, content)
	if errReq != nil {
		return env, errReq
	}

	if err := json.Unmarshal(btes, &env); err != nil {
		return env, err
	}

	return env, errReq
}
