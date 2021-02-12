package cdsclient

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (c *client) Features() ([]sdk.Feature, error) {
	res := []sdk.Feature{}
	if _, err := c.GetJSON(context.Background(), "/admin/features", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c *client) FeatureGet(name sdk.FeatureName) (sdk.Feature, error) {
	var res sdk.Feature
	if _, err := c.GetJSON(context.Background(), fmt.Sprintf("/admin/features/%s", name), &res); err != nil {
		return sdk.Feature{}, err
	}
	return res, nil
}

func (c *client) FeatureUpdate(f sdk.Feature) error {
	if _, err := c.PutJSON(context.Background(), fmt.Sprintf("/admin/features/%s", f.Name), f, nil); err != nil {
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

func (c *client) FeatureDelete(name sdk.FeatureName) error {
	var res sdk.Feature
	if _, err := c.DeleteJSON(context.Background(), fmt.Sprintf("/admin/features/%s", name), &res); err != nil {
		return err
	}
	return nil
}
