package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectRepositoryManagerList(projectKey string) ([]sdk.ProjectVCSServer, error) {
	path := fmt.Sprintf("/project/%s/repositories_manager", projectKey)
	var s []sdk.ProjectVCSServer
	if _, err := c.GetJSON(context.Background(), path, &s); err != nil {
		return s, err
	}
	return s, nil
}

func (c *client) ProjectRepositoryManagerDelete(projectKey string, repomanagerName string, force bool) error {
	path := fmt.Sprintf("/project/%s/repositories_manager/%s", projectKey, repomanagerName)
	if force {
		path += "?force=true"
	}
	var s sdk.ProjectVCSServer
	if _, err := c.DeleteJSON(context.Background(), path, &s); err != nil {
		return err
	}
	return nil
}
