package cdsclient

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectsList(opts ...RequestModifier) ([]sdk.Project, error) {
	p := []sdk.Project{}
	path := fmt.Sprintf("/project")
	if _, err := c.GetJSON(path, &p, opts...); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ApplicationsList(projectKey string, opts ...RequestModifier) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	if _, err := c.GetJSON("/project/"+projectKey+"/applications", &apps, opts...); err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *client) ApplicationDeploymentStrategyUpdate(projectKey, applicationName, platformName string, config sdk.PlatformConfig) error {
	path := fmt.Sprintf("/project/%s/application/%s/deployment/config/%s", projectKey, applicationName, platformName)
	if _, err := c.PostJSON(path, config, nil); err != nil {
		return err
	}
	return nil
}
