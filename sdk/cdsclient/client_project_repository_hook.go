package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectRepositoryHookAdd(ctx context.Context, projectKey string, r sdk.PostProjectRepositoryHook) (*sdk.HookAccessData, error) {
	path := fmt.Sprintf("/v2/project/%s/hook", projectKey)
	var hookData sdk.HookAccessData
	_, err := c.PostJSON(ctx, path, &r, &hookData)
	if err != nil {
		return nil, err
	}
	return &hookData, nil
}

func (c *client) ProjectRepositoryHookList(ctx context.Context, projectKey string) ([]sdk.ProjectRepositoryHook, error) {
	path := fmt.Sprintf("/v2/project/%s/hook", projectKey)
	var datas []sdk.ProjectRepositoryHook
	_, err := c.GetJSON(ctx, path, &datas)
	if err != nil {
		return nil, err
	}
	return datas, nil
}

func (c *client) ProjectRepositoryHookGet(ctx context.Context, projectKey string, uuid string) (*sdk.ProjectRepositoryHook, error) {
	path := fmt.Sprintf("/v2/project/%s/hook/%s", projectKey, uuid)
	var data sdk.ProjectRepositoryHook
	_, err := c.GetJSON(ctx, path, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *client) ProjectRepositoryHookDelete(ctx context.Context, projectKey string, uuid string) error {
	path := fmt.Sprintf("/v2/project/%s/hook/%s", projectKey, uuid)
	var data sdk.ProjectRepositoryHook
	_, err := c.DeleteJSON(ctx, path, &data)
	return err
}
