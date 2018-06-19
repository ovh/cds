package cdsclient

import (
	"fmt"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) ApplicationPipelinesAttach(projectKey string, appName string, pipelineNames ...string) error {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/attach", projectKey, appName)
	code, err := c.PostJSON(uri, pipelineNames, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("cds: api error (%d)", code)
	}

	return nil
}

func (c *client) ApplicationDoMigrationWorkflow(projectKey string, appName string, force, disablePrefix, withCurrentVersion, withRepositoryWebHook bool) error {
	params := url.Values{}
	params.Add("force", fmt.Sprintf("%t", force))
	params.Add("disablePrefix", fmt.Sprintf("%t", disablePrefix))
	params.Add("withCurrentVersion", fmt.Sprintf("%t", withCurrentVersion))
	params.Add("withRepositoryWebHook", fmt.Sprintf("%t", withRepositoryWebHook))
	uri := fmt.Sprintf("/project/%s/application/%s/workflow/migrate?%s", projectKey, appName, params.Encode())

	code, err := c.PostJSON(uri, nil, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("cds: api error (%d)", code)
	}

	return nil
}

func (c *client) ApplicationPipelineTriggerAdd(t *sdk.PipelineTrigger) error {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger", t.SrcProject.Key, t.SrcApplication.Name, t.SrcPipeline.Name)

	if t.SrcEnvironment.Name != "" {
		uri = fmt.Sprintf("%s?env=%s", uri, url.QueryEscape(t.SrcEnvironment.Name))
	}

	code, err := c.PostJSON(uri, t, nil)
	if err != nil {
		return err
	}

	if code >= 300 {
		return fmt.Errorf("cds: api error (%d)", code)
	}

	return nil
}

func (c *client) ApplicationPipelineTriggersGet(projectKey string, appName string, pipelineName string, envName string) ([]sdk.PipelineTrigger, error) {
	uri := fmt.Sprintf("/project/%s/application/%s/pipeline/%s/trigger/source", projectKey, appName, pipelineName)

	if envName != "" {
		uri = fmt.Sprintf("%s?env=%s", uri, url.QueryEscape(envName))
	}

	var triggers []sdk.PipelineTrigger
	code, err := c.GetJSON(uri, &triggers)
	if err != nil {
		return nil, err
	}

	if code >= 300 {
		return nil, fmt.Errorf("cds: api error (%d)", code)
	}

	return triggers, nil
}

func (c *client) AddHookOnRepositoriesManager(projectKey, appName, reposManager, repoFullname, pipelineName string) error {
	uri := fmt.Sprintf("/project/%s/application/%s/repositories_manager/%s/hook", projectKey, appName, reposManager)
	data := map[string]string{
		"repository_fullname": repoFullname,
		"pipeline_name":       pipelineName,
	}

	app := &sdk.Application{}
	code, err := c.PostJSON(uri, data, app)
	if err != nil {
		return err
	}
	if code >= 300 {
		return fmt.Errorf("HTTP %d", code)
	}
	return nil
}
