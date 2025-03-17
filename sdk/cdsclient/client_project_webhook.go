package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) ProjectWebHookAdd(ctx context.Context, projectKey string, r sdk.PostProjectWebHook) (*sdk.HookAccessData, error) {
	path := fmt.Sprintf("/v2/project/%s/hook", projectKey)
	var hookData sdk.HookAccessData
	_, err := c.PostJSON(ctx, path, &r, &hookData)
	if err != nil {
		return nil, err
	}
	return &hookData, nil
}

func (c *client) ProjectWebHookList(ctx context.Context, projectKey string) ([]sdk.ProjectWebHook, error) {
	path := fmt.Sprintf("/v2/project/%s/hook", projectKey)
	var datas []sdk.ProjectWebHook
	_, err := c.GetJSON(ctx, path, &datas)
	if err != nil {
		return nil, err
	}
	return datas, nil
}

func (c *client) ProjectWebHookGet(ctx context.Context, projectKey string, uuid string) (*sdk.ProjectWebHook, error) {
	path := fmt.Sprintf("/v2/project/%s/hook/%s", projectKey, uuid)
	var data sdk.ProjectWebHook
	_, err := c.GetJSON(ctx, path, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *client) ProjectWebHookDelete(ctx context.Context, projectKey string, uuid string) error {
	path := fmt.Sprintf("/v2/project/%s/hook/%s", projectKey, uuid)
	var data sdk.ProjectWebHook
	_, err := c.DeleteJSON(ctx, path, &data)
	return err
}
