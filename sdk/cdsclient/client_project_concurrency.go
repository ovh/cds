package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectConcurrencyCreate(ctx context.Context, pKey string, concu *sdk.ProjectConcurrency) error {
	path := fmt.Sprintf("/v2/project/%s/concurrency", pKey)
	_, err := c.PostJSON(ctx, path, concu, concu)
	return err
}

func (c *client) ProjectConcurrencyDelete(ctx context.Context, pKey string, name string) error {
	path := fmt.Sprintf("/v2/project/%s/concurrency/%s", pKey, name)
	_, err := c.DeleteJSON(ctx, path, nil)
	return err
}

func (c *client) ProjectConcurrencyList(ctx context.Context, pKey string) ([]sdk.ProjectConcurrency, error) {
	var pcs []sdk.ProjectConcurrency
	path := fmt.Sprintf("/v2/project/%s/concurrency", pKey)
	_, err := c.GetJSON(ctx, path, &pcs)
	return pcs, err
}

func (c *client) ProjectConcurrencyGet(ctx context.Context, pKey string, name string) (*sdk.ProjectConcurrency, error) {
	var pc sdk.ProjectConcurrency
	path := fmt.Sprintf("/v2/project/%s/concurrency/%s", pKey, name)
	_, err := c.GetJSON(ctx, path, &pc)
	return &pc, err
}

func (c *client) ProjectConcurrencyUpdate(ctx context.Context, pKey string, concu *sdk.ProjectConcurrency) error {
	path := fmt.Sprintf("/v2/project/%s/concurrency/%s", pKey, concu.Name)
	_, err := c.PutJSON(ctx, path, concu, concu)
	return err
}

func (c *client) ProjectConcurrencyListRuns(ctx context.Context, pKey string, name string) ([]sdk.ProjectConcurrencyRunObject, error) {
	var pcrs []sdk.ProjectConcurrencyRunObject
	path := fmt.Sprintf("/v2/project/%s/concurrency/%s/runs", pKey, name)
	_, err := c.GetJSON(ctx, path, &pcrs)
	return pcrs, err
}
