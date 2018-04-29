package cdsclient

import "github.com/ovh/cds/sdk"

func (c client) PluginsList() ([]sdk.GRPCPlugin, error) {
	res := []sdk.GRPCPlugin{}
	if _, err := c.GetJSON("/admin/plugin", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c client) PluginsGet(name string) (*sdk.GRPCPlugin, error) {
	path := "/admin/plugin/" + name
	res := sdk.GRPCPlugin{}
	if _, err := c.GetJSON(path, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c client) PluginAdd(p *sdk.GRPCPlugin) error {
	_, err := c.PostJSON("/admin/plugin", p, p)
	return err
}

func (c client) PluginUpdate(p *sdk.GRPCPlugin) error {
	_, err := c.PutJSON("/admin/plugin", p, p)
	return err
}

func (c client) PluginDelete(name string) error {
	path := "/admin/plugin/" + name
	_, err := c.DeleteJSON(path, nil)
	return err
}
