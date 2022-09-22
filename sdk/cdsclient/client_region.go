package cdsclient

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (c *client) RegionAdd(ctx context.Context, orga sdk.Region) error {
	if _, err := c.PostJSON(ctx, "/v2/region", &orga, nil); err != nil {
		return err
	}
	return nil
}

func (c *client) RegionGet(ctx context.Context, regionIdentifier string) (sdk.Region, error) {
	var reg sdk.Region
	if _, err := c.GetJSON(ctx, "/v2/region/"+regionIdentifier, &reg, nil); err != nil {
		return reg, err
	}
	return reg, nil
}

func (c *client) RegionList(ctx context.Context) ([]sdk.Region, error) {
	var regions []sdk.Region
	if _, err := c.GetJSON(ctx, "/v2/region", &regions, nil); err != nil {
		return nil, err
	}
	return regions, nil
}

func (c *client) RegionDelete(ctx context.Context, regionIdentifier string) error {
	if _, err := c.DeleteJSON(ctx, "/v2/region/"+regionIdentifier, nil, nil); err != nil {
		return err
	}
	return nil
}
