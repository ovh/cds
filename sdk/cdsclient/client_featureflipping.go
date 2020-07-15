package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) Features() ([]sdk.Feature, error) {
	res := []sdk.Feature{}
	if _, err := c.GetJSON(context.Background(), "/admin/features", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) FeatureGet(name string) (sdk.Feature, error) {
	var res sdk.Feature
	if _, err := c.GetJSON(context.Background(), "/admin/features/"+name, &res); err != nil {
		return sdk.Feature{}, err
	}
	return res, nil
}

func (c *client) FeatureUpdate(f sdk.Feature) error {
	if _, err := c.PutJSON(context.Background(), "/admin/features/"+f.Name, f, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) FeatureCreate(f sdk.Feature) error {
	if _, err := c.PostJSON(context.Background(), "/admin/features", &f, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) FeatureDelete(name string) error {
	var res sdk.Feature
	if _, err := c.DeleteJSON(context.Background(), "/admin/features/"+name, &res); err != nil {
		return err
	}
	return nil
}
