package cdsclient

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (c *client) ProjectsList(opts ...RequestModifier) ([]sdk.Project, error) {
	p := []sdk.Project{}
	path := fmt.Sprintf("/project")
	if _, err := c.GetJSON(context.Background(), path, &p, opts...); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *client) ApplicationsList(projectKey string, opts ...RequestModifier) ([]sdk.Application, error) {
	apps := []sdk.Application{}
	if _, err := c.GetJSON(context.Background(), "/project/"+projectKey+"/applications", &apps, opts...); err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *client) ApplicationDeploymentStrategyUpdate(projectKey, applicationName, integrationName string, config sdk.IntegrationConfig) error {
	path := fmt.Sprintf("/project/%s/application/%s/deployment/config/%s", projectKey, applicationName, integrationName)
	if _, err := c.PostJSON(context.Background(), path, config, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) ApplicationMetadataUpdate(projectKey, applicationName, key, value string) error {
	path := fmt.Sprintf("/project/%s/application/%s/metadata/%s", projectKey, applicationName, url.PathEscape(key))
	if _, _, _, err := c.Request(context.Background(), "POST", path, strings.NewReader(value)); err != nil {
		return err
	}
	return nil
}

func (c *client) WorkflowsList(projectKey string) ([]sdk.Workflow, error) {
	ws := []sdk.Workflow{}
	if _, err := c.GetJSON(context.Background(), "/project/"+projectKey+"/workflows", &ws); err != nil {
		return nil, err
	}
	return ws, nil
}

func (c *client) WorkflowLoad(projectKey, workflowName string) (*sdk.Workflow, error) {
	url := fmt.Sprintf("/project/%s/workflows/%s?withDeepPipelines=true", projectKey, workflowName)
	w := &sdk.Workflow{}
	if _, err := c.GetJSON(context.Background(), url, &w); err != nil {
		return nil, err
	}
	return w, nil
}

func (c *client) ProjectIntegrationGet(projectKey string, integrationName string, clearPassword bool) (sdk.ProjectIntegration, error) {
	path := fmt.Sprintf("/project/%s/integrations/%s?clearPassword=%v", projectKey, integrationName, clearPassword)
	var pf sdk.ProjectIntegration
	if _, err := c.GetJSON(context.Background(), path, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}

func (c *client) ProjectIntegrationList(projectKey string) ([]sdk.ProjectIntegration, error) {
	path := fmt.Sprintf("/project/%s/integrations", projectKey)
	var pfs []sdk.ProjectIntegration
	if _, err := c.GetJSON(context.Background(), path, &pfs); err != nil {
		return pfs, err
	}
	return pfs, nil
}

func (c *client) ProjectIntegrationDelete(projectKey string, integrationName string) error {
	path := fmt.Sprintf("/project/%s/integrations/%s", projectKey, integrationName)
	var pf sdk.ProjectIntegration
	if _, err := c.DeleteJSON(context.Background(), path, &pf); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectIntegrationImport(projectKey string, content io.Reader, format string, force bool) (sdk.ProjectIntegration, error) {
	var pf sdk.ProjectIntegration

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

	//Get the integration to know if we have to POST or PUT
	oldPF, _ := c.ProjectIntegrationGet(projectKey, pf.Name, false)
	if oldPF.Name == "" {
		path := fmt.Sprintf("/project/%s/integrations", projectKey)
		if _, err := c.PostJSON(context.Background(), path, &pf, &pf); err != nil {
			return pf, err
		}
		return pf, nil
	}

	path := fmt.Sprintf("/project/%s/integrations/%s", projectKey, pf.Name)
	if _, err := c.PutJSON(context.Background(), path, &pf, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}
