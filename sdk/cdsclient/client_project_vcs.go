package cdsclient

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/ghodss/yaml"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectVCSGet(ctx context.Context, projectKey string, vcsName string) (sdk.VCSProject, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, url.PathEscape(vcsName))
	var vcsProject sdk.VCSProject
	if _, err := c.GetJSON(ctx, path, &vcsProject); err != nil {
		return vcsProject, err
	}
	return vcsProject, nil
}

func (c *client) ProjectVCSList(ctx context.Context, projectKey string) ([]sdk.VCSProject, error) {
	path := fmt.Sprintf("/v2/project/%s/vcs", projectKey)
	var vcsProjects []sdk.VCSProject
	if _, err := c.GetJSON(ctx, path, &vcsProjects); err != nil {
		return vcsProjects, err
	}
	return vcsProjects, nil
}

func (c *client) ProjectVCSDelete(ctx context.Context, projectKey string, vcsName string) error {
	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, url.PathEscape(vcsName))
	var vcsProject sdk.VCSProject
	if _, err := c.DeleteJSON(ctx, path, &vcsProject); err != nil {
		return err
	}
	return nil
}

func (c *client) ProjectVCSImport(ctx context.Context, projectKey string, content io.Reader, mods ...RequestModifier) (sdk.VCSProject, error) {
	var vcsProject sdk.VCSProject

	body, err := io.ReadAll(content)
	if err != nil {
		return vcsProject, err
	}

	if err := yaml.Unmarshal(body, &vcsProject); err != nil {
		return vcsProject, err
	}

	oldvcs, _ := c.ProjectVCSGet(ctx, projectKey, vcsProject.Name)
	if oldvcs.Name == "" {
		path := fmt.Sprintf("/v2/project/%s/vcs", projectKey)
		if _, err := c.PostJSON(ctx, path, &vcsProject, &vcsProject, mods...); err != nil {
			return vcsProject, err
		}
		return vcsProject, nil
	}

	path := fmt.Sprintf("/v2/project/%s/vcs/%s", projectKey, url.PathEscape(vcsProject.Name))
	if _, err := c.PutJSON(ctx, path, &vcsProject, &vcsProject, mods...); err != nil {
		return vcsProject, err
	}
	return vcsProject, nil
}
