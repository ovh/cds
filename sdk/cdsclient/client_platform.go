package cdsclient

import (
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) PlatformModelList() ([]sdk.PlatformModel, error) {
	models := []sdk.PlatformModel{}
	if _, err := c.GetJSON("/platform/models", &models); err != nil {
		return nil, err
	}
	return models, nil
}

func (c *client) PlatformModelGet(name string) (sdk.PlatformModel, error) {
	var model sdk.PlatformModel
	if _, err := c.GetJSON("/platform/models/"+url.QueryEscape(name), &model); err != nil {
		return model, err
	}
	return model, nil
}

func (c *client) PlatformModelAdd(m *sdk.PlatformModel) error {
	if _, err := c.PostJSON("/platform/models", m, m); err != nil {
		return err
	}
	return nil
}

func (c *client) PlatformModelUpdate(m *sdk.PlatformModel) error {
	if _, err := c.PutJSON("/platform/models/"+m.Name, m, m); err != nil {
		return err
	}
	return nil
}
