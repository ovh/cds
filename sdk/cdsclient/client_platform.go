package cdsclient

import (
	"context"
	"net/url"

	"github.com/ovh/cds/sdk"
)

func (c *client) IntegrationModelList() ([]sdk.IntegrationModel, error) {
	models := []sdk.IntegrationModel{}
	if _, err := c.GetJSON(context.Background(), "/integration/models", &models); err != nil {
		return nil, err
	}
	return models, nil
}

func (c *client) IntegrationModelGet(name string) (sdk.IntegrationModel, error) {
	var model sdk.IntegrationModel
	if _, err := c.GetJSON(context.Background(), "/integration/models/"+url.QueryEscape(name), &model); err != nil {
		return model, err
	}
	return model, nil
}

func (c *client) IntegrationModelAdd(m *sdk.IntegrationModel) error {
	if _, err := c.PostJSON(context.Background(), "/integration/models", m, m); err != nil {
		return err
	}
	return nil
}

func (c *client) IntegrationModelUpdate(m *sdk.IntegrationModel) error {
	if _, err := c.PutJSON(context.Background(), "/integration/models/"+m.Name, m, m); err != nil {
		return err
	}
	return nil
}

func (c *client) IntegrationModelDelete(name string) error {
	if _, err := c.DeleteJSON(context.Background(), "/integration/models/"+name, nil, nil); err != nil {
		return err
	}
	return nil
}
