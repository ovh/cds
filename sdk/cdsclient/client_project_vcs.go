package cdsclient

import (
	"context"
	"fmt"
	"io"

	"github.com/ghodss/yaml"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVCSGet(ctx context.Context, projectKey string, vcsName string) (sdk.VCSProject, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, vcsName)
	var pf sdk.VCSProject
	if _, err := c.GetJSON(ctx, path, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}

func (c *client) ProjectVCSList(ctx context.Context, projectKey string) ([]sdk.VCSProject, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs", projectKey)
	var pfs []sdk.VCSProject
	if _, err := c.GetJSON(ctx, path, &pfs); err != nil {
		return pfs, err
	}
	return pfs, nil
}

func (c *client) ProjectVCSDelete(ctx context.Context, projectKey string, vcsName string) error {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, vcsName)
	var pf sdk.VCSProject
	if _, err := c.DeleteJSON(ctx, path, &pf); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectVCSImport(ctx context.Context, projectKey string, content io.Reader, mods ...RequestModifier) (sdk.VCSProject, error) {
	var pf sdk.VCSProject

	body, err := io.ReadAll(content)
	if err != nil {
		return pf, err
	}

	if err := yaml.Unmarshal(body, &pf); err != nil {
		return pf, err
	}

	oldvcs, _ := c.ProjectVCSGet(ctx, projectKey, pf.Name)
	if oldvcs.Name == "" {
		path := fmt.Sprintf("/v2/project/%s/vcs", projectKey)
		if _, err := c.PostJSON(ctx, path, &pf, &pf, mods...); err != nil {
			return pf, err
		}
		return pf, nil
	}

	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, pf.Name)
	if _, err := c.PutJSON(ctx, path, &pf, &pf, mods...); err != nil {
		return pf, err
	}
	return pf, nil
}
