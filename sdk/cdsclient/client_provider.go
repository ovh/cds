package cdsclient

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
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

func (c *client) ProjectPlatformGet(projectKey string, platformName string, clearPassword bool) (sdk.ProjectPlatform, error) {
	path := fmt.Sprintf("/project/%s/platforms/%s?clearPassword=%v", projectKey, platformName, clearPassword)
	var pf sdk.ProjectPlatform
	if _, err := c.GetJSON(path, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}

func (c *client) ProjectPlatformList(projectKey string) ([]sdk.ProjectPlatform, error) {
	path := fmt.Sprintf("/project/%s/platforms", projectKey)
	var pfs []sdk.ProjectPlatform
	if _, err := c.GetJSON(path, &pfs); err != nil {
		return pfs, err
	}
	return pfs, nil
}

func (c *client) ProjectPlatformDelete(projectKey string, platformName string) error {
	path := fmt.Sprintf("/project/%s/platforms/%s", projectKey, platformName)
	var pf sdk.ProjectPlatform
	if _, err := c.DeleteJSON(path, &pf); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectPlatformImport(projectKey string, content io.Reader, format string, force bool) (sdk.ProjectPlatform, error) {
	var pf sdk.ProjectPlatform

	body, err := ioutil.ReadAll(content)
	if err != nil {
		return pf, err
	}

	f, err := exportentities.GetFormat(format)
	if err != nil {
		return pf, err
	}

	if err := exportentities.Unmarshal(body, f, &pf); err != nil {
		return pf, err
	}

	//Get the platform to know if we have to POST or PUT
	oldPF, _ := c.ProjectPlatformGet(projectKey, pf.Name, false)
	if oldPF.Name == "" {
		path := fmt.Sprintf("/project/%s/platforms", projectKey)
		if _, err := c.PostJSON(path, &pf, &pf); err != nil {
			return pf, err
		}
		return pf, nil
	}

	path := fmt.Sprintf("/project/%s/platforms/%s", projectKey, pf.Name)
	if _, err := c.PutJSON(path, &pf, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}
