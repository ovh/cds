package cdsclient

import (
	"context"
	"fmt"
	"io"

	"github.com/ghodss/yaml"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVCSGet(projectKey string, vcsName string) (sdk.ProjectVCSServer, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, vcsName)
	var pf sdk.ProjectVCSServer
	if _, err := c.GetJSON(context.Background(), path, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}

func (c *client) ProjectVCSList(projectKey string) ([]sdk.ProjectVCSServer, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs", projectKey)
	var pfs []sdk.ProjectVCSServer
	if _, err := c.GetJSON(context.Background(), path, &pfs); err != nil {
		return pfs, err
	}
	return pfs, nil
}

func (c *client) ProjectVCSDelete(projectKey string, vcsName string) error {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, vcsName)
	var pf sdk.ProjectVCSServer
	if _, err := c.DeleteJSON(context.Background(), path, &pf); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectVCSImport(projectKey string, content io.Reader, mods ...RequestModifier) (sdk.ProjectVCSServer, error) {
	var pf sdk.ProjectVCSServer

	body, err := io.ReadAll(content)
	if err != nil {
		return pf, err
	}

	if err := yaml.Unmarshal(body, &pf); err != nil {
		return pf, err
	}

	oldvcs, _ := c.ProjectVCSGet(projectKey, pf.Name)
	if oldvcs.Name == "" {
		path := fmt.Sprintf("/v2/project/%s/vcs", projectKey)
		if _, err := c.PostJSON(context.Background(), path, &pf, &pf, mods...); err != nil {
			return pf, err
		}
		return pf, nil
	}

	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, pf.Name)
	if _, err := c.PutJSON(context.Background(), path, &pf, &pf, mods...); err != nil {
		return pf, err
	}
	return pf, nil
}
