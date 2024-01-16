package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

// ProjectSecretList(ctx context.Context, projectKey string) ([]sdk.ProjectSecret, error)
func (c *client) ProjectSecretAdd(ctx context.Context, projectKey string, secret sdk.ProjectSecret) error {
	path := fmt.Sprintf("/v2/project/%s/secret", projectKey)
	_, err := c.PostJSON(ctx, path, &secret, nil)
	return err
}

func (c *client) ProjectSecretUpdate(ctx context.Context, projectKey string, secret sdk.ProjectSecret) error {
	path := fmt.Sprintf("/v2/project/%s/secret/%s", projectKey, secret.Name)
	_, err := c.PutJSON(ctx, path, &secret, nil)
	return err
}

func (c *client) ProjectSecretDelete(ctx context.Context, projectKey string, name string) error {
	path := fmt.Sprintf("/v2/project/%s/secret/%s", projectKey, name)
	_, err := c.DeleteJSON(ctx, path, nil)
	return err
}

func (c *client) ProjectSecretList(ctx context.Context, projectKey string) ([]sdk.ProjectSecret, error) {
	var secrets []sdk.ProjectSecret
	path := fmt.Sprintf("/v2/project/%s/secret", projectKey)
	_, err := c.GetJSON(ctx, path, &secrets)
	return secrets, err
}
