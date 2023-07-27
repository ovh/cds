package cdsclient

import (
	"context"
	"github.com/ovh/cds/sdk"
)

func (c client) PluginsGet(name string) (*sdk.GRPCPlugin, error) {
	path := "/v2/plugin/" + name
	res := sdk.GRPCPlugin{}
	if _, err := c.GetJSON(context.Background(), path, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c client) PluginImport(p *sdk.GRPCPlugin, mods ...RequestModifier) error {
	_, err := c.PostJSON(context.Background(), "/v2/plugin", p, p, mods...)
	return err
}
